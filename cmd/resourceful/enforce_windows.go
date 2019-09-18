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
		logger = cliLogger{}
		prepareConsole(false)
	} else {
		elog, err := eventlog.Open(enforcer.ServiceName)
		if err != nil {
			return
		}
		defer elog.Close()
		logger = svcLogger{elog: elog}
	}

	executable, err := os.Executable()
	if err != nil {
		fmt.Printf("Failed to query executable: %v\n", err)
		os.Exit(1)
	}
	uiCommand := enforcer.Command{
		Path: executable,
		Args: []string{"ui"},
	}

	hostname, err := os.Hostname()
	if err != nil {
		fmt.Printf("Failed to query local hostname: %v\n", err)
		os.Exit(1)
	}

	service := enforcer.New(client, time.Second, time.Minute, uiCommand, hostname, passive, logger)

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
