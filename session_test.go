package main

import (
	"bytes"
	"encoding/binary"
	"io"
	"net"
	"testing"

	"code.google.com/p/go.net/proxy"
)

func TestSession(t *testing.T) {
	// server
	wait := make(chan bool)
	listener, err := NewDeliveryListener("localhost:35000")
	if err != nil {
		t.Fatal(err)
	}
	var serverDelivery *Delivery
	listener.OnSignal("notify:delivery", func(delivery *Delivery) {
		serverDelivery = delivery
		wait <- true
	})

	// local
	delivery, err := NewOutgoingDelivery("localhost:35000") // delivery
	if err != nil {
		t.Fatal(err)
	}
	socksServer, err := NewSocksServer("localhost:36000") // socks server
	if err != nil {
		t.Fatal(err)
	}
	localSessionManager := NewSessionManager(delivery) // local session manager
	socksServer.OnSignal("client", func(conn net.Conn, hostPort string) {
		session, err := NewOutgoingSession(delivery, hostPort, conn)
		if err != nil {
			t.Fatal(err)
		}
		localSessionManager.Sessions[session.id] = session
	})

	<-wait
	NewSessionManager(serverDelivery)

	// ping pong server
	go func() {
		ln, err := net.Listen("tcp", "localhost:37000")
		if err != nil {
			t.Fatal(err)
		}
		for {
			conn, err := ln.Accept()
			if err != nil {
				t.Fatal(err)
			}
			go func() {
				for {
					var length uint16
					err = binary.Read(conn, binary.BigEndian, &length)
					if err != nil {
						t.Fatal(err)
					}
					data := make([]byte, length)
					n, _ := io.ReadFull(conn, data)
					if n != int(length) {
						t.Fatal(err)
					}
					// pong
					binary.Write(conn, binary.BigEndian, length)
					conn.Write(data)
				}
			}()
		}
	}()

	// socks client
	proxy, err := proxy.SOCKS5("tcp", "localhost:36000", nil, proxy.Direct)
	if err != nil {
		t.Fatal(err)
	}
	conn, err := proxy.Dial("tcp", "localhost:37000")
	if err != nil {
		t.Fatal(err)
	}

	// ping pong test
	data := []byte("hello")
	binary.Write(conn, binary.BigEndian, uint16(len(data)))
	conn.Write(data)
	var length uint16
	err = binary.Read(conn, binary.BigEndian, &length)
	if err != nil {
		t.Fatal(err)
	}
	response := make([]byte, length)
	n, _ := io.ReadFull(conn, response)
	if n != int(length) {
		t.Fatal("data incomplete")
	}
	if !bytes.Equal(data, response) {
		t.Fatal("data not match")
	}
}
