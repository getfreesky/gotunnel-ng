package main

import "reflect"

type Actor struct {
	IsClosed       bool
	cases          []reflect.SelectCase
	callbacks      []callbackFunc
	signalChan     chan signalInfo
	signalHandlers map[string][]reflect.Value
}

type callbackFunc func(reflect.Value)

type signalInfo struct {
	signal string
	v      []interface{}
}

func NewActor() *Actor {
	actor := &Actor{
		signalChan:     make(chan signalInfo, 1024),
		signalHandlers: make(map[string][]reflect.Value),
	}
	actor.cases = []reflect.SelectCase{
		{
			Dir:  reflect.SelectRecv,
			Chan: reflect.ValueOf(actor.signalChan),
		},
	}
	actor.callbacks = []callbackFunc{actor.onSignal}
	actor.OnSignal("__recv", actor.recv)
	actor.OnSignal("__stop_recv", actor.stopRecv)
	actor.OnSignal("__close", actor.onClose)
	go actor.start()
	return actor
}

func (self *Actor) start() {
	for {
		n, v, ok := reflect.Select(self.cases)
		if ok {
			self.callbacks[n](v)
		}
		if self.IsClosed {
			break
		}
	}
}

func (self *Actor) onSignal(v reflect.Value) {
	info := v.Interface().(signalInfo)
	if handlers, ok := self.signalHandlers[info.signal]; ok {
		var args []reflect.Value
		for _, arg := range info.v {
			args = append(args, reflect.ValueOf(arg))
		}
		for _, handler := range handlers {
			handler.Call(args)
		}
	}
}

func (self *Actor) recv(c interface{}, f callbackFunc) {
	self.cases = append(self.cases, reflect.SelectCase{
		Dir:  reflect.SelectRecv,
		Chan: reflect.ValueOf(c),
	})
	self.callbacks = append(self.callbacks, f)
}

func (self *Actor) stopRecv(c interface{}) {
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

func (self *Actor) onClose() {
	self.IsClosed = true
}

func (self *Actor) OnSignal(signal string, f interface{}) {
	//TODO not thread safe
	self.signalHandlers[signal] = append(self.signalHandlers[signal], reflect.ValueOf(f))
}

func (self *Actor) Signal(signal string, v ...interface{}) {
	self.signalChan <- signalInfo{
		signal: signal,
		v:      v,
	}
}

func (self *Actor) Recv(c interface{}, f callbackFunc) {
	self.Signal("__recv", c, f)
}

func (self *Actor) StopRecv(c interface{}) {
	self.Signal("__stop_recv", c)
}

func (self *Actor) Close() {
	self.Signal("__close")
}

func (self *Actor) OnClose(f func()) {
	self.OnSignal("__close", f)
}
