package main

import (
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
}
