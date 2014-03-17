package main

import (
	"bytes"
	"encoding/binary"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"sync"
)

const (
	CONNECT = uint8(0)
	DATA    = uint8(1)
	CLOSE   = uint8(2)
)

type Session struct {
	*Closer
	*Logger
	Id             int64
	DeliverySource int64
	Conn           net.Conn
	connReady      chan bool
	HostPort       string
	OutgoingPacket chan SessionPacketInfo
	localClosed    bool
	remoteClosed   bool
	connClosed     bool
	closeConnOnce  sync.Once
}

type SessionPacketInfo struct {
	Session *Session
	Data    []byte
}

func NewSession(id, source int64, conn net.Conn, hostPort string) *Session {
	session := &Session{
		Closer:         new(Closer),
		Logger:         new(Logger),
		Id:             id,
		DeliverySource: source,
		Conn:           conn,
		connReady:      make(chan bool),
		HostPort:       hostPort,
		OutgoingPacket: make(chan SessionPacketInfo, 1024),
	}
	if id < 0 { // local session
		session.Id = rand.Int63()
		go session.Connect(hostPort)
		go session.startConnReader()
		close(session.connReady)
	}
	return session
}

func (self *Session) startConnReader() {
	for {
		buf := make([]byte, 4096)
		n, err := self.Conn.Read(buf)
		if n > 0 {
			self.SendData(buf[:n])
			self.Log("%d bytes from connection", n)
		}
		if err != nil {
			if self.connClosed {
				break
			} else {
				self.closeConnOnce.Do(func() {
					self.connClosed = true
					if self.Conn != nil {
						self.Conn.Close()
					}
				})
				self.SignalClose()
				self.localClosed = true
				if self.remoteClosed {
					self.Close()
				}
				break
			}
		}
	}
}

func (self *Session) Connect(hostPort string) {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, self.Id)
	binary.Write(buf, binary.BigEndian, CONNECT)
	buf.Write([]byte(hostPort))
	self.OutgoingPacket <- SessionPacketInfo{
		Session: self,
		Data:    buf.Bytes(),
	}
}

func (self *Session) SendData(data []byte) {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, self.Id)
	binary.Write(buf, binary.BigEndian, DATA)
	buf.Write(data)
	self.OutgoingPacket <- SessionPacketInfo{
		Session: self,
		Data:    buf.Bytes(),
	}
}

func (self *Session) SignalClose() {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, self.Id)
	binary.Write(buf, binary.BigEndian, CLOSE)
	self.OutgoingPacket <- SessionPacketInfo{
		Session: self,
		Data:    buf.Bytes(),
	}
}

func (self *Session) handlePacket(data []byte) {
	var packetType uint8
	reader := bytes.NewReader(data)
	binary.Read(reader, binary.BigEndian, &packetType)
	if packetType == CONNECT {
		self.handleConnect(reader)
	} else if packetType == DATA {
		self.handleData(reader)
	} else if packetType == CLOSE {
		self.handleClose()
	} else {
		log.Fatalf("unknown packet type %d\n", packetType)
	}
}

func (self *Session) handleConnect(reader io.Reader) {
	hostPortBs, err := ioutil.ReadAll(reader)
	if err != nil {
		log.Fatal(err)
	}
	hostPort := string(hostPortBs)
	self.Log("connect %s", hostPort)
	self.HostPort = hostPort
	conn, err := net.Dial("tcp", hostPort)
	if err != nil {
		self.SignalClose()
		self.localClosed = true
		if self.remoteClosed {
			self.Close()
		}
		return
	}
	self.Conn = conn
	close(self.connReady)
	go self.startConnReader()
}

func (self *Session) handleData(reader io.Reader) {
	data, err := ioutil.ReadAll(reader)
	if err != nil {
		log.Fatal(err)
	}
	self.Log("%d bytes from delivery", len(data))
	<-self.connReady
	self.Conn.Write(data)
	self.Log("%d bytes wrote to conn", len(data))
}

func (self *Session) handleClose() {
	self.closeConnOnce.Do(func() {
		self.connClosed = true
		if self.Conn != nil {
			self.Conn.Close()
		}
	})
	self.SignalClose()
	self.remoteClosed = true
	if self.localClosed {
		self.Close()
	}
}
