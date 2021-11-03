//go:build windows
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

	polMutex sync.RWMutex
	policies policy.Set

	cacheMutex sync.Mutex
	cache      policy.Cache
}

// NewPolicyManager returns a new policy manager that will use the given
// client and logger.
//
// If cache is non-nil, it will be used to perform an initial policy load and
// will be updated whenever the manager receives policies from the client.
func NewPolicyManager(client *guardian.Client, cache policy.Cache, logger Logger) *PolicyManager {
	return &PolicyManager{
		client: client,
		cache:  cache,
		logger: logger,
	}
}

// Policies returns the most recently retrieved set of policies.
func (m *PolicyManager) Policies() policy.Set {
	m.polMutex.RLock()
	defer m.polMutex.RUnlock()
	return m.policies
}

// Load causes the policy manager to load policies from its cache, if it
// has one.
func (m *PolicyManager) Load() {
	if m.cache == nil {
		m.log("No policy cache provided")
		return
	}

	m.cacheMutex.Lock()
	updated, err := m.cache.Policies()
	m.cacheMutex.Unlock()
	if err != nil {
		m.log("Failed to load policies from cache: %v", err)
		return
	}

	m.polMutex.Lock()
	previous := m.policies
	m.policies = updated
	m.polMutex.Unlock()

	switch d := len(updated); d {
	case 0:
		m.log("POL: No policies found in the local policy cache")
	case 1:
		m.log("POL: Loading 1 policy from local policy cache")
	default:
		m.log("POL: Loading %d policies from local policy cache", d)
	}

	additions, deletions := previous.Diff(updated)

	for _, pol := range additions {
		m.log("POL: ADD %s: %s", pol.Hash().String(), pol.String())
	}
	for _, pol := range deletions {
		m.log("POL: REM %s: %s", pol.Hash().String(), pol.String())
	}
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

	m.polMutex.Lock()
	previous := m.policies
	m.policies = updated
	m.polMutex.Unlock()

	if m.cache != nil {
		m.cacheMutex.Lock()
		cacheErr := m.cache.SetPolicies(updated)
		m.cacheMutex.Unlock()
		if cacheErr != nil {
			m.log("CACHE UPDATE: %v", cacheErr)
		}
	}

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
