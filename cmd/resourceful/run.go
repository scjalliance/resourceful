package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/gentlemanautomaton/serviceresolver"
	"github.com/scjalliance/resourceful/guardian"
	"github.com/scjalliance/resourceful/lease/leaseui"
	"github.com/scjalliance/resourceful/runner"
)

func runError(err error) {
	leaseui.Notify("resourceful run error", err.Error())
	os.Exit(2)
}

func run(args []string) {
	if len(args) == 0 {
		runError(errors.New("no executable path provided to run"))
	}

	endpoints, args := splitEndpointArgs(args)
	program := args[0]
	args = args[1:]
	icon := programIcon()

	logger := log.New(os.Stderr, "", log.LstdFlags)

	ctx, shutdown := context.WithCancel(context.Background())
	defer shutdown()
	go func() {
		waitForSignal(logger)
		shutdown()
	}()

	config := runner.Config{
		Icon:    icon,
		Program: program,
		Args:    args,
	}

	if len(endpoints) == 0 {
		var err error
		endpoints, err = collectEndpoints(ctx)
		if err != nil {
			runError(err)
		}
	}

	client, err := guardian.NewClient(endpoints...)
	if err != nil {
		runError(fmt.Errorf("unable to create resourceful guardian client: %v", err))
	}

	err = runner.Run(ctx, client, config)
	if err != nil {
		runError(err)
	}
}

func splitEndpointArgs(combined []string) (endpoints []guardian.Endpoint, args []string) {
	args = combined
	for len(args) > 2 && args[0] == "-s" && args[1] != "" {
		endpoints = append(endpoints, guardian.Endpoint(args[1]))
		args = args[2:]
	}
	return
}

func collectEndpoints(ctx context.Context) (endpoints []guardian.Endpoint, err error) {
	services, err := serviceresolver.DefaultResolver.Resolve(ctx, "resourceful")
	if err != nil {
		return nil, fmt.Errorf("failed to locate resourceful endpoints: %v", err)
	}
	if len(services) == 0 {
		return nil, errors.New("unable to detect host domain")
	}
	for _, service := range services {
		for _, addr := range service.Addrs {
			endpoint := guardian.Endpoint(fmt.Sprintf("http://%s:%d", addr.Target, addr.Port))
			endpoints = append(endpoints, endpoint)
		}
	}
	return endpoints, nil
}
