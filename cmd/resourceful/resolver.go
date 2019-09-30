package main

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/gentlemanautomaton/serviceresolver"
	"github.com/scjalliance/resourceful/guardian"
)

type resolver struct{}

func (resolver) Resolve(ctx context.Context) (endpoints guardian.EndpointSet, err error) {
	services, err := serviceresolver.DefaultResolver.Resolve(ctx, "resourceful")
	if err != nil {
		return nil, fmt.Errorf("failed to locate resourceful endpoints: %v", err)
	}
	if len(services) == 0 {
		return nil, errors.New("unable to detect host domain")
	}
	for _, service := range services {
		for _, addr := range service.Addrs {
			endpoint := guardian.Endpoint(fmt.Sprintf("http://%s:%d", strings.TrimSuffix(addr.Target, "."), addr.Port))
			endpoints = append(endpoints, endpoint)
		}
	}
	return endpoints, nil
}
