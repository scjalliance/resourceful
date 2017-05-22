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
	Duration    time.Duration           `json:"duration"`
	Decay       time.Duration           `json:"decay"`
}

// Match returns true if the lease is for the given resource, consumer and instance.
func (l *Lease) Match(resource, consumer, instance string) bool {
	return l.Resource == resource && l.Consumer == consumer && l.Instance == instance
}

// Expired returns true if the lease has expired.
func (l *Lease) Expired() bool {
	expiration := l.Renewed.Add(l.Duration)
	return time.Now().After(expiration)
}

// Clone returns a deep copy of the lease.
func Clone(from Lease) (to Lease) {
	to.Resource = from.Resource
	to.Consumer = from.Consumer
	to.Environment = environment.Clone(from.Environment)
	return
}
