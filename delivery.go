package main

import (
	"encoding/binary"
	"io"
	"math/rand"
	"net"
)

type Delivery struct {
	*Closer
	conn           net.Conn
	Source         int64
	IncomingPacket chan []byte
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
		Closer:         new(Closer),
		conn:           conn,
		IncomingPacket: make(chan []byte, 1024),
		Source:         source,
	}
	delivery.OnClose(func() {
		delivery.conn.Close()
	})
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

func (self *Delivery) Send(data []byte) {
	self.conn.Write(data)
}
