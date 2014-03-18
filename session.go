package main

import (
	"bytes"
	"encoding/binary"
	"math/rand"
	"net"
)

const (
	CONNECT = uint8(0)
	DATA    = uint8(1)
)

type Session struct {
	*Actor
	id        int64
	delivery  *Delivery
	hostPort  string
	conn      net.Conn
	connReady chan bool
}

func NewOutgoingSession(delivery *Delivery, hostPort string, conn net.Conn) (*Session, error) {
	session := &Session{
		Actor:     NewActor(),
		id:        rand.Int63(),
		delivery:  delivery,
		hostPort:  hostPort,
		conn:      conn,
		connReady: make(chan bool),
	}
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
	}
	go func() {
		conn, err := net.Dial("tcp", hostPort)
		if err != nil {
			session.Signal("dialError")
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
	<-self.connReady
	self.conn.Write(data)
}
