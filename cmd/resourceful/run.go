package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/scjalliance/resourceful/guardian"
	"github.com/scjalliance/resourceful/lease/leaseui"
	"github.com/scjalliance/resourceful/runner"
)

func runError(err error) {
	leaseui.Notify("resourceful run error", err.Error())
	os.Exit(2)
}

func run(ctx context.Context, server, program string, args []string) {
	if program == "" {
		runError(errors.New("no executable path provided to run"))
	}

	var endpoints []guardian.Endpoint
	if server != "" {
		endpoints = append(endpoints, guardian.Endpoint(server))
	} else {
		var err error
		endpoints, err = collectEndpoints(ctx)
		if err != nil {
			runError(err)
		}
	}

	icon := programIcon()

	config := runner.Config{
		Icon:    icon,
		Program: program,
		Args:    args,
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

// splitEndpointArgs extracts a single -s argument from the start of the arg
// list if present and interpets it as a guardian endpoint. Any remaining
// arguments are returned and will be passed to the executable being run.
func splitEndpointArgs(combined []string) (endpoints []guardian.Endpoint, args []string) {
	args = combined
	for len(args) > 2 && args[0] == "-s" && args[1] != "" {
		endpoints = append(endpoints, guardian.Endpoint(args[1]))
		args = args[2:]
	}
	return
}
