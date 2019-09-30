package guardian

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/scjalliance/resourceful/guardian/transport"
	"github.com/scjalliance/resourceful/lease"
)

// An Endpoint is a guardian service URL.
type Endpoint string

// Health returns the current health of the endpoint.
func (e Endpoint) Health(ctx context.Context) (response transport.HealthResponse, err error) {
	return response, e.get(ctx, "health", &response)
}

// Policies returns the current set of policies from the endpoint.
func (e Endpoint) Policies(ctx context.Context) (response transport.PoliciesResponse, err error) {
	return response, e.get(ctx, "policies", &response)
}

// Leases returns the current set of leases for a resource from the endpoint.
func (e Endpoint) Leases(ctx context.Context, resource string) (response transport.PoliciesResponse, err error) {
	return response, e.get(ctx, "leases", &response)
}

// Acquire attempts to acquire a lease for the given resource and consumer.
func (e Endpoint) Acquire(ctx context.Context, subject lease.Subject, props lease.Properties) (response transport.AcquireResponse, err error) {
	return response, e.post(ctx, "acquire", subject, props, &response)
}

// Release attempts to remove the lease for the given resource and consumer.
func (e Endpoint) Release(ctx context.Context, subject lease.Subject) (response transport.ReleaseResponse, err error) {
	return response, e.post(ctx, "release", subject, nil, &response)
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

func (e Endpoint) get(ctx context.Context, path string, response interface{}) (err error) {
	if e == "" {
		return ErrEmptyEndpoint
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	addr := e.prefix() + path
	req, err := http.NewRequest("GET", addr, nil)
	if err != nil {
		return err
	}

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

func (e Endpoint) post(ctx context.Context, path string, subject lease.Subject, props lease.Properties, response interface{}) (err error) {
	if e == "" {
		return ErrEmptyEndpoint
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	addr := e.prefix() + path
	body := strings.NewReader(urlValues(subject, props).Encode())
	req, err := http.NewRequest("POST", addr, body)
	if err != nil {
		return err
	}
	req = req.WithContext(ctx)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusNoContent:
		// TODO: Extract a retry interval from the Cache-Control header?
		/*
			if cc := r.Header.Get("Cache-Control"); cc != "" {
			}
		*/
		return ErrLeaseNotRequired
	case http.StatusOK:
		return json.NewDecoder(resp.Body).Decode(response)
	default:
		return fmt.Errorf("http status: %v", resp.Status)
	}
}

func urlValues(subject lease.Subject, props lease.Properties) url.Values {
	v := url.Values{}
	if subject.Resource != "" {
		v.Set("resource", subject.Resource)
	}
	if subject.Instance.Host != "" {
		v.Set("host", subject.Instance.Host)
	}
	if subject.Instance.User != "" {
		v.Set("user", subject.Instance.User)
	}
	if subject.Instance.ID != "" {
		v.Set("instance", subject.Instance.ID)
	}
	if props != nil {
		for key, value := range props {
			v.Set(key, value)
		}
	}
	return v
}
