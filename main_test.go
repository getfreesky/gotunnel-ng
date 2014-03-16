package main

import (
	"testing"
	"time"
)

func TestMain(t *testing.T) {
	server, err := NewServer("0.0.0.0:35000")
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	local, err := NewLocal("0.0.0.0:10800", "localhost:35000")
	if err != nil {
		t.Fatal(err)
	}
	defer local.Close()

	time.Sleep(time.Second * 1)
}
