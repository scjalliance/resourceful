package main

import (
	"fmt"
	"os"

	"github.com/scjalliance/resourceful/lease"
)

func buildEnvironment() (lease.Properties, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, fmt.Errorf("failed to query local hostname: %v", err)
	}

	return lease.Properties{
		"host.name":      hostname,
		"client.version": Version,
	}, nil
}
