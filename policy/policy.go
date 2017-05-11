package policy

import (
	"encoding/json"
	"time"

	"github.com/scjalliance/resourceful/environment"
)

// Policy describes the matching conditions and rules for handling a particular
// resource.
//
// A policy is applied only if all of its conditions are matched.
type Policy struct {
	Criteria Criteria      `json:"criteria"`
	Limit    uint          `json:"limit"`
	Duration time.Duration `json:"duration"` // Time between scheduled re-evaluations of the policy condition
	// FIXME: JSON duration codec
}

// New returns a new policy with the given limit, duration and criteria.
func New(limit uint, duration time.Duration, criteria Criteria) Policy {
	return Policy{
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
