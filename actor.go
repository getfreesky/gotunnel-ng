package main

import (
	"reflect"
	"sync"
)

type Actor struct {
	*Closer
	cases        []reflect.SelectCase
	callbacks    []callbackFunc
	continueLoop chan bool
	lock         sync.Mutex
}

func NewActor() *Actor {
	actor := &Actor{
		Closer:       new(Closer),
		continueLoop: make(chan bool),
	}
	// select loop
	breakLoop := make(chan bool)
	actor.Recv(breakLoop, func(v reflect.Value, ok bool) {})
	actor.OnClose(func() {
		close(breakLoop)
	})
	actor.Recv(actor.continueLoop, func(v reflect.Value, ok bool) {})
	go func() {
		for {
			n, v, ok := reflect.Select(actor.cases)
			actor.callbacks[n](v, ok)
			if actor.IsClosed {
				break
			}
		}
	}()

	return actor
}

type callbackFunc func(reflect.Value, bool)

func (self *Actor) Recv(c interface{}, f callbackFunc) {
	self.lock.Lock()
	defer self.lock.Unlock()
	self.cases = append(self.cases, reflect.SelectCase{
		Dir:  reflect.SelectRecv,
		Chan: reflect.ValueOf(c),
	})
	self.callbacks = append(self.callbacks, f)
	select {
	case self.continueLoop <- true:
	default:
	}
}

func (self *Actor) StopRecv(c interface{}) {
	self.lock.Lock()
	defer self.lock.Unlock()
	var n int
	for i, c := range self.cases {
		if reflect.DeepEqual(c.Chan.Interface(), c) {
			n = i
			break
		}
	}
	self.cases = append(self.cases[:n], self.cases[n+1:]...)
	self.callbacks = append(self.callbacks[:n], self.callbacks[n+1:]...)
	select {
	case self.continueLoop <- true:
	default:
	}
}
