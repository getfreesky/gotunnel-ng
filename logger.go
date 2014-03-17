package main

import "fmt"

type Logger struct {
	LogPrefix string
}

func (self *Logger) Log(format string, args ...interface{}) {
	fmt.Printf("%s %s\n", self.LogPrefix, fmt.Sprintf(format, args...))
}
