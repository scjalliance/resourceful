package guardian

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/scjalliance/resourceful/guardian/transport"
	"github.com/scjalliance/resourceful/lease"
	"github.com/scjalliance/resourceful/lease/leaseutil"
	"github.com/scjalliance/resourceful/policy"
	"github.com/scjalliance/resourceful/strategy"
)

// ServerConfig is the configuration for a resourceful guardian server.
type ServerConfig struct {
	ListenSpec      string
	PolicyProvider  policy.Provider
	LeaseProvider   lease.Provider
	ShutdownTimeout time.Duration // Time allowed to the HTTP server to perform a graceful shutdown
	Logger          *log.Logger
}

// Server is a resourceful guardian HTTP server that coordinates locks on
// finite resources.
type Server struct {
	ServerConfig
}

// NewServer creates a new resourceful guardian server that will handle HTTP
// requests.
func NewServer(cfg ServerConfig) *Server {
	return &Server{
		ServerConfig: cfg,
	}
}

// Run will create and run a resourceful guardian server until the provided
// context is canceled.
func Run(ctx context.Context, cfg ServerConfig) (err error) {
	s := NewServer(cfg)
	return s.Run(ctx)
}

// Run will start the server and let it run until the context is cancelled.
//
// If the server cannot be started it will return an error immediately.
func (s *Server) Run(ctx context.Context) (err error) {
	s.Purge()
	defer s.Purge()
	printf(s.Logger, "Starting HTTP listener on %s", s.ListenSpec)

	listener, err := net.Listen("tcp", s.ListenSpec)
	if err != nil {
		s.Logger.Printf("Error creating HTTP listener on %s: %v", s.ListenSpec, err)
		return
	}

	mux := http.NewServeMux()
	mux.Handle("/health", http.HandlerFunc(s.healthHandler))
	mux.Handle("/policies", http.HandlerFunc(s.policiesHandler))
	mux.Handle("/leases", http.HandlerFunc(s.leasesHandler))
	mux.Handle("/acquire", http.HandlerFunc(s.acquireHandler))
	mux.Handle("/release", http.HandlerFunc(s.releaseHandler))

	srv := &http.Server{
		ReadTimeout:    60 * time.Second,
		WriteTimeout:   60 * time.Second,
		MaxHeaderBytes: 1 << 16,
		Handler:        mux,
	}

	result := make(chan error)

	go func() {
		result <- srv.Serve(listener)
		close(result)
	}()

	select {
	case err = <-result:
		printf(s.Logger, "Stopped HTTP listener on %s due to error: %v", s.ListenSpec, err)
		return
	case <-ctx.Done():
	}

	printf(s.Logger, "Stopping HTTP listener on %s due to shutdown signal", s.ListenSpec)
	shutdownCtx, cancel := context.WithTimeout(context.Background(), s.ShutdownTimeout)
	defer cancel()
	srv.Shutdown(shutdownCtx)

	err = <-result

	printf(s.Logger, "Stopped HTTP listener on %s", s.ListenSpec)
	return
}

// healthHandler will return the condition of the server.
func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	response := transport.HealthResponse{OK: true}
	data, err := json.Marshal(response)
	if err != nil {
		http.Error(w, "Unable to marshal health response", http.StatusBadRequest)
		return
	}
	fmt.Fprintf(w, string(data))
}

// policiesHandler will return the complete set of policies.
func (s *Server) policiesHandler(w http.ResponseWriter, r *http.Request) {
	policies, err := s.PolicyProvider.Policies()
	if err != nil {
		printf(s.Logger, "Policy retrieval failed: %v\n", err)
		http.Error(w, "Unable to retrieve policies", http.StatusInternalServerError)
		return
	}

	response := transport.PoliciesResponse{
		Policies: policies,
	}
	data, err := json.MarshalIndent(response, "", "\t")
	if err != nil {
		http.Error(w, "Unable to marshal policies", http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, string(data))
}

// leasesHandler will return the set of leases for a particular resource.
func (s *Server) leasesHandler(w http.ResponseWriter, r *http.Request) {
	req, err := parseRequest(r)
	if err != nil {
		err = fmt.Errorf("unable to parse request: %v", err)
		printf(s.Logger, "Bad leases request: %v\n", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	revision, leases, err := s.LeaseProvider.LeaseView(req.Resource)
	if err != nil {
		printf(s.Logger, "Lease retrieval failed: %v\n", err)
	}
	now := time.Now()
	tx := lease.NewTx(req.Resource, revision, leases)
	leaseutil.Refresh(tx, now)

	response := transport.LeasesResponse{
		Request: req,
		Leases:  tx.Leases(),
	}
	data, err := json.MarshalIndent(response, "", "\t")
	if err != nil {
		http.Error(w, "Unable to marshal health response", http.StatusBadRequest)
		return
	}
	fmt.Fprintf(w, string(data))
}

// acquireHandler will attempt to acquire a lease for the specified resource.
func (s *Server) acquireHandler(w http.ResponseWriter, r *http.Request) {
	req, policies, err := s.initRequest(r)
	if err != nil {
		printf(s.Logger, "Bad acquire request: %v\n", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// TODO: When the matching policy set dictates consumption of more than
	// one resource, produce a lease for each one.

	// Determine what resource the lease should be issued for
	if resource := policies.Resource(); resource != req.Resource {
		if req.Resource != "" {
			// This is a renewal and the current set of policies dictate
			// use of a different resource than before, or none at all.
			// Attempt to release the previously held resource before
			// acquiring the new one.
			if err := s.release(req.Subject, policies); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		}
		req.Resource = resource
	}

	// Return HTTP 204 if there are no matching policies
	if req.Resource == "" {
		w.Header().Set("Cache-Control", "max-age=300")
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Merge the client-provided properties with the policy-provided properties
	props := lease.MergeProperties(req.Properties, policies.Properties())

	prefix := req.Subject.String()

	printf(s.Logger, "%s: Lease acquisition requested\n", prefix)

	ls, leases, err := s.acquire(req.Subject, props, policies)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	response := transport.AcquireResponse{
		Request: req,
		Lease:   ls,
		Leases:  leases,
	}

	data, err := json.Marshal(response)
	if err != nil {
		printf(s.Logger, "%s: Failed to marshal response: %v\n", prefix, err)
		http.Error(w, "Failed to marshal response", http.StatusBadRequest)
		return
	}

	fmt.Fprintf(w, string(data))
}

// acquireHandler will attempt to acquire a lease for the specified resource.
func (s *Server) acquire(subject lease.Subject, props lease.Properties, policies policy.Set) (ls lease.Lease, leases lease.Set, err error) {
	prefix := subject.String()

	strat := policies.Strategy()
	limit := policies.Limit()
	duration := policies.Duration()
	decay := policies.Decay()
	refresh := policies.Refresh()

	mode := "Creation" // Only used for logging

	for attempt := 0; attempt < 5; attempt++ {
		var revision uint64

		revision, leases, err = s.LeaseProvider.LeaseView(subject.Resource)
		if err != nil {
			printf(s.Logger, "%s: Lease retrieval failed: %v\n", prefix, err)
		}
		now := time.Now()

		ls = lease.Lease{
			Subject:    subject,
			Started:    now,
			Renewed:    now,
			Strategy:   strat,
			Limit:      limit,
			Duration:   duration,
			Decay:      decay,
			Refresh:    refresh,
			Properties: props,
		}

		if ls.Refresh.Active != 0 {
			if ls.Duration <= ls.Refresh.Active {
				printf(s.Logger, "%s: The lease policy specified an active refresh interval of %s for a lease with a duration of %s. The refresh interval will be overridden.\n", prefix, ls.Refresh.Active.String(), ls.Duration.String())
				ls.Refresh.Active = 0 // Use the default refresh rate instead of nonsense
			}
		}
		if ls.Refresh.Queued != 0 {
			if ls.Duration <= ls.Refresh.Queued {
				printf(s.Logger, "%s: The lease policy specified a queued refresh interval of %s for a lease with a duration of %s. The refresh interval will be overridden.\n", prefix, ls.Refresh.Queued.String(), ls.Duration.String())
				ls.Refresh.Queued = 0 // Use the default refresh rate instead of nonsense
			}
		}

		tx := lease.NewTx(subject.Resource, revision, leases)

		acc := leaseutil.Refresh(tx, now)
		consumed := acc.Total(strat)
		released := acc.Released(subject.HostUser())

		existing, found := tx.Instance(subject.Instance)
		if found {
			if existing.Status == lease.Released {
				// Renewal of a released lease, possibly because of timing skew
				// Because the lease has expired we treat this as a creation
				if consumed <= limit {
					ls.Status = lease.Active
				} else {
					ls.Status = lease.Queued
				}
				tx.Update(existing.Instance, ls)
			} else {
				// Renewal of active or queued lease
				mode = "Renewal"
				ls.Status = existing.Status
				ls.Started = existing.Started
				tx.Update(existing.Instance, ls)
			}
		} else {
			if released > 0 && consumed <= limit {
				// Lease replacement (for an expired or released lease previously
				// issued to the the same consumer, that's in a decaying state)
				replaceable := tx.HostUser(subject.Instance.Host, subject.Instance.User).Status(lease.Released)
				if uint(len(replaceable)) != released {
					panic("server: acquireHandler: accumulator returned a different count for relased leases than the transaction")
				}
				replaced := replaceable[released-1]
				ls.Status = lease.Active
				tx.Update(replaced.Instance, ls)
			} else {
				// New lease
				if leaseutil.CanActivate(strat, acc.Active(subject.HostUser()), consumed, limit) {
					ls.Status = lease.Active
				} else {
					ls.Status = lease.Queued
				}
				tx.Create(ls)
			}
		}

		// Don't bother committing empty transactions
		if tx.Empty() {
			break
		}

		// Attempt to commit the transaction
		err = s.LeaseProvider.LeaseCommit(tx)
		if err == nil {
			leases = tx.Leases()
			break
		}

		printf(s.Logger, "%s: Lease acquisition failed: %v\n", prefix, err)
	}

	if err != nil {
		return
	}

	summary := statsSummary(limit, leases.Stats(), strat)
	printf(s.Logger, "%s: %s of %s lease succeeded (%s)\n", prefix, mode, ls.Status, summary)

	return
}

// releaseHandler will attempt to remove the lease for the given resource and
// consumer.
func (s *Server) releaseHandler(w http.ResponseWriter, r *http.Request) {
	req, policies, err := s.initRequest(r)
	if err != nil {
		printf(s.Logger, "Bad release request: %v\n", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	prefix := req.Subject.String()

	printf(s.Logger, "%s: Release requested\n", prefix)

	err = s.release(req.Subject, policies)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	response := transport.ReleaseResponse{
		Request: req,
		Success: true,
	}

	data, err := json.Marshal(response)
	if err != nil {
		printf(s.Logger, "%s: Failed to marshal response: %v\n", prefix, err)
		http.Error(w, "Failed to marshal response", http.StatusBadRequest)
		return
	}

	fmt.Fprintf(w, string(data))
}

func (s *Server) release(subject lease.Subject, policies policy.Set) (err error) {
	prefix := subject.String()

	strat := policies.Strategy()
	limit := policies.Limit()

	var leases lease.Set
	var ls lease.Lease
	var found bool

	for attempt := 0; attempt < 5; attempt++ {
		var revision uint64
		revision, leases, err = s.LeaseProvider.LeaseView(subject.Resource)
		if err != nil {
			printf(s.Logger, "%s: Release failed: %v\n", prefix, err)
			continue
		}

		// Prepare a delete transaction
		now := time.Now()
		tx := lease.NewTx(subject.Resource, revision, leases)
		leaseutil.Refresh(tx, now) // Update stale values
		ls, found = tx.Instance(subject.Instance)
		tx.Release(subject.Instance, now)
		leaseutil.Refresh(tx, now) // Updates leases after release

		// Don't bother committing empty transactions
		if tx.Empty() {
			break
		}

		// Attempt to commit the transaction
		err = s.LeaseProvider.LeaseCommit(tx)
		if err == nil {
			leases = tx.Leases()
			break
		}

		printf(s.Logger, "%s: Release failed: %v\n", prefix, err)
	}

	if err != nil {
		return err
	}

	summary := statsSummary(limit, leases.Stats(), strat)
	if found {
		if ls.Status == lease.Released {
			printf(s.Logger, "%s: Release ignored because the lease had already been released (%s)\n", prefix, summary)
		} else {
			printf(s.Logger, "%s: Release of %s lease succeeded (%s)\n", prefix, ls.Status, summary)
		}
	} else {
		printf(s.Logger, "%s: Release ignored because the lease could not be found (%s)\n", prefix, summary)
	}

	return nil
}

// Purge instructs the server to conduct a full survey of all lease data
// and delete expired leases.
func (s *Server) Purge() error {
	resources, err := s.LeaseProvider.LeaseResources()
	if err != nil {
		return err
	}
	for _, resource := range resources {
		for attempt := 0; attempt < 5; attempt++ {
			var (
				leases   lease.Set
				revision uint64
			)
			revision, leases, err = s.LeaseProvider.LeaseView(resource)
			if err != nil {
				printf(s.Logger, "Purge of \"%s\" failed: %v\n", resource, err)
				continue
			}

			// Prepare a purge transaction
			now := time.Now()
			tx := lease.NewTx(resource, revision, leases)
			leaseutil.Refresh(tx, now)
			if len(tx.Ops()) == 0 {
				break // Nothing to purge
			}

			// Attempt to commit the transaction
			err = s.LeaseProvider.LeaseCommit(tx)
			if err == nil {
				break
			}
			printf(s.Logger, "Purge of \"%s\" failed: %v\n", resource, err)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) initRequest(r *http.Request) (req transport.Request, policies policy.Set, err error) {
	req, err = parseRequest(r)
	if err != nil {
		err = fmt.Errorf("unable to parse request: %v", err)
		return
	}

	if req.HostUser() == "" {
		err = errors.New("consumer not specified or determinable")
		return
	}

	policies, err = s.PolicyProvider.Policies()
	if err != nil {
		err = fmt.Errorf("unable to retrieve policies: %v", err)
		return
	}

	return req, policies.Match(req.Properties), nil
}

func parseRequest(r *http.Request) (req transport.Request, err error) {
	err = r.ParseForm()
	if err != nil {
		return
	}
	req.Properties = make(lease.Properties)
	for k, values := range r.Form {
		if len(values) == 0 {
			continue
		}
		value := values[0] // Ignore multiple values
		switch k {
		case "resource":
			req.Resource = value
		case "host":
			req.Instance.Host = value
		case "user":
			req.Instance.User = value
		case "instance":
			req.Instance.ID = value
		default:
			req.Properties[k] = value
		}
	}
	return
}

func statsSummary(limit uint, stats lease.Stats, strat strategy.Strategy) string {
	consumed := stats.Consumed(strat)
	active := stats.Active(strat)
	released := stats.Released(strat)
	queued := stats.Queued(strat)
	return fmt.Sprintf("alloc: %d/%d, active: %d, released: %d, queued: %d", consumed, limit, active, released, queued)
}

func printf(logger *log.Logger, format string, v ...interface{}) {
	if logger != nil {
		logger.Printf(format, v...)
	}
}
