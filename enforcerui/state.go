//go:build windows
// +build windows

package enforcerui

import (
	"fmt"

	"github.com/scjalliance/resourceful/lease"
	"github.com/scjalliance/resourceful/policy"
)

// State holds a snapshot of enforcer state that is relevant to the UI.
type State struct {
	Policies policy.Set
	Leases   lease.Set
	Running  []lease.Properties
}

// Summary returns a single-line text summary of the state.
func (s State) Summary() string {
	var summary string

	switch count := len(s.Policies); count {
	case 1:
		summary = "Enforcing 1 policy"
	default:
		summary = fmt.Sprintf("Enforcing %d policies", count)
	}

	switch count := len(s.Running); count {
	case 0:
	case 1:
		summary += "on 1 process"
	default:
		summary += fmt.Sprintf("on %d processes", count)
	}

	return summary
}
