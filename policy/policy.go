package policy

import (
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

// New returns a new policy with the given limit, duration and conditions.
func New(limit uint, duration time.Duration, criteria Criteria) Policy {
	return Policy{
		Criteria: criteria,
		Limit:    limit,
		Duration: duration,
	}
}

// Match returns true if the policy applies to the given resource, consumer and
// environment.
//
// All of the policy's conditions must match for the policy to be applied.
func (pol *Policy) Match(resource, consumer string, env environment.Environment) bool {
	return pol.Criteria.Match(resource, consumer, env)
}
