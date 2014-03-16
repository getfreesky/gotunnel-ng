package main

import "reflect"

type Actor struct {
	*Closer
	Loop *SelectLoop
}

func NewActor() *Actor {
	actor := &Actor{
		Closer: new(Closer),
	}
	// select loop
	selectLoop := new(SelectLoop)
	closeChan := make(chan bool)
	selectLoop.Recv(closeChan, func(v reflect.Value, ok bool) {})
	actor.OnClose(func() {
		close(closeChan)
	})
	go func() {
		for {
			selectLoop.Select()
			if actor.IsClosed {
				break
			}
		}
	}()
	actor.Loop = selectLoop

	return actor
}

type callbackFunc func(reflect.Value, bool)

type SelectLoop struct {
	cases     []reflect.SelectCase
	callbacks []callbackFunc
}

func (self *SelectLoop) Recv(c interface{}, f callbackFunc) {
	self.cases = append(self.cases, reflect.SelectCase{
		Dir:  reflect.SelectRecv,
		Chan: reflect.ValueOf(c),
	})
	self.callbacks = append(self.callbacks, f)
}

func (self *SelectLoop) Select() {
	n, v, ok := reflect.Select(self.cases)
	self.callbacks[n](v, ok)
}

func (self *SelectLoop) StopRecv(c interface{}) {
	var n int
	for i, c := range self.cases {
		if reflect.DeepEqual(c.Chan.Interface(), c) {
			n = i
			break
		}
	}
	self.cases = append(self.cases[:n], self.cases[n+1:]...)
	self.callbacks = append(self.callbacks[:n], self.callbacks[n+1:]...)
}
