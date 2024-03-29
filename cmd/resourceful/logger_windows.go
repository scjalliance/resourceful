//go:build windows
// +build windows

package main

import (
	"fmt"

	"github.com/scjalliance/resourceful/enforcer"
	"golang.org/x/sys/windows/svc/eventlog"
)

type cliLogger struct {
	Debug bool
}

func (l cliLogger) Log(e enforcer.Event) {
	if e.IsDebug() && !l.Debug {
		return
	}
	s := e.String()
	if len(s) == 0 || s[len(s)-1] != '\n' {
		s = s + "\n"
	}
	fmt.Print(s)
}

type svcLogger struct {
	elog *eventlog.Log
}

func (logger svcLogger) Log(e enforcer.Event) {
	logger.elog.Info(e.ID(), e.String())
}
