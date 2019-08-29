package lease

import (
	"time"

	"github.com/scjalliance/resourceful/strategy"
)

// Lease describes a single assignment of a leased resource.
type Lease struct {
	Subject
	Properties Properties        `json:"properties"` // Properties of the lease
	Status     Status            `json:"status"`
	Started    time.Time         `json:"started,omitempty"`
	Renewed    time.Time         `json:"renewed,omitempty"`
	Released   time.Time         `json:"released,omitempty"`
	Strategy   strategy.Strategy `json:"strategy,omitempty"`
	Limit      uint              `json:"limit"`
	Duration   time.Duration     `json:"duration"`
	Decay      time.Duration     `json:"decay"`
	Refresh    Refresh           `json:"refresh,omitempty"`
}

// MatchHostUser returns true if the lease is for the given resource, host and user.
func (ls *Lease) MatchHostUser(resource, host, user string) (matched bool) {
	return ls.Resource == resource && ls.Instance.Host == host && ls.Instance.User == user
}

// MatchInstance returns true if the lease is for the given resource and instance.
func (ls *Lease) MatchInstance(resource string, inst Instance) (matched bool) {
	return ls.Resource == resource && ls.Instance == inst
}

// MatchStatus returns true if the lease has the given status.
func (ls *Lease) MatchStatus(status Status) (matched bool) {
	return ls.Status == status
}

// Consumptive returns true if the lease is active or released.
func (ls *Lease) Consumptive() (matched bool) {
	switch ls.Status {
	case Active, Released:
		return true
	default:
		return false
	}
}

// ExpirationTime returns the time at which the lease expires.
func (ls *Lease) ExpirationTime() time.Time {
	return ls.Renewed.Add(ls.Duration)
}

// Expired returns true if the lease will be expired at the given time.
func (ls *Lease) Expired(at time.Time) bool {
	return at.After(ls.ExpirationTime())
}

// DecayTime returns the time at which the lease decays.
func (ls *Lease) DecayTime() time.Time {
	var release time.Time
	if !ls.Released.IsZero() {
		release = ls.Released
	} else {
		release = ls.ExpirationTime()
	}
	return release.Add(ls.Decay)
}

// Decayed returns true if the lease will be fully decayed at the given time.
func (ls *Lease) Decayed(at time.Time) bool {
	return at.After(ls.DecayTime())
}

// EffectiveRefresh returns the effective refresh interval for the lease.
//
// If the lease refresh interval is non-zero, it will be returned. If the
// refresh interval is zero a computed interval of half the lease duration will
// will be returned instead.
//
// The returned value will always be greater than or equal to the minimum
// refresh rate defined by MinimumRefresh.
func (ls *Lease) EffectiveRefresh() (interval time.Duration) {
	switch ls.Status {
	case Active:
		interval = ls.Refresh.Active
	case Queued:
		interval = ls.Refresh.Queued
	}

	if interval == 0 {
		interval = ls.Duration / 2
	}

	if interval < MinimumRefresh {
		interval = MinimumRefresh
	}

	return interval
}

// ResourceName returns the user-friendly name of the resource.
func (ls *Lease) ResourceName() string {
	if name := ls.Properties["resource.name"]; name != "" {
		return name
	}

	if name := ls.Properties["resource.id"]; name != "" {
		return name
	}

	return ls.Resource
}

// Clone returns a deep copy of the lease.
func Clone(from Lease) (to Lease) {
	to.Subject = from.Subject
	to.Properties = from.Properties.Clone()
	to.Status = from.Status
	to.Started = from.Started
	to.Renewed = from.Renewed
	to.Released = from.Released
	to.Strategy = from.Strategy
	to.Limit = from.Limit
	to.Duration = from.Duration
	to.Decay = from.Decay
	to.Refresh = from.Refresh
	return
}
