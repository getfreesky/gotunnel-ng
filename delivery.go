package main

import (
	"encoding/binary"
	"io"
	"math/rand"
	"net"
	"reflect"
	"time"
)

type Delivery struct {
	*Actor
	hostPort       string
	conn           net.Conn
	Source         int64
	IncomingPacket chan []byte
	connReady      chan bool
	SendQueue      chan []byte
	reconnectTimes int
	FlowControl    chan int
	Load           float64
}

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
	return NewDelivery(hostPort, conn, source)
}

func NewIncomingDelivery(conn net.Conn, source int64) (*Delivery, error) {
	return NewDelivery("", conn, source)
}

func NewDelivery(hostPort string, conn net.Conn, source int64) (*Delivery, error) {
	delivery := &Delivery{
		Actor:          NewActor(),
		hostPort:       hostPort,
		conn:           conn,
		Source:         source,
		IncomingPacket: make(chan []byte),
		connReady:      make(chan bool),
		SendQueue:      make(chan []byte),
	}
	delivery.OnClose(func() {
		conn.Close()
	})
	delivery.OnSignal("connBroken", delivery.onConnBroken)
	go delivery.startConnReader()
	go delivery.startFlowControl()
	delivery.Recv(delivery.SendQueue, delivery.send)
	return delivery, nil
}

func (self *Delivery) onConnBroken() {
	if self.hostPort != "" { // outgoing delivery
		conn, err := net.DialTimeout("tcp", self.hostPort, time.Second*30)
		if err != nil {
			self.Signal("connBroken")
			return
		}
		err = binary.Write(conn, binary.BigEndian, self.Source)
		if err != nil {
			self.Signal("connBroken")
			return
		}
		self.conn = conn
	} else { // incoming delivery
		<-self.connReady
		self.connReady = make(chan bool)
	}
	go self.startConnReader()
	self.reconnectTimes++
	self.Signal("notify:reconnected")
}

func (self *Delivery) send(v reflect.Value) {
	data := v.Interface().([]byte)
	err := binary.Write(self.conn, binary.BigEndian, uint16(len(data)))
	if err != nil {
		self.Signal("connBroken")
		return
	}
	n, err := self.conn.Write(data)
	if err != nil || n != len(data) {
		self.Signal("connBroken")
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
		err = binary.Read(self.conn, binary.BigEndian, &length)
		if err != nil {
			if self.IsClosed { // close by Close()
				break
			} else { // conn broken
				self.Signal("connBroken")
				break
			}
		}
		data := make([]byte, length)
		n, err = io.ReadFull(self.conn, data)
		if err != nil || n != int(length) { // conn broken
			self.Signal("connBroken")
			break
		}
		self.IncomingPacket <- data
		self.Signal("incoming", data)
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
				if self.IsClosed {
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
				if self.IsClosed {
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
