// +build windows

package enforcer

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/scjalliance/resourceful/guardian"
	"github.com/scjalliance/resourceful/lease"
)

// Service is a resourceful policy enforcement service. It watches the local
// set of processes and enforces resourceful policies.
type Service struct {
	client              *guardian.Client
	enforcementInterval time.Duration // Process polling interval
	policyInterval      time.Duration // Policy polling interval
	passive             bool          // Don't kill processes if true
	logger              Logger

	policies *PolicyManager
	sessions *SessionManager
	procs    *ProcessManager

	opMutex  sync.Mutex
	shutdown chan<- struct{} // Close to signal shutdown
	stopped  <-chan struct{} // Closed when shutdown completed
}

// New returns a new policy enforcement service with the given client.
func New(client *guardian.Client, enforcementInterval, policyInterval time.Duration, ui Command, environment lease.Properties, passive bool, logger Logger) *Service {
	var (
		policies = NewPolicyManager(client, logger)
		sessions = NewSessionManager(ui, logger)
		procs    = NewProcessManager(client, environment, passive, sessions, logger)
	)
	return &Service{
		client:              client,
		enforcementInterval: enforcementInterval,
		policyInterval:      policyInterval,
		passive:             passive,
		logger:              logger,
		policies:            policies,
		sessions:            sessions,
		procs:               procs,
	}
}

// Start starts the service if it isn't running.
func (s *Service) Start() error {
	s.opMutex.Lock()
	defer s.opMutex.Unlock()

	if s.shutdown != nil {
		return errors.New("the policy enforcement service is already running")
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

func (s *Service) run(shutdown <-chan struct{}, stopped chan<- struct{}) {
	defer close(stopped)

	var wg sync.WaitGroup
	wg.Add(3)

	// Attempt initial retrieval of policies
	s.policies.Update()

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
				if err := s.procs.Enforce(s.policies.Policies()); err != nil {
					s.log("Enforcement failed: %s", err)
				}
			}
		}
	}()

	// Update policies on an interval
	go func() {
		defer wg.Done()
		defer s.debug("Stopped policy manager")

		policyTimer := time.NewTicker(s.policyInterval)
		defer policyTimer.Stop()

		for {
			select {
			case <-shutdown:
				return
			case <-policyTimer.C:
				if changed := s.policies.Update(); changed {
					s.sessions.UpdatePolicies(s.policies.Policies())
				}
			}
		}
	}()

	// Update sessions on an interval (for now)
	go func() {
		defer wg.Done()

		// Prime the session manager with the current policy set
		s.sessions.UpdatePolicies(s.policies.Policies())

		// Attempt initial scan of sessions
		s.sessions.Scan()

		sessionTimer := time.NewTicker(time.Second * 5)
		defer sessionTimer.Stop()

		for {
			select {
			case <-shutdown:
				return
			case <-sessionTimer.C:
				s.sessions.Scan()
			}
		}
	}()

	// Wait for both goroutines to shutdown
	wg.Wait()

	// Stop all process management
	s.debug("Stopping process manager")
	s.procs.Stop()
	s.debug("Stopped process manager")

	// Stop all session management
	s.debug("Stopping session manager")
	s.sessions.Stop()
	s.debug("Stopped session manager")
}

func (s *Service) log(format string, v ...interface{}) {
	if s.logger == nil {
		return
	}
	s.logger.Log(ServiceEvent{
		Msg: fmt.Sprintf(format, v...),
	})
}

func (s *Service) debug(format string, v ...interface{}) {
	if s.logger == nil {
		return
	}
	s.logger.Log(ServiceEvent{
		Msg:   fmt.Sprintf(format, v...),
		Debug: true,
	})
}
