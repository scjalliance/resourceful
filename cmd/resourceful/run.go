package main

import (
	"context"
	"errors"
	"log"
	"os"

	"github.com/scjalliance/resourceful/runner"
)

func runError(err error) {
	msgBox("resourceful run error", err.Error())
	os.Exit(2)
}

func run(args []string) {
	if len(args) == 0 {
		runError(errors.New("no executable path provided to run"))
	}

	servers, args := splitServersArgs(args)
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

	err := runner.RunWithIcon(ctx, program, args, servers, icon)
	if err != nil {
		runError(err)
	}
}

func splitServersArgs(combined []string) (servers []string, args []string) {
	args = combined
	for len(args) > 2 && args[0] == "-s" && args[1] != "" {
		servers = append(servers, args[1])
		args = args[2:]
	}
	return
}
