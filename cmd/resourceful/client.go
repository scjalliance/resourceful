package main

import (
	"github.com/scjalliance/resourceful/guardian"
)

func newClient(server string) *guardian.Client {
	if server != "" {
		return guardian.NewClient(guardian.EndpointSet{guardian.Endpoint(server)})
	}
	return guardian.NewClient(resolver{})
}
