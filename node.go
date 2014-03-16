package main

import "reflect"

type Node struct {
	*Actor
	Deliveries []*Delivery
}

func NewNode() *Node {
	// node
	node := &Node{
		Actor: NewActor(),
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
	self.Loop.Recv(delivery.IncomingPacket, func(v reflect.Value, ok bool) { // incoming packet TODO
	})
	delivery.OnClose(func() {
		self.Loop.StopRecv(delivery.IncomingPacket)
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
