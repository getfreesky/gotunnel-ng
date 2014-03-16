package main

import (
	"log"
	"net"
	"reflect"
	"time"
)

type Server struct {
	ln         net.Listener
	IsClosed   chan bool
	deliveries []*Delivery
}

func NewServer(addr string) (*Server, error) {
	// select loop
	selectLoop := new(SelectLoop)
	// server
	server := &Server{
		IsClosed: make(chan bool),
	}
	isClosed := false
	selectLoop.Recv(server.IsClosed, func(v reflect.Value, ok bool) {
		isClosed = true
	})
	// delivery
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	server.ln = ln
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				if isClosed {
					return
				}
				log.Fatal(err)
			}
			//TODO auth by password or ip
			delivery := NewDelivery(conn)
			go delivery.Run()
			selectLoop.Recv(delivery.IncomingPacket, func(v reflect.Value, ok bool) { // incoming packet
			})
			selectLoop.Recv(delivery.IsClosed, func(v reflect.Value, ok bool) { // delivery is broken
				selectLoop.StopRecv(delivery.IncomingPacket)
				selectLoop.StopRecv(delivery.IsClosed)
				var n int
				for i, d := range server.deliveries {
					if d == delivery {
						n = i
						break
					}
				}
				server.deliveries = append(server.deliveries[:n], server.deliveries[n+1:]...)
			})
			server.deliveries = append(server.deliveries, delivery)
		}
	}()
	// select loop
	go func() {
		for {
			selectLoop.Select()
			if isClosed {
				break
			}
		}
	}()
	return server, nil
}

func (self *Server) Close() {
	close(self.IsClosed)
	time.Sleep(time.Millisecond * 200)
	self.ln.Close()
	for _, delivery := range self.deliveries {
		delivery.Close()
	}
}
