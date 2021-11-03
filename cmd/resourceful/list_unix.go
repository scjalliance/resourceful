//go:build !windows
// +build !windows

package main

import (
	"context"
	"fmt"
	"os"
	"strconv"

	ps "github.com/mitchellh/go-ps"
	"github.com/scjalliance/resourceful/lease"
)

func list(ctx context.Context, conf ListConfig) {
	prepareConsole(false)

	host, err := os.Hostname()
	if err != nil {
		fmt.Printf("Failed to query local hostname: %v\n", err)
		os.Exit(1)
	}

	policies, err := collectPolicies(ctx, conf.Server)
	if err != nil {
		fmt.Printf("Failed to collect resourceful policies: %v\n", err)
		os.Exit(1)
	}

	procs, err := ps.Processes()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to retrieve process list: %v\n", err)
		os.Exit(2)
	}

	if len(procs) == 0 {
		fmt.Printf("No matching processes.\n")
		os.Exit(0)
	}

	fmt.Printf("Matching processes:\n")
	for _, p := range procs {
		props := lease.Properties{
			"program.name": p.Executable(),
			"process.id":   strconv.Itoa(p.Pid()),
			"host.name":    host,
		}
		if matches := policies.Match(props); len(matches) > 0 {
			fmt.Printf("  %s\n", p.Executable())
		}
	}
}
