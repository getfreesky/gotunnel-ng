package main

import (
	"fmt"
	"net"
	"reflect"
)

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
	// start socks server
	socksServer, err := NewSocksServer(listenAddr)
	if err != nil {
		return nil, err
	}
	local.Recv(socksServer.NewClient, func(v reflect.Value, ok bool) { // new socks client
		info := v.Interface().(SocksClientInfo)
		fmt.Printf("new socks client %v %s\n", info.Conn, info.HostPort)
	})
	return local, nil
}
