package leaseui

import (
	"context"
	"sync"
	"time"

	"github.com/scjalliance/resourceful/guardian"
	"github.com/scjalliance/resourceful/lease"
)

// Manager manages a lease user interace.
type Manager struct {
	cfg   Config
	ready chan struct{}

	mutex       sync.RWMutex
	running     bool
	primed      bool
	current     Type
	ch          chan Directive
	lease       lease.Lease
	acquisition guardian.Acquisition
	model       Model
}

// New creates a new lease user interface manager.
func New(cfg Config) *Manager {
	return &Manager{
		cfg:   cfg,
		ready: make(chan struct{}),
		ch:    make(chan Directive),
	}
}

// Run will run the user interface.
//
// Run must be called from the main thread of execution.
func (m *Manager) Run(ctx context.Context, shutdown context.CancelFunc) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	ch := m.start()

	go func() {
		<-ctx.Done()
		m.stop()
	}()

	go m.refresh(ctx)

	var (
		current = Directive{}
		next    = Directive{}
		ok      = true
	)

	for {
		uiCtx, uiCancel := context.WithCancel(context.Background())

		go func() {
			next, ok = <-ch
			uiCancel()
		}()

		err := m.run(uiCtx, current)
		if err != nil {
			return err
		}

		if !ok {
			return nil
		}
		current = next
	}
}

// Change instructs the manager to change the user interface type.
func (m *Manager) Change(t Type, callback Callback) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.ch == nil {
		return // Closed
	}

	if m.current != t {
		m.current = t
		m.ch <- Directive{Type: t, Callback: callback}
	}
}

// Update will update the user interface with the given acquisition.
func (m *Manager) Update(ls lease.Lease, acquisition guardian.Acquisition) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.lease = ls
	m.acquisition = acquisition

	if !m.primed {
		m.primed = true
		close(m.ready)
	}

	if m.model != nil {
		m.model.Update(m.lease, m.acquisition)
	}
}

func (m *Manager) run(ctx context.Context, d Directive) error {
	select {
	case <-ctx.Done():
		return nil
	case <-m.ready:
	}

	switch d.Type {
	default:
		<-ctx.Done()
		return nil
	case Queued:
		return m.queued(ctx, d.Callback)
	case Connected:
		return m.connected(ctx, d.Callback)
	case Disconnected:
		return m.disconnected(ctx, d.Callback)
	}
}

func (m *Manager) start() chan Directive {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.running {
		panic("The lease user interface manager is already running")
	}

	m.running = true

	return m.ch
}

func (m *Manager) stop() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.ch == nil {
		panic("The lease user interface manager has already been stopped")
	}

	close(m.ch)
	m.ch = nil

	if !m.primed {
		m.primed = true
		close(m.ready)
	}
}

func (m *Manager) refresh(ctx context.Context) {
	sleepRound()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.mutex.RLock()
			model := m.model
			m.mutex.RUnlock()
			if model != nil {
				model.Refresh()
			}
		case <-ctx.Done():
			return
		}
	}
}
