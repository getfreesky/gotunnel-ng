package main

import "sync"

type Closer struct {
	lock         sync.Mutex
	onCloseFuncs []func()
	IsClosed     bool
	closeOnce    sync.Once
}

func (self *Closer) OnClose(f func()) {
	self.lock.Lock()
	defer self.lock.Unlock()
	self.onCloseFuncs = append(self.onCloseFuncs, f)
}

func (self *Closer) Close() {
	self.closeOnce.Do(func() {
		self.lock.Lock()
		defer self.lock.Unlock()
		for _, f := range self.onCloseFuncs {
			f()
		}
		self.IsClosed = true
	})
}
