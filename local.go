package main

import (
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
	local.LogPrefix = "."
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
	local.Recv(socksServer.NewClient, func(v reflect.Value) { // new socks client
		info := v.Interface().(SocksClientInfo)
		local.Log("new socks client %v %s", info.Conn, info.HostPort)
		session := NewSession(-1, delivery.Source, info.Conn, info.HostPort)
		local.AddSession(session)
	})
	return local, nil
}
