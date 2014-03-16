package main

import (
	"reflect"
	"sync"
	"time"
)

type Node struct {
	Loop       *SelectLoop
	Deliveries []*Delivery

	onCloseFuncs []func()
	CloseChan    chan bool
	IsClosed     bool
	closeOnce    sync.Once
}

func NewNode() *Node {
	// node
	node := &Node{
		CloseChan: make(chan bool),
	}
	// select loop
	selectLoop := new(SelectLoop)
	selectLoop.Recv(node.CloseChan, func(v reflect.Value, ok bool) {
		node.IsClosed = true
	})
	go func() {
		for {
			selectLoop.Select()
			if node.IsClosed {
				break
			}
		}
	}()
	node.Loop = selectLoop

	return node
}

func (self *Node) AddDelivery(delivery *Delivery) {
	self.Loop.Recv(delivery.IncomingPacket, func(v reflect.Value, ok bool) { // incoming packet
	})
	self.Loop.Recv(delivery.CloseChan, func(v reflect.Value, ok bool) { // delivery is broken
		self.Loop.StopRecv(delivery.IncomingPacket)
		self.Loop.StopRecv(delivery.CloseChan)
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

func (self *Node) OnClose(f func()) {
	self.onCloseFuncs = append(self.onCloseFuncs, f)
}

func (self *Node) Close() {
	self.closeOnce.Do(func() {
		close(self.CloseChan)
		time.Sleep(time.Millisecond * 50)
		for _, delivery := range self.Deliveries {
			select {
			case <-delivery.CloseChan:
			default:
				delivery.Close()
			}
		}
		for _, f := range self.onCloseFuncs {
			f()
		}
	})
}
