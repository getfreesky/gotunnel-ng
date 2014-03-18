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
	s.Recv(make(chan bool), func(v reflect.Value) {})
	s.Recv(c, func(v reflect.Value) {
		success = true
		close(done)
	})
	c <- true
	<-done
	if !success {
		t.Fail()
	}
}

func TestActorSignal(t *testing.T) {
	s := &testActor{
		Actor: NewActor(),
	}
	succ := false
	done := make(chan bool)
	s.OnSignal("foo", func() {
		succ = true
		close(done)
	})
	s.Signal("foo")
	<-done
	if !succ {
		t.Fail()
	}
	s.OnSignal("bar", func(s string) {
		if s != "bar" {
			t.Fail()
		}
	})
	s.Signal("bar", "bar")
}
