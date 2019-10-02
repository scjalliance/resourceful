package main

import (
	"fmt"
	"time"

	"github.com/scjalliance/resourceful/lease"
	"github.com/scjalliance/resourceful/lease/leaseutil"
	"github.com/scjalliance/resourceful/policy"
)

// A StatRecipient is capable of receiving resource statistics.
type StatRecipient interface {
	SendResource(resource string, stats ResourceStats) error
	SendUser(resource, user string, count uint, t time.Time) error
}

// ResourceStatsMap holds resource statistics for a set of resources.
type ResourceStatsMap map[string]ResourceStats

// ResourceStats hold statistics for a particular resource
type ResourceStats struct {
	Time     time.Time
	Consumed uint
	Limit    uint
	Active   uint
	Released uint
	Queued   uint
	Users    UserStatsMap
}

// UserStatsMap holds resource statistics for a user.
type UserStatsMap map[string]uint

// Contains returns true if m contains user.
func (m UserStatsMap) Contains(user string) bool {
	if m == nil {
		return false
	}
	_, found := m[user]
	return found
}

// StatManager manages resources statistics.
type StatManager struct {
	recipient StatRecipient
	last      ResourceStatsMap // The last set of statistics that were collected
}

// NewStatManager returns a new statistics manager.
func NewStatManager(r StatRecipient) *StatManager {
	return &StatManager{recipient: r}
}

// Init initializes the stat manager.
func (m *StatManager) Init(polProv policy.Provider, leaseProv lease.Provider) error {
	stats, err := m.collect(polProv, leaseProv, false)
	if err != nil {
		return err
	}
	m.last = stats
	return nil
}

// CollectAndSend collects statistics from the given providers and sends them
// to the manager's stat recipient.
func (m *StatManager) CollectAndSend(polProv policy.Provider, leaseProv lease.Provider) error {
	// Collect the current values
	current, err := m.collect(polProv, leaseProv, true)
	if err != nil {
		return err
	}

	// Any resources or users that are no longer present need to have a final
	// set of zeroed values sent
	for resource, last := range m.last {
		if current, exists := current[resource]; !exists {
			removal := ResourceStats{
				Limit: last.Limit,
				Users: make(UserStatsMap, len(last.Users)),
			}
			for user := range last.Users {
				removal.Users[user] = 0
			}
			if err := m.recipient.SendResource(resource, removal); err != nil {
				return fmt.Errorf("failed to remove expired stats for %s: %v", resource, err)
			}
		} else {
			for user, count := range last.Users {
				if count == 0 {
					// Already zeroed in the last submission
					continue
				}
				if _, found := current.Users[user]; found {
					// User is still active
					continue
				}
				current.Users[user] = 0
			}
		}
	}

	// Any users that were previously absent need to have a zero value sent
	// prior to the current value
	for resource, current := range current {
		last, exists := m.last[resource]
		for user := range current.Users {
			if exists && last.Users.Contains(user) {
				continue // Not previously absent
			}
			m.recipient.SendUser(resource, user, 0, current.Time.Add(-time.Minute))
		}
	}

	// Send the current values
	for resource, stats := range current {
		if err := m.recipient.SendResource(resource, stats); err != nil {
			return fmt.Errorf("failed to send stats for %s: %v", resource, err)
		}
	}

	m.last = current

	return nil
}

func (m *StatManager) collect(polProv policy.Provider, leaseProv lease.Provider, refresh bool) (stats ResourceStatsMap, err error) {
	policies, err := polProv.Policies()
	if err != nil {
		return nil, err
	}

	resources, err := leaseProv.LeaseResources()
	if err != nil {
		return nil, err
	}

	stats = make(ResourceStatsMap, len(resources))

	for _, resource := range resources {
		// Collect policy settings for the resource
		policies := policies.MatchResource(resource)
		strat := policies.Strategy()
		limit := policies.Limit()

		// Collect a view of the current lease set
		now := time.Now()
		revision, leases, err := leaseProv.LeaseView(resource)
		if err != nil {
			return nil, err
		}

		if refresh {
			// Purge expired leases
			tx := lease.NewTx(resource, revision, leases)
			leaseutil.Refresh(tx, now)

			// Use the refreshed lease set
			leases = tx.Leases()
		}

		// Collect statistics
		data := leases.Stats()

		// Translate users into user account names
		users := make(UserStatsMap)
		for user, count := range data.Users(strat) {
			if props := leases.User(user).Property("user.account"); len(props) > 0 {
				name := props[0]
				users[name] = count
			} else {
				users[user] = count
			}
		}

		// Include the assembled stats
		stats[resource] = ResourceStats{
			Time:     now,
			Consumed: data.Consumed(strat),
			Limit:    limit,
			Active:   data.Active(strat),
			Released: data.Released(strat),
			Queued:   data.Queued(strat),
			Users:    users,
		}
	}

	return stats, nil
}
