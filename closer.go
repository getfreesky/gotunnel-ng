package main

import "sync"

type Closer struct {
	onCloseFuncs []func()
	IsClosed     bool
	closeOnce    sync.Once
}

func (self *Closer) OnClose(f func()) {
	self.onCloseFuncs = append(self.onCloseFuncs, f)
}

func (self *Closer) Close() {
	self.closeOnce.Do(func() {
		for _, f := range self.onCloseFuncs {
			f()
		}
		self.IsClosed = true
	})
}
