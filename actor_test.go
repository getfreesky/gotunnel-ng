package main

import (
	"reflect"
	"testing"
)

type testActor struct {
	*Actor
}

func TestActor(t *testing.T) {
	s := &testActor{
		Actor: NewActor(),
	}
	c := make(chan bool)
	var success bool
	done := make(chan bool)
	s.Recv(make(chan bool), func(v reflect.Value, ok bool) {})
	s.Recv(c, func(v reflect.Value, ok bool) {
		success = true
		close(done)
	})
	c <- true
	<-done
	if !success {
		t.Fail()
	}
}
