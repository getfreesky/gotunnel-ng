package main

import (
	"bytes"
	"encoding/binary"
	"math/rand"
	"net"
	"reflect"
	"time"
)

const (
	CONNECT = uint8(0)
	DATA    = uint8(1)
)

type Session struct {
	*Actor
	id             int64
	delivery       *Delivery
	hostPort       string
	conn           net.Conn
	connReady      chan bool
	connWriteQueue chan []byte
}

func NewOutgoingSession(delivery *Delivery, hostPort string, conn net.Conn) (*Session, error) {
	session := &Session{
		Actor:          NewActor(),
		id:             rand.Int63(),
		delivery:       delivery,
		hostPort:       hostPort,
		conn:           conn,
		connReady:      make(chan bool),
		connWriteQueue: make(chan []byte),
	}
	session.Recv(session.connWriteQueue, session.writeConn)
	session.SendConnect()
	go session.startConnReader()
	close(session.connReady)
	return session, nil
}

func NewIncomingSession(id int64, delivery *Delivery, hostPort string) (*Session, error) {
	session := &Session{
		Actor:     NewActor(),
		id:        id,
		delivery:  delivery,
		hostPort:  hostPort,
		connReady: make(chan bool),
		connWriteQueue: make(chan []byte),
	}
	session.Recv(session.connWriteQueue, session.writeConn)
	go func() {
		conn, err := net.DialTimeout("tcp", hostPort, time.Second*5)
		if err != nil {
			session.Signal("dialError")
			close(session.connReady)
			return
		}
		session.conn = conn
		go session.startConnReader()
		close(session.connReady)
	}()
	return session, nil
}

func (self *Session) startConnReader() {
	for {
		buf := make([]byte, 1024)
		n, err := self.conn.Read(buf)
		if n > 0 {
			self.SendData(buf[:n])
		}
		if err != nil {
			self.Signal("connClosed")
			break
		}
	}
}

func (self *Session) newPacket(t uint8) *bytes.Buffer {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, self.id)
	binary.Write(buf, binary.BigEndian, t)
	return buf
}

func (self *Session) SendConnect() {
	buf := self.newPacket(CONNECT)
	buf.Write([]byte(self.hostPort))
	self.delivery.Send(buf.Bytes())
}

func (self *Session) SendData(data []byte) {
	buf := self.newPacket(DATA)
	buf.Write(data)
	self.delivery.Send(buf.Bytes())
}

func (self *Session) HandleData(data []byte) {
	self.connWriteQueue <- data
}

func (self *Session) writeConn(v reflect.Value) {
	<-self.connReady
	if self.conn == nil {
		return
	}
	data := v.Interface().([]byte)
	self.conn.Write(data)
}
