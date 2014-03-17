package main

import "reflect"

type Actor struct {
	*Closer
	cases        []reflect.SelectCase
	callbacks    []callbackFunc
	recvChan     chan *recvInfo
	stopRecvChan chan *recvInfo
	continueLoop chan bool
}

type callbackFunc func(reflect.Value)

type recvInfo struct {
	c interface{}
	f callbackFunc
}

func NewActor() *Actor {
	actor := &Actor{
		Closer:       new(Closer),
		recvChan:     make(chan *recvInfo, 16),
		stopRecvChan: make(chan *recvInfo, 16),
		continueLoop: make(chan bool),
	}
	// recv
	actor.cases = append(actor.cases, reflect.SelectCase{
		Dir:  reflect.SelectRecv,
		Chan: reflect.ValueOf(actor.recvChan),
	})
	actor.callbacks = append(actor.callbacks, actor.recv)
	// stop recv
	actor.cases = append(actor.cases, reflect.SelectCase{
		Dir:  reflect.SelectRecv,
		Chan: reflect.ValueOf(actor.stopRecvChan),
	})
	actor.callbacks = append(actor.callbacks, actor.stopRecv)
	// loop
	go func() {
		for {
			n, v, ok := reflect.Select(actor.cases)
			if ok {
				actor.callbacks[n](v)
			}
			if actor.IsClosed {
				break
			}
		}
	}()
	// break
	breakLoop := make(chan bool)
	actor.Recv(breakLoop, func(v reflect.Value) {})
	actor.OnClose(func() {
		close(breakLoop)
	})
	// continue
	actor.Recv(actor.continueLoop, func(v reflect.Value) {})
	return actor
}

func (self *Actor) recv(v reflect.Value) {
	info := v.Interface().(*recvInfo)
	self.cases = append(self.cases, reflect.SelectCase{
		Dir:  reflect.SelectRecv,
		Chan: reflect.ValueOf(info.c),
	})
	self.callbacks = append(self.callbacks, info.f)
}

func (self *Actor) Recv(c interface{}, f callbackFunc) {
	self.recvChan <- &recvInfo{
		c: c, f: f,
	}
	select {
	case self.continueLoop <- true:
	default:
	}
}

func (self *Actor) stopRecv(v reflect.Value) {
	info := v.Interface().(*recvInfo)
	var n int
	for i, c := range self.cases {
		if reflect.DeepEqual(c.Chan.Interface(), info.c) {
			n = i
			break
		}
	}
	self.cases = append(self.cases[:n], self.cases[n+1:]...)
	self.callbacks = append(self.callbacks[:n], self.callbacks[n+1:]...)
}

func (self *Actor) StopRecv(c interface{}) {
	self.stopRecvChan <- &recvInfo{
		c: c,
	}
	select {
	case self.continueLoop <- true:
	default:
	}
}
