package main

import (
	"context"
	"fmt"

	"github.com/scjalliance/resourceful/guardian"
)

func newClient(ctx context.Context, server string) (*guardian.Client, error) {
	var endpoints []guardian.Endpoint
	if server != "" {
		endpoints = []guardian.Endpoint{guardian.Endpoint(server)}
	} else {
		var err error
		endpoints, err = collectEndpoints(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve endpoints: %v", err)
		}
	}

	client, err := guardian.NewClient(endpoints...)
	if err != nil {
		return nil, fmt.Errorf("failed to create resourceful guardian client: %v", err)
	}

	return client, nil
}
