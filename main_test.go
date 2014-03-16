package main

import "testing"

func TestMain(t *testing.T) {
	server, err := NewServer("0.0.0.0:35000")
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
}
