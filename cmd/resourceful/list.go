package main

import (
	"fmt"
	"os"
	"time"

	ps "github.com/mitchellh/go-ps"
	"github.com/scjalliance/resourceful/policy"
)

func list(args []string) {

	fmt.Println("executing list")
	var criteria policy.Criteria
	for _, target := range os.Args[2:] {
		criteria = append(criteria, policy.Criterion{Component: policy.ComponentResource, Comparison: policy.ComparisonIgnoreCase, Value: target})
	}

	pol := policy.New("test", 1, time.Minute*5, criteria)

	procs, err := ps.Processes()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to retrieve process list: %v\n", err)
		os.Exit(2)
	}

	for _, proc := range procs {
		if pol.Match(proc.Executable(), "user", nil) {
			fmt.Printf("%v\n", proc.Executable())
		}
	}
}
