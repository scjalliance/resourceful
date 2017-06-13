package policy

import (
	"encoding/json"
	"time"

	"github.com/scjalliance/resourceful/environment"
	"github.com/scjalliance/resourceful/lease"
	"github.com/scjalliance/resourceful/strategy"
)

// TODO: Use https://github.com/valyala/fasttemplate for consumer construction

// Policy describes the matching conditions and rules for handling a particular
// resource.
//
// A policy is applied only if all of its conditions are matched.
type Policy struct {
	Resource    string                  `json:"resource,omitempty"`    // Overrides client-supplied resource
	Consumer    string                  `json:"consumer,omitempty"`    // Overrides client-supplied consumer
	Environment environment.Environment `json:"environment,omitempty"` // Overrides client-supplied environment
	Criteria    Criteria                `json:"criteria,omitempty"`
	Strategy    strategy.Strategy       `json:"strategy,omitempty"`
	Limit       uint                    `json:"limit,omitempty"`
	Duration    time.Duration           `json:"duration,omitempty"` // Time before a leased resource is automatically released
	Decay       time.Duration           `json:"decay,omitempty"`    // Time before a released resource is considered available again
	Refresh     lease.Refresh           `json:"refresh,omitempty"`  // Time between lease acquisitions while maintaining a lease
}

// New returns a new policy for a particular resource with the given limit,
// duration and criteria.
func New(resource string, strat strategy.Strategy, limit uint, duration time.Duration, criteria Criteria) Policy {
	return Policy{
		Resource: resource,
		Criteria: criteria,
		Strategy: strat,
		Limit:    limit,
		Duration: duration,
	}
}

// MarshalJSON will encode the policy as JSON.
func (p *Policy) MarshalJSON() ([]byte, error) {
	type pol Policy
	type refresh struct {
		Active string `json:"active,omitempty"`
		Queued string `json:"queued,omitempty"`
	}
	return json.Marshal(&struct {
		*pol
		Duration string  `json:"duration"`
		Decay    string  `json:"decay"`
		Refresh  refresh `json:"refresh,omitempty"`
	}{
		pol:      (*pol)(p),
		Duration: p.Duration.String(),
		Decay:    p.Decay.String(),
		Refresh: refresh{
			Active: p.Refresh.Active.String(),
			Queued: p.Refresh.Queued.String(),
		},
	})
}

// UnmarshalJSON will decode JSON policy data.
func (p *Policy) UnmarshalJSON(data []byte) error {
	type pol Policy
	type refresh struct {
		Active string `json:"active,omitempty"`
		Queued string `json:"queued,omitempty"`
	}
	aux := &struct {
		*pol
		Duration string  `json:"duration"`
		Decay    string  `json:"decay"`
		Refresh  refresh `json:"refresh,omitempty"`
	}{
		pol: (*pol)(p),
	}
	var err error
	if err = json.Unmarshal(data, aux); err != nil {
		return err
	}
	if aux.Duration != "" {
		if p.Duration, err = time.ParseDuration(aux.Duration); err != nil {
			return err
		}
	}
	if aux.Decay != "" {
		if p.Decay, err = time.ParseDuration(aux.Decay); err != nil {
			return err
		}
	}
	if aux.Refresh.Active != "" {
		if p.Refresh.Active, err = time.ParseDuration(aux.Refresh.Active); err != nil {
			return err
		}
	}
	if aux.Refresh.Queued != "" {
		if p.Refresh.Queued, err = time.ParseDuration(aux.Refresh.Queued); err != nil {
			return err
		}
	}
	return nil
}

// Match returns true if the policy applies to the given resource, consumer and
// environment.
//
// All of the policy's conditions must match for the policy to be applied.
func (p *Policy) Match(resource, consumer string, env environment.Environment) bool {
	return p.Criteria.Match(resource, consumer, env)
}
