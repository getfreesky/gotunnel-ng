package main

import (
	"encoding/binary"
	"net"
)

type DeliveryListener struct {
	*Actor
}

func NewDeliveryListener(listenAddr string) (*DeliveryListener, error) {
	ln, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return nil, err
	}
	listener := &DeliveryListener{
		Actor: NewActor(),
	}
	listener.OnClose(func() {
		ln.Close()
	})
	go func() {
		deliveries := make(map[int64]*Delivery)
		var source int64
		for {
			conn, err := ln.Accept()
			if err != nil {
				continue
			}
			err = binary.Read(conn, binary.BigEndian, &source)
			if err != nil {
				conn.Close()
				continue
			}
			if delivery, ok := deliveries[source]; ok { // existing delivery
				delivery.conn = conn
				close(delivery.connReady)
				continue
			}
			delivery, err := NewIncomingDelivery(conn, source)
			if err != nil {
				conn.Close()
				continue
			}
			listener.Signal("notify:delivery", delivery)
			deliveries[source] = delivery
		}
	}()
	return listener, nil
}
