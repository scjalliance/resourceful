// +build windows

package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/scjalliance/resourceful/enforcer"
	"github.com/scjalliance/resourceful/policy"
	"github.com/scjalliance/resourceful/provider/fsprov"
	"golang.org/x/sys/windows/svc/eventlog"
)

func enforceService(conf EnforceConfig, confErr error) {
	elog, err := eventlog.Open(enforcer.ServiceName)
	if err != nil {
		return
	}
	defer elog.Close()

	elog.Info(enforcer.ServiceEventID, fmt.Sprintf("Starting %s service.", enforcer.ServiceName))
	defer func() {
		elog.Info(enforcer.ServiceEventID, fmt.Sprintf("Stopped %s service.", enforcer.ServiceName))
	}()

	logger := svcLogger{elog: elog}

	handler := Handler{
		Name:    enforcer.ServiceName,
		Conf:    conf,
		ConfErr: confErr,
		Logger:  logger,
	}

	if err := handler.Run(); err != nil {
		elog.Error(enforcer.ServiceEventID, fmt.Sprintf("Error running service: %v", err))
	}
}

func enforceInteractive(ctx context.Context, conf EnforceConfig) {
	client := newClient(conf.Server)

	logger := cliLogger{
		Debug: conf.Debug,
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

	service := enforcer.New(client, time.Second, time.Minute, uiCommand, environment, cache, conf.Passive, logger)

	service.Start()
	<-ctx.Done()
	service.Stop()
}
