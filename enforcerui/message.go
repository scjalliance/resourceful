// +build windows

package enforcerui

import (
	"github.com/scjalliance/resourceful/lease"
	"github.com/scjalliance/resourceful/policy"
)

// Message Types
const (
	TypePolicyChange       = "policy.change"
	TypeProcessTermination = "process.termination"
	TypeLicenseLost        = "license.lost"
)

// Message is a UI message.
type Message struct {
	Type         string       `json:"type"`
	PolicyChange PolicyChange `json:"policyChange,omitempty"`
	ProcTerm     ProcTerm     `json:"procTerm,omitempty"`
}

// PolicyChange stores a policy modifcation.
type PolicyChange struct {
	Old policy.Set `json:"old,omitempty"`
	New policy.Set `json:"new,omitempty"`
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
