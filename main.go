package main

import (
	"log"
	"math/rand"
	"net/http"
	_ "net/http/pprof"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())

	go http.ListenAndServe("localhost:55555", nil)
}

func main() {
	server, err := NewServer("0.0.0.0:35000")
	if err != nil {
		log.Fatal(err)
	}
	defer server.Close()

	_, err = NewLocal("0.0.0.0:31080", "localhost:35000")
	if err != nil {
		log.Fatal(err)
	}
	<-(make(chan bool))
}
