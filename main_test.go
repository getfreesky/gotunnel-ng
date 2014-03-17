package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net"
	"sync"
	"testing"
	"time"

	"code.google.com/p/go.net/proxy"
)

func TestDelivery(t *testing.T) {
	server, err := NewServer("0.0.0.0:35000")
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	for i := 0; i < 10; i++ {
		local, err := NewLocal(fmt.Sprintf("0.0.0.0:%d", 10800+i), "localhost:35000")
		if err != nil {
			t.Fatal(err)
		}
		defer local.Close()
		time.Sleep(time.Millisecond * 100)
		if len(server.Deliveries) != i+1 {
			t.Fatalf("delivery failed")
		}
	}
}

func TestSocks(t *testing.T) {
	// server
	server, err := NewServer("0.0.0.0:35000")
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	// local
	_, err = NewLocal("0.0.0.0:31080", "localhost:35000")
	if err != nil {
		t.Fatal(err)
	}
	// target server
	ln, err := net.Listen("tcp", "localhost:35200")
	if err != nil {
		t.Fatal(err)
	}
	response := bytes.Repeat([]byte("hello"), 1024)
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				t.Fatal(err)
			}
			go func() {
				conn.Write(response)
				conn.Close()
			}()
		}
	}()
	// client
	proxy, err := proxy.SOCKS5("tcp", "localhost:31080", nil, proxy.Direct)
	if err != nil {
		t.Fatal(err)
	}
	wg := new(sync.WaitGroup)
	n := 100
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			conn, err := proxy.Dial("tcp", "localhost:35200")
			if err != nil {
				t.Fatal(err)
			}
			data, err := ioutil.ReadAll(conn)
			if !bytes.Equal(data, response) {
				t.Fail()
			}
			conn.Close()
		}()
	}
	wg.Wait()
}
