// +build windows

package enforcer

import (
	"errors"
	"sync"
	"time"

	"github.com/scjalliance/resourceful/guardian"
	"github.com/scjalliance/resourceful/policy"
)

// Service is a resourceful process monitoring service. It watches the local
// set of processes and enforces resourceful policies.
type Service struct {
	client              *guardian.Client
	enforcementInterval time.Duration // Process polling interval
	policyInterval      time.Duration // Configuration polling interval
	hostname            string
	passive             bool // Don't kill processes if true
	logger              Logger

	polMutex sync.RWMutex
	policies policy.Set

	managedMutex sync.RWMutex
	managed      map[UniqueID]*ManagedProcess
	skipped      map[UniqueID]struct{}

	opMutex  sync.Mutex
	shutdown chan<- struct{} // Close to signal shutdown
	stopped  <-chan struct{} // Closed when shutdown completed
}

// New returns a new policy monitor service with the given client.
func New(client *guardian.Client, enforcementInterval, policyInterval time.Duration, hostname string, passive bool, logger Logger) *Service {
	return &Service{
		client:              client,
		enforcementInterval: enforcementInterval,
		policyInterval:      policyInterval,
		hostname:            hostname,
		passive:             passive,
		logger:              logger,
		managed:             make(map[UniqueID]*ManagedProcess, 8),
		skipped:             make(map[UniqueID]struct{}),
	}
}

// Start starts the service if it isn't running.
func (s *Service) Start() error {
	s.opMutex.Lock()
	defer s.opMutex.Unlock()

	if s.shutdown != nil {
		return errors.New("the policy monitor service is already running")
	}

	shutdown := make(chan struct{})
	s.shutdown = shutdown

	stopped := make(chan struct{})
	s.stopped = stopped

	go s.run(shutdown, stopped)

	return nil
}

// Stop stops the service if it's running.
func (s *Service) Stop() {
	s.opMutex.Lock()
	defer s.opMutex.Unlock()

	if s.shutdown == nil {
		return
	}

	close(s.shutdown)
	s.shutdown = nil

	<-s.stopped
	s.stopped = nil
}

// Policies returns the most recently retrieved set of policies.
func (s *Service) Policies() policy.Set {
	s.polMutex.RLock()
	defer s.polMutex.RUnlock()
	return s.policies
}

// UpdatePolicies causes the service to update its policies.
func (s *Service) UpdatePolicies() {
	response, err := s.client.Policies()
	if err != nil {
		s.log("Failed to retrieve policies: %v", err.Error())
		return
	}

	s.polMutex.Lock()
	additions, deletions := s.policies.Diff(response.Policies)
	s.policies = response.Policies
	s.polMutex.Unlock()

	for _, pol := range additions {
		s.log("POL: ADD %s: %s", pol.Hash().String(), pol.String())
	}
	for _, pol := range deletions {
		s.log("POL: REM %s: %s", pol.Hash().String(), pol.String())
	}
}

func (s *Service) manage(p Process) {
	s.log("Starting management of %s", Subject(s.hostname, p))

	id := p.UniqueID()

	s.managedMutex.Lock()
	defer s.managedMutex.Unlock()

	if _, exists := s.managed[id]; exists {
		// Already managed
		return
	}

	mp, err := Manage(s.client, s.hostname, p, s.passive)
	if err != nil {
		s.log("Unable to manage process %s: %v", id, err)
	}
	s.managed[id] = mp
}

func (s *Service) run(shutdown <-chan struct{}, stopped chan<- struct{}) {
	defer close(stopped)

	var wg sync.WaitGroup
	wg.Add(2)

	// Perform enforcement on an interval
	go func() {
		defer wg.Done()

		enforceTimer := time.NewTicker(s.enforcementInterval)
		defer enforceTimer.Stop()

		for {
			select {
			case <-shutdown:
				return
			case <-enforceTimer.C:
				if err := s.Enforce(); err != nil {
					s.log("Enforcement failed: %s", err)
				}
			}
		}
	}()

	// Update policies on an interval
	go func() {
		defer wg.Done()

		// Attempt initial retrieval of policies
		s.UpdatePolicies()

		policyTimer := time.NewTicker(s.policyInterval)
		defer policyTimer.Stop()

		for {
			select {
			case <-shutdown:
				return
			case <-policyTimer.C:
				s.UpdatePolicies()
			}
		}
	}()

	// Wait for both goroutines to shutdown
	wg.Wait()

	// Stop all process management
	s.managedMutex.Lock()
	defer s.managedMutex.Unlock()
	for id, mp := range s.managed {
		mp.Stop()
		delete(s.managed, id)
		s.log("Stopped management of %s", Subject(s.hostname, mp.proc))
	}
}

// Enforce causes the service to enforce the current policy set.
func (s *Service) Enforce() error {
	policies := s.Policies()

	procs, err := Scan(policies)
	if err != nil {
		return err
	}

	scanned := make(map[UniqueID]struct{}, len(procs))

	s.managedMutex.Lock()
	defer s.managedMutex.Unlock()

	var pending []Process
	for _, proc := range procs {
		id := proc.UniqueID()
		scanned[id] = struct{}{} // Record the ID in the map of scanned procs

		// Don't manage blacklisted processes
		if Blacklisted(proc) {
			if mp, exists := s.managed[id]; exists {
				// Remove from managed and add to skipped
				mp.Stop()
				delete(s.managed, id)
				s.skipped[id] = struct{}{}
				s.log("Stopped management of blacklisted process: %s", Subject(s.hostname, proc))
			} else if _, exists := s.skipped[id]; !exists {
				// Add to skipped
				s.log("Skipped management of blacklisted process: %s", Subject(s.hostname, proc))
				s.skipped[id] = struct{}{}
			}
			continue
		} else {
			if _, exists := s.skipped[id]; exists {
				// Remove from skipped
				delete(s.skipped, id)
			}
		}

		// Don't re-process processes that are already managed
		if _, exists := s.managed[id]; exists {
			continue
		}

		// If it matches a policy add it to the pending slice
		subject := Subject(s.hostname, proc)
		env := Env(s.hostname, proc)
		matches := policies.Match(subject.Resource, subject.Consumer, env)
		if len(matches) > 0 {
			pending = append(pending, proc)
		}
	}

	// Bookkeeping for dead processes
	for id := range s.managed {
		if _, exists := scanned[id]; !exists {
			proc := s.managed[id].proc
			s.managed[id].Stop() // If the process died this is redundant, but if it no longer needs a lease this cleans up the manager
			delete(s.managed, id)
			s.log("Stopped management of %s", Subject(s.hostname, proc))
		}
	}

	for id := range s.skipped {
		if _, exists := scanned[id]; !exists {
			delete(s.skipped, id)
		}
	}

	// Exit early if nothing is pending
	if len(pending) == 0 {
		return nil
	}

	// Begin management of newly discovered processes
	s.log("Enforcement found %d new processes", len(pending))

	for _, proc := range pending {
		id := proc.UniqueID()
		mp, err := Manage(s.client, s.hostname, proc, s.passive)
		if err != nil {
			s.log("Unable to manage process %s: %v", id, err)
			continue
		}
		s.managed[id] = mp

		s.log("Started management of %s", Subject(s.hostname, proc))
	}

	return nil
}

// TODO: Accept an event ID or event interface?
func (s *Service) log(format string, v ...interface{}) {
	// TODO: Try casting s.logger to a different interface so that we can log event IDs?
	if s.logger != nil {
		s.logger.Printf(format, v...)
	}
}
