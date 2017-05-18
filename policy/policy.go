package policy

import (
	"encoding/json"
	"time"

	"github.com/scjalliance/resourceful/environment"
)

// TODO: Use https://github.com/valyala/fasttemplate for consumer construction

// Policy describes the matching conditions and rules for handling a particular
// resource.
//
// A policy is applied only if all of its conditions are matched.
type Policy struct {
	Resource string        `json:"resource,omitempty"` // Overrides client-supplied resource
	Consumer string        `json:"consumer,omitempty"` // Overrides client-supplied consumer
	Criteria Criteria      `json:"criteria,omitempty"`
	Limit    uint          `json:"limit,omitempty"`
	Duration time.Duration `json:"duration,omitempty"` // Time between scheduled re-evaluations of the policy condition
}

// New returns a new policy for a particular resource with the given limit,
// duration and criteria.
func New(resource string, limit uint, duration time.Duration, criteria Criteria) Policy {
	return Policy{
		Resource: resource,
		Criteria: criteria,
		Limit:    limit,
		Duration: duration,
	}
}

// MarshalJSON will encode the policy as JSON.
func (p *Policy) MarshalJSON() ([]byte, error) {
	type jsonPolicy Policy
	return json.Marshal(&struct {
		*jsonPolicy
		Duration string `json:"duration"`
	}{
		jsonPolicy: (*jsonPolicy)(p),
		Duration:   p.Duration.String(),
	})
}

// UnmarshalJSON will decode JSON policy data.
func (p *Policy) UnmarshalJSON(data []byte) error {
	type jsonPolicy Policy
	aux := &struct {
		*jsonPolicy
		Duration string `json:"duration"`
	}{
		jsonPolicy: (*jsonPolicy)(p),
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
	return nil
}

// Match returns true if the policy applies to the given resource, consumer and
// environment.
//
// All of the policy's conditions must match for the policy to be applied.
func (p *Policy) Match(resource, consumer string, env environment.Environment) bool {
	return p.Criteria.Match(resource, consumer, env)
}
