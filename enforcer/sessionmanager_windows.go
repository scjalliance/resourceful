// +build windows

package enforcer

import (
	"fmt"
	"sync"
	"time"

	"github.com/gentlemanautomaton/winsession"
	"github.com/gentlemanautomaton/winsession/connstate"
	"github.com/scjalliance/resourceful/enforcerui"
	"github.com/scjalliance/resourceful/lease"
	"github.com/scjalliance/resourceful/policy"
)

// sessionAttempt records information about the last time a connection
// was attempted with a session
type sessionAttempt struct {
	Attempt time.Time
	Wait    time.Duration
}

// SessionManager manages communication with any number of windows sessions.
//
// Its zero value is ready for use.
type SessionManager struct {
	command Command
	logger  Logger

	mutex     sync.RWMutex
	managed   map[SessionID]*Session
	attempted map[SessionID]sessionAttempt
	pols      policy.Set
	leases    lease.Set
}

// NewSessionManager returns a new session manager that is ready for use.
//
// The given command describes what process to launch within each session.
func NewSessionManager(cmd Command, logger Logger) *SessionManager {
	return &SessionManager{
		command:   cmd,
		logger:    logger,
		managed:   make(map[SessionID]*Session, 8),
		attempted: make(map[SessionID]sessionAttempt, 8),
	}
}

// Session returns the managed session with id, if one exists.
func (m *SessionManager) Session(id SessionID) *Session {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.managed[id]
}

// Scan causes the service to rescan the current sessions.
func (m *SessionManager) Scan() error {
	sessions, err := winsession.Local.Sessions(
		winsession.Exclude(winsession.MatchID(0)),
		winsession.Include(winsession.MatchState(connstate.Active, connstate.Disconnected)),
		winsession.CollectSessionInfo,
	)
	if err != nil {
		return nil
	}

	now := time.Now()
	scanned := make(map[winsession.ID]struct{}, len(sessions))

	m.mutex.Lock()
	defer m.mutex.Unlock()

	var pending []SessionData
	for _, session := range sessions {
		id := session.ID
		scanned[id] = struct{}{} // Record the ID in the map of scanned sessions

		// Don't re-process sessions that are already managed
		if _, exists := m.managed[id]; exists {
			continue
		}

		// Wait before retrying connections that previously failed
		if attempt, ok := m.attempted[id]; ok {
			if now.Sub(attempt.Attempt) < attempt.Wait {
				continue
			}
		}

		pending = append(pending, session)
	}

	// Bookkeeping for dead sessions
	for id, session := range m.managed {
		if _, exists := scanned[id]; !exists || !session.Connected() {
			go session.Disconnect()
			delete(m.managed, id)
			m.log("Stopped management of session %d", id)
		}
	}

	for id := range m.attempted {
		if _, exists := scanned[id]; !exists {
			delete(m.attempted, id)
		}
	}

	// Establish a connection with newly discovered sessions
	for _, data := range pending {
		data := data
		id := data.ID
		session := NewSession(data, m.command, 64, m.logger)
		m.managed[id] = session
		go func() {
			now := time.Now()
			err := session.Connect()

			m.mutex.Lock()
			pols := m.pols
			leases := m.leases
			if err != nil {
				if m.managed[id] == session {
					delete(m.managed, id)
					m.attempted[id] = nextSessionAttempt(m.attempted[id], now)
				}
			} else {
				delete(m.attempted, id)
			}
			m.mutex.Unlock()

			if err != nil {
				return
			}

			// Send the current policy set to the session
			session.SendPolicies(pols)

			// Send the current lease set to the session
			session.SendLeases(leases)
		}()
	}

	return nil
}

// UpdatePolicies sends the policy set to the ui process running in each
// session.
func (m *SessionManager) UpdatePolicies(pols policy.Set) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.pols = pols

	for _, session := range m.managed {
		session.SendPolicies(pols)
	}
}

// UpdateLeases sends the lease set to the ui process running in each
// session.
func (m *SessionManager) UpdateLeases(leases lease.Set) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.leases = leases

	for _, session := range m.managed {
		session.SendLeases(leases)
	}
}

// Send sends the given message to the ui process running in each session.
func (m *SessionManager) Send(msg enforcerui.Message) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	for _, session := range m.managed {
		session.Send(msg)
	}
}

// Stop causes the session manager to stop all session management.
func (m *SessionManager) Stop() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	for id, session := range m.managed {
		session.Disconnect()
		delete(m.managed, id)
		m.log("Stopped management of session %d", session.data.ID)
	}
}

func (m *SessionManager) log(format string, v ...interface{}) {
	if m.logger == nil {
		return
	}
	m.logger.Log(ServiceEvent{
		Msg: fmt.Sprintf(format, v...),
	})
}

func (m *SessionManager) debug(format string, v ...interface{}) {
	if m.logger == nil {
		return
	}
	m.logger.Log(ServiceEvent{
		Msg:   fmt.Sprintf(format, v...),
		Debug: true,
	})
}

// nextSessionAttempt calculates the retry backoff after failed session
// connection attempts.
func nextSessionAttempt(last sessionAttempt, now time.Time) (next sessionAttempt) {
	const (
		minWait = 30 * time.Second
		maxWait = 5 * time.Minute
	)

	if last.Attempt.IsZero() {
		return sessionAttempt{
			Attempt: now,
			Wait:    minWait,
		}
	}

	wait := last.Wait
	wait *= 2
	if wait > maxWait {
		wait = maxWait
	}
	return sessionAttempt{
		Attempt: now,
		Wait:    wait,
	}
}
