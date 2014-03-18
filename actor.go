package main

import "reflect"

type Actor struct {
	*Closer
	cases          []reflect.SelectCase
	callbacks      []callbackFunc
	recvChan       chan *recvInfo
	stopRecvChan   chan *recvInfo
	signalChan     chan string
	signalHandlers map[string]func()
}

type callbackFunc func(reflect.Value)

type recvInfo struct {
	c interface{}
	f callbackFunc
}

func NewActor() *Actor {
	actor := &Actor{
		Closer:         new(Closer),
		recvChan:       make(chan *recvInfo, 128), //TODO make it unbuffered
		stopRecvChan:   make(chan *recvInfo, 128),
		signalChan:     make(chan string),
		signalHandlers: make(map[string]func()),
	}
	actor.OnClose(func() {
		actor.Signal("__break")
	})
	actor.cases = []reflect.SelectCase{
		{ // recv
			Dir:  reflect.SelectRecv,
			Chan: reflect.ValueOf(actor.recvChan),
		},
		{ // stop recv
			Dir:  reflect.SelectRecv,
			Chan: reflect.ValueOf(actor.stopRecvChan),
		},
		{ // signal
			Dir:  reflect.SelectRecv,
			Chan: reflect.ValueOf(actor.signalChan),
		},
	}
	actor.callbacks = []callbackFunc{
		func(v reflect.Value) { // recv
			info := v.Interface().(*recvInfo)
			actor.cases = append(actor.cases, reflect.SelectCase{
				Dir:  reflect.SelectRecv,
				Chan: reflect.ValueOf(info.c),
			})
			actor.callbacks = append(actor.callbacks, info.f)
		},
		func(v reflect.Value) { // stop recv
			info := v.Interface().(*recvInfo)
			var n int
			for i, c := range actor.cases {
				if reflect.DeepEqual(c.Chan.Interface(), info.c) {
					n = i
					break
				}
			}
			actor.cases = append(actor.cases[:n], actor.cases[n+1:]...)
			actor.callbacks = append(actor.callbacks[:n], actor.callbacks[n+1:]...)
		},
		func(v reflect.Value) { // signal
			signal := v.Interface().(string)
			if handler, ok := actor.signalHandlers[signal]; ok {
				handler()
			}
		},
	}

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

	return actor
}

func (self *Actor) Recv(c interface{}, f callbackFunc) {
	self.recvChan <- &recvInfo{
		c: c, f: f,
	}
	self.Signal("__continue")
}

func (self *Actor) StopRecv(c interface{}) {
	self.stopRecvChan <- &recvInfo{
		c: c,
	}
	self.Signal("__continue")
}

func (self *Actor) OnSignal(signal string, f func()) {
	self.signalHandlers[signal] = f
}

func (self *Actor) Signal(signal string) {
	self.signalChan <- signal
}
