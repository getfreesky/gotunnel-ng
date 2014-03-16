package main

import (
	"encoding/binary"
	"io"
	"math/rand"
	"net"
	"sync"
)

type Delivery struct {
	conn           net.Conn
	Source         int64
	IncomingPacket chan []byte
	CloseChan      chan bool
	closeOnce      sync.Once
}

func NewDelivery(conn net.Conn, isIncoming bool) (*Delivery, error) {
	var source int64
	if isIncoming { // read Source id
		err := binary.Read(conn, binary.BigEndian, &source)
		if err != nil {
			conn.Close()
			return nil, err
		}
	} else { // write Source id
		source = rand.Int63()
		err := binary.Write(conn, binary.BigEndian, source)
		if err != nil {
			conn.Close()
			return nil, err
		}
	}
	delivery := &Delivery{
		conn:           conn,
		IncomingPacket: make(chan []byte),
		CloseChan:      make(chan bool),
		Source:         source,
	}
	go delivery.Run()
	return delivery, nil
}

func (self *Delivery) Run() {
	var err error
	var length int16
	for {
		err = binary.Read(self.conn, binary.BigEndian, &length)
		if err != nil {
			self.Close()
			break
		}
		data := make([]byte, length)
		_, err = io.ReadFull(self.conn, data)
		if err != nil {
			self.Close()
			break
		}
		self.IncomingPacket <- data
	}
}

func (self *Delivery) Close() {
	self.closeOnce.Do(func() {
		self.conn.Close()
		close(self.CloseChan)
	})
}
