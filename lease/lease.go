package lease

import (
	"time"

	"github.com/scjalliance/resourceful/environment"
	"github.com/scjalliance/resourceful/strategy"
)

// Lease describes a single assignment of a leased resource.
type Lease struct {
	Resource    string                  `json:"resource"`    // The thing that is being leased
	Consumer    string                  `json:"consumer"`    // The entity that holds the lease
	Instance    string                  `json:"instance"`    // Optional identifier of a particular lease instance
	Environment environment.Environment `json:"environment"` // Map of additional properties of the lease
	Status      Status                  `json:"status"`
	Started     time.Time               `json:"started,omitempty"`
	Renewed     time.Time               `json:"renewed,omitempty"`
	Released    time.Time               `json:"released,omitempty"`
	Strategy    strategy.Strategy       `json:"strategy,omitempty"`
	Limit       uint                    `json:"limit"`
	Duration    time.Duration           `json:"duration"`
	Decay       time.Duration           `json:"decay"`
	Refresh     Refresh                 `json:"refresh,omitempty"`
}

// MatchConsumer returns true if the lease is for the given resource and consumer.
func (ls *Lease) MatchConsumer(resource, consumer string) (matched bool) {
	return ls.Resource == resource && ls.Consumer == consumer
}

// MatchInstance returns true if the lease is for the given resource,
// consumer and instance.
func (ls *Lease) MatchInstance(resource, consumer, instance string) (matched bool) {
	return ls.Resource == resource && ls.Consumer == consumer && ls.Instance == instance
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
	name := ls.Environment["resource.name"]
	if name != "" {
		return name
	}

	name = ls.Environment["resource.id"]
	if name != "" {
		return name
	}

	return ls.Resource
}

// Subject returns a string identifying the subject of the lease, which
// includes the instance, consumer and resource.
func (ls *Lease) Subject() Subject {
	return Subject{
		Resource: ls.Resource,
		Consumer: ls.Consumer,
		Instance: ls.Instance,
	}
}

// Specified returns true if the lease has a subject.
func (ls *Lease) Specified() bool {
	return ls.Resource != "" || ls.Consumer != "" || ls.Instance != ""
}

// Clone returns a deep copy of the lease.
func Clone(from Lease) (to Lease) {
	to.Resource = from.Resource
	to.Consumer = from.Consumer
	to.Instance = from.Instance
	to.Environment = environment.Clone(from.Environment)
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
