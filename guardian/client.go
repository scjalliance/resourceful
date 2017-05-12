package guardian

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/gentlemanautomaton/serviceresolver"
	"github.com/scjalliance/resourceful/environment"
	"github.com/scjalliance/resourceful/guardian/transport"
)

type serviceSelection struct {
	Service int
	Addr    int
}

// Client coordinates resource leasing with a resourceful guardian server.
type Client struct {
	service  string
	services []serviceresolver.Service
	endpoint string
}

// NewClient creates a new guardian client that will resolve services for the
// given service name.
//
// TODO:
// Query the health of multiple servers in parallel, probably with a
// 50ms delay between each, in order to proactively locate a functional
// endpoint.
//
// TODO:
// Break service resolution out into its own thing that plugs into a client.
func NewClient(service string) (*Client, error) {
	ctx := context.TODO()

	services, err := serviceresolver.DefaultResolver.Resolve(ctx, service)
	if err != nil {
		return nil, fmt.Errorf("client: %v", err)
	}
	if len(services) == 0 {
		return nil, errors.New("Unable to detect host domain")
	}

	return &Client{
		service:  service,
		services: services,
	}, nil
}

// Acquire will attempt to acquire a lease for the given resource and consumer.
func (c *Client) Acquire(resource, consumer string, env environment.Environment) (response transport.AcquireResponse, err error) {
	err = c.query("acquire", resource, consumer, env, &response)
	return
}

// Release will attempt to remove the lease for the given resource and consumer.
func (c *Client) Release(resource, consumer string) (response transport.ReleaseResponse, err error) {
	err = c.query("release", resource, consumer, nil, &response)
	return
}

// query works through the services list in-order looking for a
// service endpoint that can successfully service the query.
func (c *Client) query(path string, resource, consumer string, env environment.Environment, response interface{}) (err error) {
	var failed []error

	// Try to use the same endpoint as last time if we've already selected one
	if len(c.endpoint) > 0 {
		err = post(c.endpoint+path, resource, consumer, env, response)
		if err == nil {
			return
		}
		failed = append(failed, err)
	}

	// Failover or initial selection
	for _, service := range c.services {
		for _, addr := range service.Addrs {
			endpoint := "http://" + strings.TrimRight(addr.Target, ".") + ":" + strconv.Itoa((int)(addr.Port)) + "/"
			err = post(endpoint+path, resource, consumer, env, response)
			if err == nil {
				c.endpoint = endpoint
				return
			}
			failed = append(failed, err)
		}
	}

	f := len(failed)
	switch {
	case f == 1:
		err = fmt.Errorf("%s failed: %v", path, failed[0])
	case f > 1:
		err = fmt.Errorf("%s failed: attempts to connect to %d servers failed, the last error was: %v", path, f, failed[f-1])
	default:
		err = fmt.Errorf("%s failed: no servers available", path)
	}

	return
}

func post(address, resource, consumer string, env environment.Environment, response interface{}) (err error) {
	v := url.Values{}
	if resource != "" {
		v.Set("resource", resource)
	}
	if consumer != "" {
		v.Set("consumer", consumer)
	}
	if env != nil {
		for key, value := range env {
			v.Set(key, value)
		}
	}
	r, err := http.PostForm(address, v)
	if err != nil {
		return
	}

	defer r.Body.Close()

	if r.StatusCode != 200 {
		err = fmt.Errorf("http status: %v", r.Status)
		return
	}

	err = json.NewDecoder(r.Body).Decode(response)
	if err != nil {
		log.Print(err)
	}

	return
}
