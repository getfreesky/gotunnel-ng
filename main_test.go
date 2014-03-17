package main

import (
	"fmt"
	"testing"
	"time"
)

func TestBasic(t *testing.T) {
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
		fmt.Printf("%d deliveries\n", len(server.Deliveries))
		if len(server.Deliveries) != i+1 {
			t.Fatalf("delivery failed")
		}
	}
}

func TestSocks(t *testing.T) {
	server, err := NewServer("0.0.0.0:35000")
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	local, err := NewLocal("0.0.0.0:31080", "localhost:35000")
	if err != nil {
		t.Fatal(err)
	}
	_ = local
}
