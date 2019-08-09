package main

import (
	"context"
	"fmt"

	"github.com/scjalliance/resourceful/policy"
)

func collectPolicies(ctx context.Context, server string) (policy.Set, error) {
	client, err := newClient(ctx, server)
	if err != nil {
		return nil, err
	}

	response, err := client.Policies()
	if err != nil {
		return nil, fmt.Errorf("failed to collect resourceful policies: %v", err)
	}

	return response.Policies, nil
}
