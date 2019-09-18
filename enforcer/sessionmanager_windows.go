// +build windows

package enforcer

import (
	"fmt"
	"sync"

	"github.com/gentlemanautomaton/winsession"
	"github.com/gentlemanautomaton/winsession/connstate"
)

// SessionManager manages communication with any number of windows sessions.
//
// Its zero value is ready for use.
type SessionManager struct {
	command Command
	logger  Logger

	mutex   sync.RWMutex
	managed map[SessionID]*Session
}

// NewSessionManager returns a new session manager that is ready for use.
//
// The given command describes what process to launch within each session.
func NewSessionManager(cmd Command, logger Logger) *SessionManager {
	return &SessionManager{
		command: cmd,
		logger:  logger,
		managed: make(map[SessionID]*Session, 8),
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

	// Establish a connection with newly discovered sessions
	for _, data := range pending {
		data := data
		session := NewSession(data, m.command, 64, m.logger)
		m.managed[data.ID] = session
		go func() {
			if err := session.Connect(); err != nil {
				// TODO: Try again?
				m.mutex.Lock()
				defer m.mutex.Unlock()
				if m.managed[data.ID] == session {
					delete(m.managed, data.ID)
				}
			}

			// Send the current policy set to the session
			/*
				mgr.Send(enforcerui.Message{
					Type: "policy.change",
					PolicyChange: enforcerui.PolicyChange{
						New: m.Policies(),
					},
				})
			*/
		}()
	}

	return nil
}

/*
// Send sends the given message to the ui process running in each session.
func (s *Service) Send(msg enforcerui.Message) {
	s.sessionMutex.Lock()
	defer s.sessionMutex.Unlock()
	for _, session := range s.sessions {
		session.Send(msg)
	}
}
*/

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
