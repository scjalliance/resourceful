// +build windows

package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/scjalliance/resourceful/enforcer"
)

func enforce(ctx context.Context, server string, interactive, passive bool) {
	client, err := newClient(ctx, server)
	if err != nil {
		fmt.Printf("Enforcement failed: %v\n", err)
		os.Exit(1)
	}

	if interactive {
		prepareConsole(false)
	}

	hostname, err := os.Hostname()
	if err != nil {
		fmt.Printf("Failed to query local hostname: %v\n", err)
		os.Exit(1)
	}

	service := enforcer.New(client, time.Second, time.Minute, hostname, passive)

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
