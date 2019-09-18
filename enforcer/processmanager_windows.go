// +build windows

package enforcer

import (
	"fmt"
	"sync"

	"github.com/scjalliance/resourceful/guardian"
	"github.com/scjalliance/resourceful/policy"
)

// ProcessManager manages enforcement a process for which policies are being enforced.
type ProcessManager struct {
	client   *guardian.Client
	hostname string
	passive  bool // Don't kill processes if true
	logger   Logger

	mutex   sync.RWMutex
	managed map[UniqueID]*Process
	skipped map[UniqueID]struct{}
}

// NewProcessManager returns a new process manager that is ready for use.
func NewProcessManager(client *guardian.Client, hostname string, passive bool, logger Logger) *ProcessManager {
	return &ProcessManager{
		client:   client,
		hostname: hostname,
		passive:  passive,
		logger:   logger,
		managed:  make(map[UniqueID]*Process, 8),
		skipped:  make(map[UniqueID]struct{}),
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
			subject := Instance(m.hostname, proc, id.String())
			if mp, exists := m.managed[id]; exists {
				// Remove from managed and add to skipped
				mp.Stop()
				delete(m.managed, id)
				m.skipped[id] = struct{}{}
				m.log("Stopped management of blacklisted process: %s", subject)
			} else if _, exists := m.skipped[id]; !exists {
				// Add to skipped
				m.log("Skipped management of blacklisted process: %s", subject)
				m.skipped[id] = struct{}{}
			}
			continue
		} else {
			if _, exists := m.skipped[id]; exists {
				// Remove from skipped
				delete(m.skipped, id)
			}
		}

		// Don't re-process processes that are already managed
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
	for id := range m.managed {
		if _, exists := scanned[id]; !exists {
			proc := m.managed[id].data
			m.managed[id].Stop() // If the process died this is redundant, but if it no longer needs a lease this cleans up the manager
			delete(m.managed, id)
			m.log("Stopped management of %s", Instance(m.hostname, proc, id.String()))
		}
	}

	for id := range m.skipped {
		if _, exists := scanned[id]; !exists {
			delete(m.skipped, id)
		}
	}

	// Exit early if nothing is pending
	if len(pending) == 0 {
		return nil
	}

	// Begin management of newly discovered processes
	m.log("Enforcement found %d new processes", len(pending))

	for _, proc := range pending {
		id := proc.UniqueID()
		instance := Instance(m.hostname, proc, id.String())
		mp, err := Manage(m.client, proc, instance, m.passive, m.logger)
		if err != nil {
			m.log("Unable to manage process %s: %v", id, err)
			continue
		}
		m.managed[id] = mp

		m.log("Started management of %s", instance)
	}

	return nil
}

// Stop causes the process manager to stop all process management.
func (m *ProcessManager) Stop() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	for id, process := range m.managed {
		process.Stop()
		delete(m.managed, id)
		m.log("Stopped management of %s", Instance(m.hostname, process.data, id.String()))
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
