package policy

import (
	"time"

	"github.com/scjalliance/resourceful/lease"
	"github.com/scjalliance/resourceful/strategy"
)

// Set is a set of policies.
type Set []Policy

// Diff returns the additions and deletions in next when compared with s.
func (s Set) Diff(next Set) (additions, deletions Set) {
	if s == nil {
		additions = next
		return
	}

	if next == nil {
		deletions = s
		return
	}

	var (
		a = make(map[Hash]*Policy)
		b = make(map[Hash]*Policy)
	)

	for i := range s {
		a[s[i].Hash()] = &s[i]
	}

	for i := range next {
		b[next[i].Hash()] = &next[i]
	}

	for key, pol := range a {
		if _, exists := b[key]; !exists {
			deletions = append(deletions, *pol)
		}
	}

	for key, pol := range b {
		if _, exists := a[key]; !exists {
			additions = append(additions, *pol)
		}
	}

	return
}

// MatchResource returns the subset of policies which apply to resource.
func (s Set) MatchResource(resource string) (matches Set) {
	for p := range s {
		if s[p].Resource == resource {
			matches = append(matches, s[p])
		}
	}
	return
}

// Match returns the subset of policies which match the given properties.
func (s Set) Match(props lease.Properties) (matches Set) {
	for p := range s {
		if s[p].Match(props) {
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

// Resources returns a slice of unique resources within the policy set.
//
// If the set is empty, the returned value will be nil.
func (s Set) Resources() (resources []string) {
	seen := make(map[string]bool)
	for i := 0; i < len(s); i++ {
		resource := s[i].Resource
		if resource != "" && !seen[resource] {
			resources = append(resources, resource)
		}
	}
	return
}

// Properties returns the merged properties of all policies in the set.
func (s Set) Properties() (props lease.Properties) {
	var list []lease.Properties
	for i := 0; i < len(s); i++ {
		list = append(list, s[i].Properties)
	}
	return lease.MergeProperties(list...)
}
