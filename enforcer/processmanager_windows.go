// +build windows

package enforcer

import (
	"fmt"
	"strings"
	"sync"

	"github.com/scjalliance/resourceful/guardian"
	"github.com/scjalliance/resourceful/lease"
	"github.com/scjalliance/resourceful/policy"
)

// ProcessManager enforces a set of policies on local processes.
type ProcessManager struct {
	client   *guardian.Client
	hostname string
	passive  bool // Don't kill processes if true
	sessions *SessionManager
	logger   Logger

	mutex       sync.RWMutex
	managed     map[UniqueID]lease.Instance
	skipped     map[UniqueID]struct{}
	invocations map[lease.Instance]*Invocation // Keyed by resource consumed
}

// NewProcessManager returns a new process manager that is ready for use.
func NewProcessManager(client *guardian.Client, hostname string, passive bool, sessions *SessionManager, logger Logger) *ProcessManager {
	return &ProcessManager{
		client:      client,
		hostname:    hostname,
		passive:     passive,
		sessions:    sessions,
		logger:      logger,
		managed:     make(map[UniqueID]lease.Instance, 8),
		skipped:     make(map[UniqueID]struct{}),
		invocations: make(map[lease.Instance]*Invocation, 8),
	}
}

// Enforce causes the process manager to enforce the given policy set.
func (m *ProcessManager) Enforce(policies policy.Set) error {
	procs, err := Scan(policies)
	if err != nil {
		return err
	}

	scanned := make(map[UniqueID]struct{}, len(procs))

	m.mutex.Lock()
	defer m.mutex.Unlock()

	var pending []ProcessData
	for _, proc := range procs {
		id := proc.UniqueID()
		scanned[id] = struct{}{} // Record the ID in the map of scanned procs

		// Don't manage blacklisted processes
		if Blacklisted(proc) {
			if instance, exists := m.managed[id]; exists {
				// Stop the invocation
				if inv := m.invocations[instance]; inv != nil {
					inv.Stop()
					delete(m.invocations, instance)
					m.log("Stopped management of blacklisted invocation %s (%s)", instance, proc.Name)
				}
				// Remove from managed and add to skipped
				delete(m.managed, id)
				m.skipped[id] = struct{}{}
				m.log("Stopped management of blacklisted process %s (%s)", id, proc.Name)
			} else if _, exists := m.skipped[id]; !exists {
				// Add to skipped
				m.log("Skipped management of blacklisted process %s (%s)", id, proc.Name)
				m.skipped[id] = struct{}{}
			}
			continue
		} else {
			if _, exists := m.skipped[id]; exists {
				// Remove from skipped
				delete(m.skipped, id)
			}
		}

		// Don't re-manage processes that are already managed
		if _, exists := m.managed[id]; exists {
			continue
		}

		// If it matches a policy add it to the pending slice
		matches := policies.Match(Properties(proc, m.hostname))
		if len(matches) > 0 {
			pending = append(pending, proc)
		}
	}

	// Bookkeeping for dead processes
	for id, instance := range m.managed {
		if _, exists := scanned[id]; !exists {
			if inv := m.invocations[instance]; inv != nil {
				// TODO: Ask the invocation whether it's still alive. Check
				// if it has stale policies. Remove if necessary.
				if inv.Done() {
					inv.Stop()
					delete(m.invocations, instance)
					m.debug("Stopped management of invocation %s", instance.ID)
				}
			}
			delete(m.managed, id)
			m.debug("Stopped management of process %s", id)
		}
	}

	for id := range m.skipped {
		if _, exists := scanned[id]; !exists {
			delete(m.skipped, id)
		}
	}

	// Bookkeeping for dead invocations
	for instance, inv := range m.invocations {
		if !inv.Done() {
			continue
		}
		inv.Stop()
		delete(m.invocations, instance)
		m.debug("Stopped management of invocation %s", instance.ID)
	}

	// Exit early if nothing is pending
	if len(pending) == 0 {
		return nil
	}

	// Begin management of newly discovered processes
	if len(pending) == 1 {
		m.debug("Enforcement found 1 new process: %s", pending[0].ID)
	} else {
		ids := make([]string, len(pending))
		for i := range pending {
			ids[i] = pending[i].ID.String()
		}
		m.debug("Enforcement found %d new processes: %s", len(pending), strings.Join(ids, ", "))
	}

	for _, proc := range pending {
		id := proc.UniqueID()

		// Open a reference to the process.
		process, err := NewProcess(proc, m.passive, m.logger)
		if err != nil {
			// TODO: Retry on some interval with backoff so we don't spam the logs
			m.log("Unable to manage process %s: %v", id, err)
			continue
		}

		// Look for an existing invocation that can absorb the process
		absorbed := false
		for instance, inv := range m.invocations {
			absorbed = inv.Absorb(process)
			if absorbed {
				m.managed[id] = instance
				m.debug("Process %s absorbed into instance %s", id, instance.ID)
				break
			}
		}
		if absorbed {
			continue
		}

		// Create a new invocation
		instance := Instance(m.hostname, proc, NewInstanceID(proc))
		m.debug("Started management of invocation %s", instance.ID)
		m.debug("Started management of process %s", id)
		invocation := NewInvocation(m.client, instance, process, m.sessions.Session(SessionID(proc.SessionID)), m.logger)
		m.invocations[instance] = invocation
		m.managed[id] = instance
	}

	return nil
}

// Stop causes the process manager to stop all process management.
func (m *ProcessManager) Stop() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	for instance, inv := range m.invocations {
		m.log("Stopping management of invocation %s", instance.ID)
		inv.Stop()
		delete(m.invocations, instance)
		m.log("Stopped management of invocation %s", instance.ID)
	}
	for id := range m.managed {
		delete(m.managed, id)
	}
}

func (m *ProcessManager) log(format string, v ...interface{}) {
	if m.logger == nil {
		return
	}
	m.logger.Log(ServiceEvent{
		Msg: fmt.Sprintf(format, v...),
	})
}

func (m *ProcessManager) debug(format string, v ...interface{}) {
	if m.logger == nil {
		return
	}
	m.logger.Log(ServiceEvent{
		Msg:   fmt.Sprintf(format, v...),
		Debug: true,
	})
}
