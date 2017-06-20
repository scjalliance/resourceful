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
	defer cancel() // Make sure all of the goroutines shut down when we stop

	ch := m.start()

	go func() {
		<-ctx.Done()
		m.stop()
	}()

	go m.refresh(ctx) // Tells the view models to refresh every second

	var (
		current = Directive{}
		next    = Directive{}
		ok      = true
	)

	for {
		uiCtx, uiCancel := context.WithCancel(context.Background())

		var received sync.WaitGroup
		received.Add(1)

		go func() {
			select {
			case <-uiCtx.Done():
				// The user closed the user interface before we received the next
				// directive, so we default to None.
				next = Directive{}
			case next, ok = <-ch:
				// A new directive has been received
				uiCancel()
			}
			received.Done()
		}()

		err := m.run(uiCtx, current)
		uiCancel()
		received.Wait()

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

	//log.Printf("lui: changing from %s to %s", m.current, t)

	if m.ch == nil {
		return // Closed
	}

	if m.current != t {
		//log.Printf("lui: changed to %s", t)
		m.current = t
		m.ch <- Directive{Type: t, Callback: callback}
	}
}

// CompareAndChange instructs the manager to verify the current interface type,
// then change it if the current value matches what was expected.
func (m *Manager) CompareAndChange(from Type, to Type, callback Callback) {
	if from == to {
		return
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	//log.Printf("lui: changing from %s to %s if %s", m.current, to, from)

	if m.ch == nil {
		return // Closed
	}

	if m.current == from {
		//log.Printf("lui: changed to %s", to)
		m.current = to
		m.ch <- Directive{Type: to, Callback: callback}
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

	//log.Printf("lui: directive %s", d.Type.String())

	switch d.Type {
	default:
		return m.none(ctx, d.Callback)
	case Startup:
		return m.startup(ctx, d.Callback)
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
