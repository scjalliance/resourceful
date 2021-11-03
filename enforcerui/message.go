//go:build windows
// +build windows

package enforcerui

import (
	"github.com/scjalliance/resourceful/lease"
	"github.com/scjalliance/resourceful/policy"
)

// Message Types
const (
	TypePolicyUpdate       = "policy.update"
	TypeLeaseUpdate        = "lease.update"
	TypeEnforcementUpdate  = "enforcement.update"
	TypeProcessTermination = "process.termination"
	//TypeLicenseLost        = "license.lost"
)

// Message is a UI message.
type Message struct {
	Type        string            `json:"type"`
	Policies    PolicyUpdate      `json:"policy,omitempty"`
	Leases      LeaseUpdate       `json:"leases,omitempty"`
	Enforcement EnforcementUpdate `json:"enforcement,omitempty"`
	ProcTerm    ProcTerm          `json:"procTerm,omitempty"`
}

// PolicyUpdate stores a policy set update.
type PolicyUpdate struct {
	Old policy.Set `json:"old,omitempty"`
	New policy.Set `json:"new,omitempty"`
}

// LeaseUpdate stores a lease set update.
type LeaseUpdate struct {
	Old lease.Set `json:"old,omitempty"`
	New lease.Set `json:"new,omitempty"`
}

// EnforcementUpdate reports on the current state of enforcement.
type EnforcementUpdate struct {
	Running []lease.Properties `json:"running,omitempty"`
	Waiting []lease.Properties `json:"waiting,omitempty"`
}

// ProcTerm indicates that one of the user's processes has been terminated.
type ProcTerm struct {
	Name string `json:"name"`
}
