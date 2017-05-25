package lease

import (
	"time"

	"github.com/scjalliance/resourceful/environment"
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
	Limit       uint                    `json:"limit"`
	Duration    time.Duration           `json:"duration"`
	Decay       time.Duration           `json:"decay"`
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

// Expired returns true if the lease will be expired at the given time.
func (ls *Lease) Expired(at time.Time) bool {
	expiration := ls.Renewed.Add(ls.Duration)
	return at.After(expiration)
}

// Decayed returns true if the lease will be expired and have exceeded the
// duration of its decay at the given time.
func (ls *Lease) Decayed(at time.Time) bool {
	decay := ls.Renewed.Add(ls.Duration).Add(ls.Decay)
	return at.After(decay)
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
	to.Limit = from.Limit
	to.Duration = from.Duration
	to.Decay = from.Decay
	return
}
