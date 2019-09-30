// +build windows

package enforcer

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/scjalliance/resourceful/guardian"
	"github.com/scjalliance/resourceful/policy"
)

// PolicyManager maintains a copy of the current policy set.
type PolicyManager struct {
	client *guardian.Client
	logger Logger

	mutex    sync.RWMutex
	policies policy.Set
}

// NewPolicyManager returns a new policy manager that will use the given
// client and logger.
func NewPolicyManager(client *guardian.Client, logger Logger) *PolicyManager {
	return &PolicyManager{
		client: client,
		logger: logger,
	}
}

// Policies returns the most recently retrieved set of policies.
func (m *PolicyManager) Policies() policy.Set {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.policies
}

// Update causes the policy manager to update its policy set.
func (m *PolicyManager) Update(ctx context.Context) (changed, ok bool) {
	const timeout = 10 * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	response, err := m.client.Policies(ctx)
	if err != nil {
		m.log("Failed to retrieve policies: %v", err.Error())
		return false, false
	}

	updated := response.Policies

	m.mutex.Lock()
	previous := m.policies
	m.policies = updated
	m.mutex.Unlock()

	additions, deletions := previous.Diff(updated)
	if len(additions) == 0 && len(deletions) == 0 {
		return false, true
	}

	for _, pol := range additions {
		m.log("POL: ADD %s: %s", pol.Hash().String(), pol.String())
	}
	for _, pol := range deletions {
		m.log("POL: REM %s: %s", pol.Hash().String(), pol.String())
	}

	return true, true
}

func (m *PolicyManager) log(format string, v ...interface{}) {
	if m.logger == nil {
		return
	}
	m.logger.Log(ServiceEvent{
		Msg: fmt.Sprintf(format, v...),
	})
}

func (m *PolicyManager) debug(format string, v ...interface{}) {
	if m.logger == nil {
		return
	}
	m.logger.Log(ServiceEvent{
		Msg:   fmt.Sprintf(format, v...),
		Debug: true,
	})
}
