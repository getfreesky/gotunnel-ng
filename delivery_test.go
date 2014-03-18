package main

import (
	"bytes"
	"testing"
)

func TestDelivery(t *testing.T) {
	done := make(chan bool)
	var serverDelivery *Delivery

	// server
	listener, err := NewDeliveryListener("localhost:35500")
	if err != nil {
		t.Fatal(err)
	}
	listener.OnSignal("notify:delivery", func(delivery *Delivery) {
		serverDelivery = delivery
		done <- true
	})

	// client
	delivery, err := NewOutgoingDelivery("localhost:35500")
	if err != nil {
		t.Fatal(err)
	}
	<-done // wait for server delivery

	// connection broken test
	delivery.OnSignal("notify:reconnected", func() {
		done <- true
	})
	serverDelivery.OnSignal("notify:reconnected", func() {
		done <- true
	})
	n := 100
	for i := 0; i < n; i++ {
		delivery.conn.Close()
		<-done // wait for local delivery reconnect
		<-done // wait for server delivery reconnect
		if delivery.reconnectTimes != i+1 {
			t.Fatal("local delivery reconnect fail")
		}
		if serverDelivery.reconnectTimes != i+1 {
			t.Fatal("server delivery reconnect fail")
		}
	}
	for i := 0; i < n; i++ {
		serverDelivery.conn.Close()
		<-done // wait for local delivery reconnect
		<-done // wait for server delivery reconnect
		if delivery.reconnectTimes != n+i+1 {
			t.Fatal("local delivery reconnect fail")
		}
		if serverDelivery.reconnectTimes != n+i+1 {
			t.Fatal("server delivery reconnect fail")
		}
	}

	// communication test
	n = 1024
	msg := bytes.Repeat([]byte("hello"), 4096)
	for i := 0; i < n; i++ {
		go delivery.Send(msg)
		go serverDelivery.Send(msg)
	}
	for i := 0; i < n; i++ {
		data := <-serverDelivery.IncomingPacket
		if !bytes.Equal(msg, data) {
			t.Fatal("data not match")
		}
		data = <-delivery.IncomingPacket
		if !bytes.Equal(msg, data) {
			t.Fatal("data not match")
		}
	}
}
