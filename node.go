package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"reflect"
)

type Node struct {
	*Actor
	*Logger
	Deliveries []*Delivery
	Sessions   map[int64]*Session

	logPrefix string
}

func NewNode() *Node {
	// node
	node := &Node{
		Actor:    NewActor(),
		Logger:   new(Logger),
		Sessions: make(map[int64]*Session),
	}
	// deliveries
	node.OnClose(func() {
		for _, delivery := range node.Deliveries {
			delivery.Close()
		}
	})

	return node
}

func (self *Node) AddDelivery(delivery *Delivery) {
	self.Recv(delivery.IncomingPacket, func(v reflect.Value) { // incoming packet
		data := v.Interface().([]byte)
		var sessionId int64
		binary.Read(bytes.NewReader(data[:8]), binary.BigEndian, &sessionId)
		session := self.Sessions[sessionId]
		if session == nil {
			session = NewSession(sessionId, delivery.Source, nil, "")
			self.AddSession(session)
		}
		go session.handlePacket(data[8:])
	})
	delivery.OnClose(func() {
		self.StopRecv(delivery.IncomingPacket)
		var n int
		for i, d := range self.Deliveries {
			if d == delivery {
				n = i
				break
			}
		}
		self.Deliveries = append(self.Deliveries[:n], self.Deliveries[n+1:]...)
	})
	self.Deliveries = append(self.Deliveries, delivery)
}

func (self *Node) AddSession(session *Session) {
	self.Recv(session.OutgoingPacket, func(v reflect.Value) { // deliver packet
		info := v.Interface().(SessionPacketInfo)
		for _, delivery := range self.Deliveries {
			if delivery.Source == info.Session.DeliverySource {
				buf := new(bytes.Buffer)
				binary.Write(buf, binary.BigEndian, uint16(len(info.Data)))
				delivery.Send(buf.Bytes())
				delivery.Send(info.Data)
				break
			}
		}
	})
	session.OnClose(func() {
		self.StopRecv(session.OutgoingPacket)
		delete(self.Sessions, session.Id)
	})
	self.Sessions[session.Id] = session
	session.LogPrefix = self.LogPrefix + fmt.Sprintf("%04d", session.Id % 10000)
}
