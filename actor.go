package main

import "reflect"

type Actor struct {
	IsClosed       bool
	cases          []reflect.SelectCase
	callbacks      []callbackFunc
	signalChan     chan signalInfo
	signalHandlers map[string][]interface{}
}

type callbackFunc func(reflect.Value)

type signalInfo struct {
	signal string
	v      []interface{}
}

func NewActor() *Actor {
	actor := &Actor{
		signalChan:     make(chan signalInfo, 1024),
		signalHandlers: make(map[string][]interface{}),
	}
	actor.cases = []reflect.SelectCase{
		{ // signal
			Dir:  reflect.SelectRecv,
			Chan: reflect.ValueOf(actor.signalChan),
		},
	}
	actor.callbacks = []callbackFunc{
		func(v reflect.Value) { // signal
			info := v.Interface().(signalInfo)
			if handlers, ok := actor.signalHandlers[info.signal]; ok {
				var args []reflect.Value
				for _, arg := range info.v {
					args = append(args, reflect.ValueOf(arg))
				}
				for _, handler := range handlers {
					reflect.ValueOf(handler).Call(args)
				}
			}
		},
	}
	actor.OnSignal("__recv", func(c interface{}, f callbackFunc) {
		actor.cases = append(actor.cases, reflect.SelectCase{
			Dir:  reflect.SelectRecv,
			Chan: reflect.ValueOf(c),
		})
		actor.callbacks = append(actor.callbacks, f)
	})
	actor.OnSignal("__stop_recv", func(c interface{}) {
		var n int
		for i, c := range actor.cases {
			if reflect.DeepEqual(c.Chan.Interface(), c) {
				n = i
				break
			}
		}
		actor.cases = append(actor.cases[:n], actor.cases[n+1:]...)
		actor.callbacks = append(actor.callbacks[:n], actor.callbacks[n+1:]...)
	})
	actor.OnSignal("__close", func() {
		actor.IsClosed = true
	})

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

func (self *Actor) OnSignal(signal string, f interface{}) {
	self.signalHandlers[signal] = append(self.signalHandlers[signal], f)
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
