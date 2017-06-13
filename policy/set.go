package policy

import (
	"time"

	"github.com/scjalliance/resourceful/environment"
	"github.com/scjalliance/resourceful/lease"
	"github.com/scjalliance/resourceful/strategy"
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

// Strategy returns the resource counting strategy for the policy set. The
// first non-empty strategy in the set will be returned. If the set does not
// define a non-empty strategy, DefaultStrategy will be returned.
func (s Set) Strategy() strategy.Strategy {
	for p := range s {
		switch s[p].Strategy {
		case strategy.Empty:
		default:
			return s[p].Strategy
		}
	}
	return DefaultStrategy
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

// Decay returns the lease decay for the policy set, which is the
// maximum value within the set.
//
// If the set is empty, a zero value is returned.
func (s Set) Decay() (decay time.Duration) {
	if len(s) == 0 {
		return 0
	}

	decay = s[0].Decay
	for i := 1; i < len(s); i++ {
		if s[i].Decay > decay {
			decay = s[i].Decay
		}
	}
	return
}

// Refresh returns the lease refresh intervals for the policy set, which are the
// first non-zero intervals within the set.
//
// If the set is empty, a zero value is returned.
func (s Set) Refresh() (refresh lease.Refresh) {
	if len(s) == 0 {
		return
	}

	for i := 0; i < len(s); i++ {
		if refresh.Active == 0 && s[i].Refresh.Active != 0 {
			refresh.Active = s[i].Refresh.Active
		}
		if refresh.Queued == 0 && s[i].Refresh.Queued != 0 {
			refresh.Queued = s[i].Refresh.Queued
		}
		if refresh.Active != 0 && refresh.Queued != 0 {
			break
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

// Environment returns the merged environment of all policies in the set.
func (s Set) Environment() (env environment.Environment) {
	var envs []environment.Environment
	for i := 0; i < len(s); i++ {
		envs = append(envs, s[i].Environment)
	}
	return environment.Merge(envs...)
}
