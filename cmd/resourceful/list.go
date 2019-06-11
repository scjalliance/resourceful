package main

import (
	"context"
	"fmt"
	"os"
	"time"

	ps "github.com/mitchellh/go-ps"
	"github.com/scjalliance/resourceful/policy"
	"github.com/scjalliance/resourceful/strategy"
)

func list(ctx context.Context, server string) {
	prepareConsole(false)

	var criteria policy.Criteria
	for _, target := range os.Args[2:] {
		criteria = append(criteria, policy.Criterion{Component: policy.ComponentResource, Comparison: policy.ComparisonIgnoreCase, Value: target})
	}

	pol := policy.New("test", strategy.Instance, 1, time.Minute*5, criteria)

	procs, err := ps.Processes()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to retrieve process list: %v\n", err)
		os.Exit(2)
	}

	if len(procs) == 0 {
		fmt.Println("No matching processes.")
		os.Exit(0)
	}

	fmt.Print("Matching processes:")
	for _, proc := range procs {
		if pol.Match(proc.Executable(), "user", nil) {
			fmt.Printf("\n  %v", proc.Executable())
		}
	}
}
