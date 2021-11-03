//go:build windows
// +build windows

package enforcer

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"syscall"

	"github.com/gentlemanautomaton/winsession"
	"github.com/gentlemanautomaton/winsession/wtsapi"
	"github.com/scjalliance/resourceful/enforcerui"
	"github.com/scjalliance/resourceful/lease"
	"github.com/scjalliance/resourceful/policy"
)

// SessionID is a session ID.
type SessionID = winsession.ID

// SessionData holds information about a windows session.
type SessionData = winsession.Session

// Session manages communication with an individual session.
type Session struct {
	data   SessionData
	cmd    Command
	logger Logger

	mutex    sync.RWMutex
	stop     context.CancelFunc
	kill     context.CancelFunc
	inbound  chan enforcerui.Message
	outbound chan enforcerui.Message
	stopped  chan struct{}
}

// NewSession creates a communication manager for the given session.
func NewSession(data SessionData, cmd Command, bufSize int, logger Logger) *Session {
	return &Session{
		data:     data,
		cmd:      cmd,
		logger:   logger,
		inbound:  make(chan enforcerui.Message, bufSize),
		outbound: make(chan enforcerui.Message, bufSize),
	}
}

// Send sends msg to s. It returns false if the message buffer for
// s is full.
func (s *Session) Send(msg enforcerui.Message) (ok bool) {
	select {
	case s.outbound <- msg:
		return true
	default:
		s.log("The message buffer is full")
		return false
	}
}

// SendPolicies attempts to send a policy change message to the session.
func (s *Session) SendPolicies(pols policy.Set) (ok bool) {
	return s.Send(enforcerui.Message{
		Type: enforcerui.TypePolicyUpdate,
		Policies: enforcerui.PolicyUpdate{
			New: pols,
		},
	})
}

// SendLeases attempts to send a lease change message to the session.
func (s *Session) SendLeases(leases lease.Set) (ok bool) {
	return s.Send(enforcerui.Message{
		Type: enforcerui.TypeLeaseUpdate,
		Leases: enforcerui.LeaseUpdate{
			New: leases,
		},
	})
}

// SendProcessTermination sends a process termination message to the session.
func (s *Session) SendProcessTermination(name string) bool {
	return s.Send(enforcerui.Message{
		Type: enforcerui.TypeProcessTermination,
		ProcTerm: enforcerui.ProcTerm{
			Name: name,
		},
	})
}

// Connect establishes a connection with s. It launches a ui process as the
// session's user.
func (s *Session) Connect() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.stopped != nil {
		return errors.New("a connection has already been established")
	}

	s.log("Connecting")

	// Acquire a user token for the session
	token, err := wtsapi.QueryUserToken(uint32(s.data.ID))
	if err != nil {
		s.log("Failed to acquire token: %v", err)
		return err
	}

	// Make sure the token is valid for the user we expect
	userName := s.data.Info.UserName
	userDomain := s.data.Info.UserDomain
	if err := validateTokenForUser(token, userName, userDomain); err != nil {
		token.Close()
		s.log("Failed to validate token: %v", err)
		return err
	}

	s.debug("Aquired Token: %s\\%s", userDomain, userName)

	ctx, kill := context.WithCancel(context.Background())

	cmd := exec.CommandContext(ctx, s.cmd.Path, s.cmd.Args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow: true,
		Token:      token,
	}

	writer, err := cmd.StdinPipe()
	if err != nil {
		token.Close()
		kill()
		s.log("Failed to create stdin: %v", err)
		return err
	}

	reader, err := cmd.StdoutPipe()
	if err != nil {
		token.Close()
		kill()
		writer.Close()
		s.log("Failed to create stdin: %v", err)
		return err
	}

	// TODO: Use cmd.String() from Go 1.13
	s.debug("Starting UI Process: %s", strings.Join(cmd.Args, " "))

	if err := cmd.Start(); err != nil {
		token.Close()
		kill()
		writer.Close()
		reader.Close()
		s.log("Failed to start UI process: %v", err)
		return err
	}

	stopped := make(chan struct{})

	s.kill = kill
	s.stopped = stopped

	// Spawn the ui process
	go func(kill context.CancelFunc, stopped chan<- struct{}) {
		defer close(stopped)
		defer kill()
		defer token.Close()
		//defer reader.Close()
		err := cmd.Wait()
		s.mutex.Lock()
		s.kill = nil
		s.stopped = nil
		if err != nil {
			s.log("Disconnected: %v", err)
		} else {
			s.log("Disconnected")
		}
		s.mutex.Unlock()
	}(kill, stopped)

	// Send messages to the process
	go func() {
		defer writer.Close()
		writer := enforcerui.NewWriter(writer)
		for {
			select {
			case <-ctx.Done():
				s.debug("Send: %v", ctx.Err())
				return
			case msg, ok := <-s.outbound:
				if !ok {
					s.debug("Send: EOF")
					return
				}
				s.debug("Send: %s", msg.Type)
				writer.Write(msg)
			}

		}
	}()

	// Receive messages from the process
	go func() {
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			s.debug("Received: %s", scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			s.debug("Receive: %v", err)
		} else {
			s.debug("Receive: EOF")
		}
	}()

	s.log("Process Started")
	return nil
}

// Connected returns true if the session manager is connected.
func (s *Session) Connected() bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.stopped != nil
}

// Disconnect tears down the connection with s.
func (s *Session) Disconnect() error {
	s.mutex.Lock()

	var (
		kill    = s.kill
		stopped = s.stopped
	)

	if stopped == nil {
		s.mutex.Unlock()
		return errors.New("a connection to the session has not been established")
	}

	s.log("Disconnecting")

	s.mutex.Unlock()

	kill()
	<-stopped

	return nil
}

func (s *Session) log(format string, v ...interface{}) {
	if s.logger != nil {
		s.logger.Log(SessionEvent{
			SessionID:     uint32(s.data.ID),
			WindowStation: s.data.WindowStation,
			SessionUser:   s.data.Info.User(),
			Msg:           fmt.Sprintf(format, v...),
		})
	}
}

func (s *Session) debug(format string, v ...interface{}) {
	if s.logger != nil {
		s.logger.Log(SessionEvent{
			SessionID:     uint32(s.data.ID),
			WindowStation: s.data.WindowStation,
			SessionUser:   s.data.Info.User(),
			Msg:           fmt.Sprintf(format, v...),
			Debug:         true,
		})
	}
}
