package main

import (
	"encoding/binary"
	"io"
	"log"
	"math/rand"
	"net"
	"time"
)

type Delivery struct {
	Type           int
	State          int
	hostPort       string
	conn           net.Conn
	Source         int64
	IncomingPacket chan []byte
	connReady      chan bool
	SendQueue      chan []byte
	reconnectTimes int
	FlowControl    chan int
	Load           float64
	CloseCallbacks []func()
}

const (
	INCOMING = iota
	OUTGOING
	NORMAL
	BROKEN
	CLOSED

	HeartBeatInterval = time.Second * 8
)

func NewOutgoingDelivery(hostPort string) (*Delivery, error) {
	conn, err := net.DialTimeout("tcp", hostPort, time.Second*30)
	if err != nil {
		return nil, err
	}
	source := rand.Int63()
	err = binary.Write(conn, binary.BigEndian, source)
	if err != nil {
		conn.Close()
		return nil, err
	}
	return NewDelivery(OUTGOING, hostPort, conn, source)
}

func NewIncomingDelivery(conn net.Conn, source int64) (*Delivery, error) {
	return NewDelivery(INCOMING, "", conn, source)
}

func NewDelivery(t int, hostPort string, conn net.Conn, source int64) (*Delivery, error) {
	delivery := &Delivery{
		Type:           t,
		State:          NORMAL,
		hostPort:       hostPort,
		conn:           conn,
		Source:         source,
		IncomingPacket: make(chan []byte),
		connReady:      make(chan bool),
		SendQueue:      make(chan []byte),
	}
	go delivery.startConnReader()
	go delivery.start()
	go delivery.startFlowControl()
	return delivery, nil
}

func (self *Delivery) start() {
	// main loop
	heartBeatTimer := time.NewTimer(HeartBeatInterval)
	for {
		switch self.State {
		case NORMAL:
			heartBeatTimer.Reset(HeartBeatInterval)
			select {
			case data := <-self.SendQueue:
				self.send(data)
			case <-heartBeatTimer.C: // send heartbeat
				self.heartbeat()
			}
		case BROKEN:
			self.onConnBroken()
		}
	}
	// cleaning
	self.State = CLOSED
	self.conn.Close()
	for _, cb := range self.CloseCallbacks {
		cb()
	}
}

func (self *Delivery) onConnBroken() {
	if self.Type == OUTGOING {
		print("reconnect")
		conn, err := net.DialTimeout("tcp", self.hostPort, time.Second*30)
		if err != nil {
			self.State = BROKEN
			return
		}
		err = binary.Write(conn, binary.BigEndian, self.Source)
		if err != nil {
			self.State = BROKEN
			return
		}
		self.conn = conn
		print("reconnect success")
	} else { // incoming delivery
		<-self.connReady
		self.connReady = make(chan bool)
	}
	go self.startConnReader()
	self.reconnectTimes++
	self.State = NORMAL
}

func (self *Delivery) heartbeat() {
	err := binary.Write(self.conn, binary.BigEndian, uint16(0))
	if err != nil {
		self.State = BROKEN
		return
	}
}

func (self *Delivery) send(data []byte) {
	err := binary.Write(self.conn, binary.BigEndian, uint16(len(data)))
	if err != nil {
		self.State = BROKEN
		return
	}
	for i, _ := range data {
		data[i] ^= 0xDE
	}
	n, err := self.conn.Write(data)
	if err != nil || n != len(data) {
		self.State = BROKEN
	}
}

func (self *Delivery) Send(data []byte) {
	self.SendQueue <- data
}

func (self *Delivery) startConnReader() {
	var err error
	var length uint16
	var n int
	for {
		err = self.conn.SetDeadline(time.Now().Add(HeartBeatInterval * 2))
		if err != nil {
			log.Fatal(err)
		}
		err = binary.Read(self.conn, binary.BigEndian, &length)
		if err != nil {
			if self.State == CLOSED {
				break
			} else { // conn broken
				self.State = BROKEN
				break
			}
		}
		if length == 0 { // heartbeat
			continue
		}
		data := make([]byte, length)
		n, err = io.ReadFull(self.conn, data)
		if err != nil || n != int(length) { // conn broken
			self.State = BROKEN
			break
		}
		for i, _ := range data {
			data[i] ^= 0xDE
		}
		self.IncomingPacket <- data
	}
}

func (self *Delivery) startFlowControl() {
	size := 1500
	interval := time.Millisecond * 100
	flowPerSecond := 1024 * 1024 * 2
	n := (flowPerSecond / size) / int(time.Second/interval)
	self.FlowControl = make(chan int)
	ticker := time.NewTicker(interval)
	var buf []int
	for {
		if len(buf) > 0 {
			select {
			case <-ticker.C:
				if self.State == CLOSED {
					return
				}
				for i := 0; i < n-len(buf); i++ {
					buf = append(buf, size)
				}
				self.Load = float64(n-len(buf)) / float64(n)
			case self.FlowControl <- buf[len(buf)-1]:
				buf = buf[:len(buf)-1]
			}
		} else {
			select {
			case <-ticker.C:
				if self.State == CLOSED {
					return
				}
				for i := 0; i < n; i++ {
					buf = append(buf, size)
				}
				self.Load = 1
			}
		}
	}
}
