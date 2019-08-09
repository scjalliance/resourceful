// +build windows

package main

type loggerFunc func(string, ...interface{})

func (lf loggerFunc) Printf(format string, v ...interface{}) {
	lf(format, v...)
}
