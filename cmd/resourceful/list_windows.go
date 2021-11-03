//go:build windows
// +build windows

package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/gentlemanautomaton/winproc"
	"github.com/scjalliance/resourceful/enforcer"
	"github.com/scjalliance/resourceful/lease"
)

func list(ctx context.Context, conf ListConfig) {
	prepareConsole(false)

	environment, err := buildEnvironment()
	if err != nil {
		fmt.Printf("Failed to collect environment: %v\n", err)
		os.Exit(1)
	}

	policies, err := collectPolicies(ctx, conf.Server)
	if err != nil {
		fmt.Printf("Failed to collect resourceful policies: %v\n", err)
		os.Exit(1)
	}

	procs, err := enforcer.Scan(policies, environment)
	if err != nil {
		fmt.Printf("Failed to collect processes: %v\n", err)
		os.Exit(1)
	}

	if len(procs) == 0 {
		fmt.Printf("No matching processes.\n")
		os.Exit(0)
	}

	fmt.Printf("Processes:\n")
	for _, process := range procs {
		instance := enforcer.Instance(environment["host.name"], process, enforcer.NewInstanceID(process))
		props := enforcer.Properties(process, environment)
		if matches := policies.Match(props); len(matches) > 0 {
			fmt.Printf("%s\n", process)
			fmt.Printf("  Resource: %s\n", matches.Resource())
			fmt.Printf("  Instance: %s\n", instance)
			fmt.Printf("  Limit: %d\n", matches.Limit())
			fmt.Printf("  Duration: %s\n", matches.Duration())
			merged := lease.MergeProperties(props, matches.Properties())
			for key, value := range merged {
				fmt.Printf("  %s: %s\n", key, value)
			}
		}
	}
	//printChildren(0, tree)
}

func printChildren(depth int, nodes []winproc.Node) {
	for _, node := range nodes {
		fmt.Printf("%s%s\n", strings.Repeat("  ", depth), node.Process)
		printChildren(depth+1, node.Children)
	}
}
