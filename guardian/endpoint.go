package guardian

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/scjalliance/resourceful/environment"
	"github.com/scjalliance/resourceful/guardian/transport"
)

// An Endpoint is a guardian service URL.
type Endpoint string

// Health returns the current health of the endpoint.
func (e Endpoint) Health() (response transport.HealthResponse, err error) {
	return response, e.get("health", &response)
}

// HealthWithTimeout returns the current health of the endpoint. The endpoint must
// respond within the given timeout to be deemed healthy.
func (e Endpoint) HealthWithTimeout(timeout time.Duration) (response transport.HealthResponse, err error) {
	return response, e.getWithTimeout("health", &response, timeout)
}

// Leases returns the current set of leases for a resource from the endpoint.
func (e Endpoint) Leases(resource string) (response transport.PoliciesResponse, err error) {
	return response, e.get("leases", &response)
}

// Acquire attempts to acquire a lease for the given resource and consumer.
func (e Endpoint) Acquire(resource, consumer, instance string, env environment.Environment) (response transport.AcquireResponse, err error) {
	return response, e.post("acquire", resource, consumer, instance, env, &response)
}

// Release attempts to remove the lease for the given resource and consumer.
func (e Endpoint) Release(resource, consumer, instance string) (response transport.ReleaseResponse, err error) {
	return response, e.post("release", resource, consumer, instance, nil, &response)
}

// prefix returns the URL prefix for the endpoint.
func (e Endpoint) prefix() string {
	u := string(e)
	if !strings.Contains(u, "://") {
		u = "http://" + u
	}
	if !strings.HasSuffix(u, "/") {
		u += "/"
	}
	return u
}

func (e Endpoint) get(path string, response interface{}) (err error) {
	r, err := http.Get(e.prefix() + path)
	if err != nil {
		return err
	}
	defer r.Body.Close()

	if r.StatusCode != 200 {
		return fmt.Errorf("http status: %v", r.Status)
	}

	return json.NewDecoder(r.Body).Decode(response)
}

func (e Endpoint) getWithTimeout(path string, response interface{}, timeout time.Duration) (err error) {
	if e == "" {
		return ErrEmptyEndpoint
	}

	addr := e.prefix() + path
	req, err := http.NewRequest("GET", addr, nil)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(req.Context(), timeout)
	defer cancel()

	req = req.WithContext(ctx)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("http status: %v", resp.Status)
	}

	return json.NewDecoder(resp.Body).Decode(response)
}

func (e Endpoint) post(path, resource, consumer, instance string, env environment.Environment, response interface{}) (err error) {
	if e == "" {
		return ErrEmptyEndpoint
	}

	addr := e.prefix() + path

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

	r, err := http.PostForm(addr, v)
	if err != nil {
		return err
	}
	defer r.Body.Close()

	if r.StatusCode != 200 {
		err = fmt.Errorf("http status: %v", r.Status)
		return
	}

	return json.NewDecoder(r.Body).Decode(response)
}
