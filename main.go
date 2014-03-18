package main

import (
	"log"
	"math/rand"
	"net"
	"net/http"
	_ "net/http/pprof"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())

	go http.ListenAndServe("localhost:55555", nil)
}

func main() {
	// server
	listener, err := NewDeliveryListener("localhost:35000")
	if err != nil {
		log.Fatal(err)
	}
	listener.OnSignal("notify:delivery", func(delivery *Delivery) {
		NewSessionManager(delivery)
	})
	// local
	delivery, err := NewOutgoingDelivery("localhost:35000")
	if err != nil {
		log.Fatal(err)
	}
	localSessionManager := NewSessionManager(delivery)
	socksServer, err := NewSocksServer("localhost:31080")
	if err != nil {
		log.Fatal(err)
	}
	socksServer.OnSignal("client", func(conn net.Conn, hostPort string) {
		session, err := NewOutgoingSession(delivery, hostPort, conn)
		if err != nil {
			log.Fatal(err)
		}
		localSessionManager.Sessions[session.id] = session
	})
	<-(make(chan bool))
}
