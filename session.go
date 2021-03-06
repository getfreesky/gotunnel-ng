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
	CLOSE   = uint8(2)
)

type Session struct {
	*Actor
	id           int64
	delivery     *Delivery
	hostPort     string
	conn         net.Conn
	connReady    chan bool
	localClosed  bool
	remoteClosed bool
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
	session.OnSignal("data", session.onData)
	session.OnSignal("close", session.onClose)
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
	session.OnSignal("data", session.onData)
	session.OnSignal("close", session.onClose)
	go func() {
		conn, err := net.DialTimeout("tcp", hostPort, time.Second*30)
		if err != nil {
			session.SendClose()
			session.localClosed = true
			if session.remoteClosed {
				session.Close()
			}
			session.Recv(time.After(time.Minute*3), func(v reflect.Value) {
				session.Close()
			})
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
		buf := make([]byte, <-self.delivery.FlowControl)
		n, err := self.conn.Read(buf)
		if n > 0 {
			self.SendData(buf[:n])
		}
		if err != nil {
			self.SendClose()
			self.localClosed = true
			if self.remoteClosed {
				self.Close()
			}
			self.Recv(time.After(time.Minute*3), func(v reflect.Value) {
				self.Close()
			})
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

func (self *Session) SendClose() {
	buf := self.newPacket(CLOSE)
	self.delivery.Send(buf.Bytes())
}

func (self *Session) onData(data []byte) {
	<-self.connReady
	if self.conn == nil {
		return
	}
	self.conn.Write(data)
}

func (self *Session) onClose() {
	if self.conn != nil {
		self.conn.Close()
	}
	self.remoteClosed = true
	if self.localClosed {
		self.Close()
	}
	self.Recv(time.After(time.Minute*3), func(v reflect.Value) {
		self.Close()
	})
}
