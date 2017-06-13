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

	"github.com/scjalliance/resourceful/environment"
	"github.com/scjalliance/resourceful/guardian/transport"
	"github.com/scjalliance/resourceful/lease"
	"github.com/scjalliance/resourceful/lease/leaseutil"
	"github.com/scjalliance/resourceful/policy"
)

// ServerConfig is the configuration for a resourceful guardian server.
type ServerConfig struct {
	ListenSpec     string
	PolicyProvider policy.Provider
	LeaseProvider  lease.Provider
}

// Server coordinates locks on finite resources.
/*
type Server struct {
}
*/

// Run will create and run a resourceful guardian server until the provided
// context is canceled.
func Run(ctx context.Context, cfg ServerConfig) (err error) {
	log.Printf("Starting, HTTP on: %s\n", cfg.ListenSpec)

	listener, err := net.Listen("tcp", cfg.ListenSpec)
	if err != nil {
		log.Printf("Error creating listener: %v\n", err)
		return
	}

	mux := http.NewServeMux()
	server := &http.Server{
		ReadTimeout:    60 * time.Second,
		WriteTimeout:   60 * time.Second,
		MaxHeaderBytes: 1 << 16,
		Handler:        mux,
	}

	mux.Handle("/health", healthHandler(cfg))
	mux.Handle("/leases", leasesHandler(cfg))
	mux.Handle("/acquire", acquireHandler(cfg))
	mux.Handle("/release", releaseHandler(cfg))

	result := make(chan error)

	go func() {
		result <- server.Serve(listener)
		close(result)
	}()

	select {
	case err = <-result:
		return
	case <-ctx.Done():
	}

	shutdownCtx := context.TODO()
	server.Shutdown(shutdownCtx)

	err = <-result
	return
}

// healthHandler will return the condition of the server.
func healthHandler(cfg ServerConfig) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := transport.HealthResponse{OK: true}
		data, err := json.Marshal(response)
		if err != nil {
			http.Error(w, "Unable to marshal health response", http.StatusBadRequest)
			return
		}
		fmt.Fprintf(w, string(data))
	})
}

// leasesHandler will return the set of leases for a particular resource.
func leasesHandler(cfg ServerConfig) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req, err := parseRequest(r)
		if err != nil {
			err = fmt.Errorf("unable to parse request: %v", err)
			log.Printf("Bad leases request: %v\n", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		revision, leases, err := cfg.LeaseProvider.LeaseView(req.Resource)
		if err != nil {
			log.Printf("Lease retrieval failed: %v\n", err)
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
	})
}

// acquireHandler will attempt to acquire a lease for the specified resource.
func acquireHandler(cfg ServerConfig) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req, pol, err := initRequest(cfg, r)
		if err != nil {
			log.Printf("Bad acquire request: %v\n", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		prefix := fmt.Sprintf("%s %s", req.Resource, req.Consumer)

		log.Printf("%s: Lease acquisition requested\n", prefix)

		limit := pol.Limit()
		duration := pol.Duration()
		decay := pol.Decay()

		var leases lease.Set
		var ls lease.Lease
		mode := "Creation" // Only used for logging

		for attempt := 0; attempt < 5; attempt++ {
			var revision uint64

			revision, leases, err = cfg.LeaseProvider.LeaseView(req.Resource)
			if err != nil {
				log.Printf("%s: Lease retrieval failed: %v\n", prefix, err)
			}
			now := time.Now()

			ls = lease.Lease{
				Resource:    req.Resource,
				Consumer:    req.Consumer,
				Instance:    req.Instance,
				Environment: req.Environment,
				Renewed:     now,
				Limit:       limit,
				Duration:    duration,
				Decay:       decay,
			}

			tx := lease.NewTx(req.Resource, revision, leases)

			leaseutil.Refresh(tx, now)

			existing, found := tx.Instance(req.Consumer, req.Instance)
			if found {
				// Lease renewal
				mode = "Renewal"
				ls.Status = existing.Status
				ls.Started = existing.Started
				tx.Update(existing.Consumer, existing.Instance, ls)
			} else {
				active, released, _ := tx.Stats()
				replaceable := tx.Consumer(req.Consumer).Status(lease.Released)

				ls.Started = now

				if l := len(replaceable); l > 0 && active+released <= limit {
					// Lease replacement (for an expired or released lease previously
					// issued to the the same consumer, that's in a decaying state)
					replaced := replaceable[l-1]
					ls.Status = lease.Active
					tx.Update(replaced.Consumer, replaced.Instance, ls)
				} else {
					// New lease
					if active+released < limit {
						ls.Status = lease.Active
					} else {
						ls.Status = lease.Queued
					}
					tx.Create(ls)
				}
			}

			//suffix := fmt.Sprintf("pol: %d, alloc: %d/%d, d: %v", len(pol), allocation, limit, duration)

			// Attempt to commit the transaction
			err = cfg.LeaseProvider.LeaseCommit(tx)
			if err == nil {
				leases = tx.Leases()
				break
			}

			log.Printf("%s: Lease acquisition failed: %v\n", prefix, err)
		}

		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		active, released, queued := leases.Stats()
		stats := fmt.Sprintf("alloc: %d/%d active: %d, released: %d, queued: %d", active+released, limit, active, released, queued)
		log.Printf("%s: %s of %s lease succeeded (%s)\n", prefix, mode, ls.Status, stats)

		response := transport.AcquireResponse{
			Request: req,
			Lease:   ls,
			Leases:  leases,
		}

		/*
			if !accepted {
				response.Message = fmt.Sprintf("Resource limit of %d has already been met", limit)
			}
		*/

		data, err := json.Marshal(response)
		if err != nil {
			log.Printf("%s: Failed to marshal response: %v\n", prefix, err)
			http.Error(w, "Failed to marshal response", http.StatusBadRequest)
			return
		}

		fmt.Fprintf(w, string(data))
	})
}

// releaseHandler will attempt to remove the lease for the given resource and
// consumer.
func releaseHandler(cfg ServerConfig) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req, pol, err := initRequest(cfg, r)
		if err != nil {
			log.Printf("Bad release request: %v\n", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		prefix := fmt.Sprintf("%s - %s", req.Resource, req.Consumer)

		log.Printf("%s: Release requested\n", prefix)

		limit := pol.Limit()

		var leases lease.Set
		var ls lease.Lease
		var found bool

		for attempt := 0; attempt < 5; attempt++ {
			var revision uint64
			revision, leases, err = cfg.LeaseProvider.LeaseView(req.Resource)
			if err != nil {
				log.Printf("%s: Release failed: %v\n", prefix, err)
				continue
			}

			// Prepare a delete transaction
			now := time.Now()
			tx := lease.NewTx(req.Resource, revision, leases)
			leaseutil.Refresh(tx, now) // Update stale values
			ls, found = tx.Instance(req.Consumer, req.Instance)
			tx.Release(req.Consumer, req.Instance, now)
			leaseutil.Refresh(tx, now) // Updates leases after release

			// Attempt to commit the transaction
			err = cfg.LeaseProvider.LeaseCommit(tx)
			if err == nil {
				leases = tx.Leases()
				break
			}

			log.Printf("%s: Release failed: %v\n", prefix, err)
		}

		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		active, released, queued := leases.Stats()
		stats := fmt.Sprintf("alloc: %d/%d active: %d, released: %d, queued: %d", active+released, limit, active, released, queued)
		if found {
			if ls.Status == lease.Released {
				log.Printf("%s: Release ignored because the lease had already been released (%s)\n", prefix, stats)
			} else {
				log.Printf("%s: Release of %s lease succeeded (%s)\n", prefix, ls.Status, stats)
			}
		} else {
			log.Printf("%s: Release ignored because the lease could not be found (%s)\n", prefix, stats)
		}

		response := transport.ReleaseResponse{
			Request: req,
			Success: true,
		}

		data, err := json.Marshal(response)
		if err != nil {
			log.Printf("%s: Failed to marshal response: %v\n", prefix, err)
			http.Error(w, "Failed to marshal response", http.StatusBadRequest)
			return
		}

		fmt.Fprintf(w, string(data))
	})
}

func initRequest(cfg ServerConfig, r *http.Request) (req transport.Request, policies policy.Set, err error) {
	req, err = parseRequest(r)
	if err != nil {
		err = fmt.Errorf("unable to parse request: %v", err)
		return
	}

	all, err := cfg.PolicyProvider.Policies()
	if err != nil {
		err = fmt.Errorf("unable to retrieve policies: %v", err)
		return
	}

	policies = all.Match(req.Resource, req.Consumer, req.Environment)

	resource := policies.Resource()
	if resource != "" {
		req.Resource = resource
	}
	req.Environment["resource.id"] = req.Resource

	consumer := policies.Consumer()
	if consumer != "" {
		req.Consumer = consumer
	}

	env := policies.Environment()
	if len(env) > 0 {
		req.Environment = environment.Merge(req.Environment, env)
	}

	if req.Resource == "" {
		err = errors.New("resource not specified or determinable")
	} else if req.Consumer == "" {
		err = errors.New("consumer not specified or determinable")
	}
	return
}

func parseRequest(r *http.Request) (req transport.Request, err error) {
	err = r.ParseForm()
	if err != nil {
		return
	}
	req.Environment = make(environment.Environment)
	for k, values := range r.Form {
		if len(values) == 0 {
			continue
		}
		value := values[0] // Ignore multiple values
		switch k {
		case "resource":
			req.Resource = value
		case "consumer":
			req.Consumer = value
		case "instance":
			req.Instance = value
		default:
			req.Environment[k] = value
		}
	}
	return
}
