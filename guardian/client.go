package guardian

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

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

// NewClientWithServers creates a new guardian client that will use the given
// list of guardian servers.
func NewClientWithServers(servers []string) (client *Client, err error) {
	var services []serviceresolver.Service
	for _, server := range servers {
		var (
			target string
			port   uint16
		)
		target, port, err = splitTargetPort(server)
		if err != nil {
			return
		}
		services = append(services, serviceresolver.Service{
			Addrs: []*net.SRV{
				&net.SRV{
					Target: target,
					Port:   port,
				},
			},
		})
	}
	if len(services) == 0 {
		return nil, errors.New("no services specified")
	}
	return &Client{
		services: services,
	}, nil
}

// Acquire will attempt to acquire a lease for the given resource and consumer.
func (c *Client) Acquire(resource, consumer, instance string, env environment.Environment) (response transport.AcquireResponse, err error) {
	err = c.query("acquire", resource, consumer, instance, env, &response)
	return
}

// Maintain will attempt to acquire and automatically renew a lease until ctx
// is cancelled. When ctx is cancelled the lease will be released.
//
// The result of each acquisition or observation will be retuned via the
// lease manager to all listeners.
//
// If retry is a non-zero duration the maintainer will attempt to acquire a
// lease on an interval of retry.
func (c *Client) Maintain(ctx context.Context, resource, consumer, instance string, env environment.Environment, retry time.Duration) (lm *LeaseMaintainer) {
	return newLeaseMaintainer(ctx, c, resource, consumer, instance, env, retry)
}

// Release will attempt to remove the lease for the given resource and consumer.
func (c *Client) Release(resource, consumer, instance string) (response transport.ReleaseResponse, err error) {
	err = c.query("release", resource, consumer, instance, nil, &response)
	return
}

// query works through the services list in-order looking for a
// service endpoint that can successfully service the query.
func (c *Client) query(path string, resource, consumer, instance string, env environment.Environment, response interface{}) (err error) {
	var failed []error

	// Try to use the same endpoint as last time if we've already selected one
	if len(c.endpoint) > 0 {
		err = post(c.endpoint+path, resource, consumer, instance, env, response)
		if err == nil {
			return
		}
		failed = append(failed, err)
	}

	// Failover or initial selection
	for _, service := range c.services {
		for _, addr := range service.Addrs {
			endpoint := "http://" + strings.TrimRight(addr.Target, ".") + ":" + strconv.Itoa((int)(addr.Port)) + "/"
			err = post(endpoint+path, resource, consumer, instance, env, response)
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

func post(address, resource, consumer, instance string, env environment.Environment, response interface{}) (err error) {
	v := url.Values{}
	if resource != "" {
		v.Set("resource", resource)
	}
	if consumer != "" {
		v.Set("consumer", consumer)
	}
	if instance != "" {
		v.Set("instance", instance)
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

	return
}

func splitTargetPort(addr string) (target string, port uint16, err error) {
	target = addr
	port = uint16(DefaultPort)

	if strings.Contains(addr, ":") {
		var p1 string
		var p2 uint64
		target, p1, err = net.SplitHostPort(addr)
		if err != nil {
			return
		}
		p2, err = strconv.ParseUint(p1, 10, 16)
		if err != nil {
			return
		}
		port = uint16(p2)
	}

	return
}
