package policy

import (
	"bufio"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/scjalliance/resourceful/lease"
	"github.com/scjalliance/resourceful/strategy"
	"golang.org/x/crypto/sha3"
)

// TODO: Consider giving policies the ability to construct a "consumer" from
// any number of properties. Doing this would allow policies flexibility in
// describing the unit of consumption.
//
// We could use something like https://github.com/valyala/fasttemplate for
// this construction.

// Policy describes the matching conditions and rules for handling a particular
// resource.
//
// A policy is applied only if all of its conditions are matched.
type Policy struct {
	Resource   string            `json:"resource,omitempty"`   // Which resource pool this policy counts against
	Criteria   Criteria          `json:"criteria,omitempty"`   // Matching criteria for lease properties
	Strategy   strategy.Strategy `json:"strategy,omitempty"`   // Lease counting strategy
	Limit      uint              `json:"limit,omitempty"`      // Max concurrent leases
	Duration   time.Duration     `json:"duration,omitempty"`   // Time before a leased resource is automatically released
	Decay      time.Duration     `json:"decay,omitempty"`      // Time before a released resource is considered available again
	Refresh    lease.Refresh     `json:"refresh,omitempty"`    // Time between lease acquisitions while maintaining a lease
	Properties lease.Properties  `json:"properties,omitempty"` // Merged with each lease's properties
}

// New returns a new policy for a particular resource with the given limit,
// duration and criteria.
func New(resource string, strat strategy.Strategy, limit uint, duration time.Duration, criteria Criteria) Policy {
	return Policy{
		Resource: resource,
		Criteria: criteria,
		Strategy: strat,
		Limit:    limit,
		Duration: duration,
	}
}

// MarshalJSON will encode the policy as JSON.
func (p *Policy) MarshalJSON() ([]byte, error) {
	type pol Policy
	type refresh struct {
		Active string `json:"active,omitempty"`
		Queued string `json:"queued,omitempty"`
	}
	return json.Marshal(&struct {
		*pol
		Duration string  `json:"duration"`
		Decay    string  `json:"decay"`
		Refresh  refresh `json:"refresh,omitempty"`
	}{
		pol:      (*pol)(p),
		Duration: p.Duration.String(),
		Decay:    p.Decay.String(),
		Refresh: refresh{
			Active: p.Refresh.Active.String(),
			Queued: p.Refresh.Queued.String(),
		},
	})
}

// UnmarshalJSON will decode JSON policy data.
func (p *Policy) UnmarshalJSON(data []byte) error {
	type pol Policy
	type refresh struct {
		Active string `json:"active,omitempty"`
		Queued string `json:"queued,omitempty"`
	}
	aux := &struct {
		*pol
		Duration string  `json:"duration"`
		Decay    string  `json:"decay"`
		Refresh  refresh `json:"refresh,omitempty"`
	}{
		pol: (*pol)(p),
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
	if aux.Decay != "" {
		if p.Decay, err = time.ParseDuration(aux.Decay); err != nil {
			return err
		}
	}
	if aux.Refresh.Active != "" {
		if p.Refresh.Active, err = time.ParseDuration(aux.Refresh.Active); err != nil {
			return err
		}
	}
	if aux.Refresh.Queued != "" {
		if p.Refresh.Queued, err = time.ParseDuration(aux.Refresh.Queued); err != nil {
			return err
		}
	}
	return nil
}

// Match returns true if the policy applies to a lease with the given
// properites.
//
// All of the policy's conditions must match for the policy to be applied.
func (p *Policy) Match(props lease.Properties) bool {
	return p.Criteria.Match(props)
}

// String returns a string representation of the policy.
func (p *Policy) String() string {
	var parts []string
	if p.Resource != "" {
		parts = append(parts, fmt.Sprintf("Resource: %q", p.Resource))
	}
	if len(p.Criteria) > 0 {
		parts = append(parts, fmt.Sprintf("Criteria: %q", p.Criteria.String()))
	}
	if p.Strategy != "" {
		parts = append(parts, fmt.Sprintf("Strategy: %s", p.Strategy))
	}
	if p.Limit != 0 {
		parts = append(parts, fmt.Sprintf("Limit: %d", p.Limit))
	}
	if p.Duration != 0 {
		parts = append(parts, fmt.Sprintf("Duration: %s", p.Duration))
	}
	if p.Decay != 0 {
		parts = append(parts, fmt.Sprintf("Decay: %s", p.Decay))
	}
	if p.Refresh.Active != 0 {
		parts = append(parts, fmt.Sprintf("Active Refresh: %s", p.Refresh.Active))
	}
	if p.Refresh.Queued != 0 {
		parts = append(parts, fmt.Sprintf("Queued Refresh: %s", p.Refresh.Queued))
	}
	if len(p.Properties) > 0 {
		parts = append(parts, fmt.Sprintf("Properties: %q", p.Properties))
	}
	return strings.Join(parts, " ")
}

// Hash returns a 224-bit hash of the policy.
func (p *Policy) Hash() Hash {
	var (
		hash = sha3.New224()
		w    = hashWriter{bufio.NewWriterSize(hash, hash.BlockSize())}
	)

	w.WriteString(p.Resource)
	w.WriteInt(len(p.Criteria))
	for _, c := range p.Criteria {
		w.WriteString(c.Key)
		w.WriteString(c.Comparison)
		w.WriteString(c.Value)
	}
	w.WriteString(string(p.Strategy))
	w.WriteInt(int(p.Limit))
	w.WriteDuration(p.Duration)
	w.WriteDuration(p.Decay)
	w.WriteDuration(p.Refresh.Active)
	w.WriteDuration(p.Refresh.Queued)
	w.WriteInt(len(p.Properties))
	for key, value := range p.Properties {
		w.WriteString(key)
		w.WriteString(value)
	}

	if err := w.Flush(); err != nil {
		panic(err)
	}

	var h Hash
	hash.Sum(h[:0])
	return h
}
