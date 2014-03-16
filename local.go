package main

import "net"

type Local struct {
	*Node
}

func NewLocal(listenAddr, serverAddr string) (*Local, error) {
	local := &Local{
		Node: NewNode(),
	}
	// connect to server
	conn, err := net.Dial("tcp", serverAddr)
	if err != nil {
		return nil, err
	}
	delivery, err := NewDelivery(conn, false)
	if err != nil {
		return nil, err
	}
	local.AddDelivery(delivery)
	return local, nil
}
