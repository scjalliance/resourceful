// +build windows

package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/scjalliance/resourceful/enforcer"
	"golang.org/x/sys/windows/svc/eventlog"
)

func enforce(ctx context.Context, server string, interactive, passive bool) {
	client, err := newClient(ctx, server)
	if err != nil {
		fmt.Printf("Enforcement failed: %v\n", err)
		os.Exit(1)
	}

	var logger enforcer.Logger
	if interactive {
		logger = loggerFunc(func(format string, v ...interface{}) {
			s := fmt.Sprintf(format, v...)
			if len(s) == 0 || s[len(s)-1] != '\n' {
				s = s + "\n"
			}
			fmt.Print(s)
		})
		prepareConsole(false)
	} else {
		elog, err := eventlog.Open(enforcer.ServiceName)
		if err != nil {
			return
		}
		defer elog.Close()
		logger = loggerFunc(func(format string, v ...interface{}) {
			elog.Info(0, fmt.Sprintf(format, v))
		})
	}

	hostname, err := os.Hostname()
	if err != nil {
		fmt.Printf("Failed to query local hostname: %v\n", err)
		os.Exit(1)
	}

	service := enforcer.New(client, time.Second, time.Minute, hostname, passive, logger)

	if interactive {
		service.Start()
		<-ctx.Done()
		service.Stop()
		return
	}

	handler := enforcer.Handler{
		Name:    enforcer.ServiceName,
		Service: service,
	}

	if err := handler.Run(); err != nil {
		fmt.Printf("Error running service: %v\n", err)
		os.Exit(1)
	}
}
