package main

import (
	"encoding/binary"
	"io"
	"net"
)

type Delivery struct {
	conn           net.Conn
	source         int64
	IncomingPacket chan []byte
	IsClosed       chan bool
}

func NewDelivery(conn net.Conn) *Delivery {
	return &Delivery{
		conn:           conn,
		IncomingPacket: make(chan []byte),
		IsClosed:       make(chan bool),
	}
}

func (self *Delivery) Run() {
	var err error
	err = binary.Read(self.conn, binary.BigEndian, &self.source)
	if err != nil {
		self.Close()
		return
	}
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
	self.conn.Close()
	close(self.IsClosed)
}
