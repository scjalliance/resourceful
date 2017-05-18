package policy

import (
	"time"

	"github.com/scjalliance/resourceful/environment"
)

// Set is a set of policies.
type Set []Policy

// Match returns the subset of policies which match the given parameters.
func (s Set) Match(resource, consumer string, env environment.Environment) (matches Set) {
	for p := range s {
		if s[p].Match(resource, consumer, env) {
			matches = append(matches, s[p])
		}
	}
	return
}

// Limit returns the lease limit for the policy set, which is the
// minimum value within the set.
//
// If the set is empty, DefaultLimit is returned.
func (s Set) Limit() (limit uint) {
	if len(s) == 0 {
		return DefaultLimit
	}

	limit = s[0].Limit
	for i := 1; i < len(s); i++ {
		if s[i].Limit < limit {
			limit = s[i].Limit
		}
	}
	return
}

// Duration returns the lease duration for the policy set, which is the
// minimum value within the set.
//
// If the set is empty, DefaultDuration is returned.
func (s Set) Duration() (duration time.Duration) {
	if len(s) == 0 {
		return DefaultDuration
	}

	duration = s[0].Duration
	for i := 1; i < len(s); i++ {
		if s[i].Duration < duration {
			duration = s[i].Duration
		}
	}
	return
}

// Resource returns the first resource defined in the policy set.
//
// If the set is empty, the returned value will be blank.
func (s Set) Resource() (resource string) {
	for i := 0; i < len(s); i++ {
		if s[i].Resource != "" {
			return s[i].Resource
		}
	}
	return ""
}

// Consumer returns the first consumer defined in the policy set.
//
// If the set is empty, the returned value will be blank.
func (s Set) Consumer() (consumer string) {
	for i := 0; i < len(s); i++ {
		if s[i].Consumer != "" {
			return s[i].Consumer
		}
	}
	return ""
}
