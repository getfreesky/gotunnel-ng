package main

import (
	"fmt"
	"log"
	"net"
)

type Server struct {
	*Node
	ln         net.Listener
	deliveries []*Delivery
}

func NewServer(addr string) (*Server, error) {
	server := &Server{
		Node: NewNode(),
	}
	server.LogPrefix = "="
	// delivery
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	server.ln = ln
	server.OnClose(func() {
		ln.Close()
	})
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				if server.IsClosed {
					return
				}
				log.Fatal(err)
			}
			//TODO auth by password or ip

			delivery, err := NewDelivery(conn, true)
			if err != nil {
				fmt.Printf("NewDelivery: %v", err)
				continue
			}
			server.AddDelivery(delivery)

		}
	}()
	return server, nil
}
