// +build windows

package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/gentlemanautomaton/winproc"
	"github.com/scjalliance/resourceful/enforcer"
)

func list(ctx context.Context, server string) {
	prepareConsole(false)

	policies, err := collectPolicies(ctx, server)
	if err != nil {
		fmt.Printf("Failed to collect resourceful policies: %v\n", err)
		os.Exit(1)
	}

	procs, err := enforcer.Scan(policies)
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
		if matches := policies.Match(process.Resource, process.Consumer, process.Env()); len(matches) > 0 {
			fmt.Printf("%s\n", process)
			if resource := matches.Resource(); resource != "" && resource != process.Resource {
				fmt.Printf("  Resource: %s (%s)\n", resource, process.Resource)
			} else {
				fmt.Printf("  Resource: %s\n", process.Resource)
			}

			if consumer := matches.Consumer(); consumer != "" && consumer != process.Consumer {
				fmt.Printf("  Consumer: %s (%s)\n", consumer, process.Consumer)
			} else {
				fmt.Printf("  Consumer: %s\n", process.Consumer)
			}

			fmt.Printf("  Instance: %s\n", process.Instance)
			fmt.Printf("  Limit: %d\n", matches.Limit())
			fmt.Printf("  Duration: %s\n", matches.Duration())
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
