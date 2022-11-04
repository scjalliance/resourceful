//go:build windows
// +build windows

package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/alecthomas/kong"
	"github.com/scjalliance/resourceful/enforcer"
	"github.com/scjalliance/resourceful/policy"
	"github.com/scjalliance/resourceful/provider/fsprov"
	"golang.org/x/sys/windows/svc/eventlog"
)

// Run executes the enforce command.
func (cmd *EnforceCmd) Run(ctx context.Context) error {
	client := newClient(cmd.Server)

	logger := cliLogger{
		Debug: cmd.Debug,
	}
	prepareConsole(false)

	executable, err := os.Executable()
	if err != nil {
		fmt.Printf("Failed to query executable: %v\n", err)
		os.Exit(1)
	}
	uiCommand := enforcer.Command{
		Path: executable,
		Args: []string{"ui"},
	}

	environment, err := buildEnvironment()
	if err != nil {
		fmt.Printf("Failed to collect environment: %v\n", err)
		os.Exit(1)
	}

	polDir, err := cacheDir()
	if err != nil {
		fmt.Printf("Failed to locate cache directory: %v\n", err)
		os.Exit(1)
	}

	var cache policy.Cache
	if polDir != "" {
		prov := fsprov.New(polDir)
		defer prov.Close()
		cache = prov
	}

	service := enforcer.New(client, time.Second, time.Minute, uiCommand, environment, cache, cmd.Passive, logger)

	service.Start()
	<-ctx.Done()
	service.Stop()

	return nil
}

func runServiceHandler() {
	var cmd EnforceCmd
	parser := kong.Must(&cmd)
	_, parseErr := parser.Parse(os.Args[1:])

	elog, err := eventlog.Open(enforcer.ServiceName)
	if err != nil {
		return
	}
	defer elog.Close()

	elog.Info(enforcer.ServiceEventID, fmt.Sprintf("Starting %s service version %s.", enforcer.ServiceName, Version))
	defer func() {
		elog.Info(enforcer.ServiceEventID, fmt.Sprintf("Stopped %s service version %s.", enforcer.ServiceName, Version))
	}()

	logger := svcLogger{elog: elog}

	handler := Handler{
		Name:    enforcer.ServiceName,
		Conf:    cmd.Config(),
		ConfErr: parseErr,
		Logger:  logger,
	}

	if err := handler.Run(); err != nil {
		elog.Error(enforcer.ServiceEventID, fmt.Sprintf("Error running service: %v", err))
	}
}
