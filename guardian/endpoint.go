package guardian

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/scjalliance/resourceful/guardian/transport"
	"github.com/scjalliance/resourceful/lease"
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

// Policies returns the current set of policies from the endpoint.
func (e Endpoint) Policies() (response transport.PoliciesResponse, err error) {
	return response, e.get("policies", &response)
}

// Leases returns the current set of leases for a resource from the endpoint.
func (e Endpoint) Leases(resource string) (response transport.PoliciesResponse, err error) {
	return response, e.get("leases", &response)
}

// Acquire attempts to acquire a lease for the given resource and consumer.
func (e Endpoint) Acquire(subject lease.Subject, props lease.Properties) (response transport.AcquireResponse, err error) {
	return response, e.post("acquire", subject, props, &response)
}

// Release attempts to remove the lease for the given resource and consumer.
func (e Endpoint) Release(subject lease.Subject) (response transport.ReleaseResponse, err error) {
	return response, e.post("release", subject, nil, &response)
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

func (e Endpoint) post(path string, subject lease.Subject, props lease.Properties, response interface{}) (err error) {
	if e == "" {
		return ErrEmptyEndpoint
	}

	addr := e.prefix() + path

	r, err := http.PostForm(addr, urlValues(subject, props))
	if err != nil {
		return err
	}
	defer r.Body.Close()

	switch r.StatusCode {
	case http.StatusNoContent:
		// TODO: Extract a retry interval from the Cache-Control header?
		/*
			if cc := r.Header.Get("Cache-Control"); cc != "" {
			}
		*/
		return ErrLeaseNotRequired
	case http.StatusOK:
		return json.NewDecoder(r.Body).Decode(response)
	default:
		return fmt.Errorf("http status: %v", r.Status)
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
