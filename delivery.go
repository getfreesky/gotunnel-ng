package main

import (
	"encoding/binary"
	"io"
	"math/rand"
	"net"
	"reflect"
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
}

func NewOutgoingDelivery(hostPort string) (*Delivery, error) {
	conn, err := net.Dial("tcp", hostPort)
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
	delivery.Recv(delivery.SendQueue, delivery.send)
	return delivery, nil
}

func (self *Delivery) onConnBroken() {
	if self.hostPort != "" { // outgoing delivery
		conn, err := net.Dial("tcp", self.hostPort)
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
	}
}

func (self *Delivery) Send(data []byte) {
	self.SendQueue <- data
}
