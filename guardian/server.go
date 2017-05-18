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

	//mux.Handle("/", indexHandler(m))
	mux.Handle("/health", healthHandler(cfg))
	mux.Handle("/acquire", acquireHandler(cfg))
	mux.Handle("/release", releaseHandler(cfg))
	//mux.Handle("/js/", http.FileServer(cfg.Assets))

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

// acquireHandler will attempt to acquire a lease for the specified resource.
func acquireHandler(cfg ServerConfig) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req, pol, err := initRequest(cfg, r)
		if err != nil {
			log.Printf("Failed request: %v\n", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		prefix := fmt.Sprintf("%s %s", req.Resource, req.Consumer)

		log.Printf("%s: Lease acquisition requested\n", prefix)

		limit := pol.Limit()
		duration := pol.Duration()

		l, allocation, accepted, err := cfg.LeaseProvider.Acquire(req.Resource, req.Consumer, req.Environment, limit, duration)
		suffix := fmt.Sprintf("pol: %d, alloc: %d/%d, d: %v", len(pol), allocation, limit, duration)
		if err != nil {
			log.Printf("%s: Lease acquisition failed: %v (%s)\n", prefix, err, suffix)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if accepted {
			log.Printf("%s: Lease acquisition accepted (%s)\n", prefix, suffix)
		} else {
			log.Printf("%s: Lease acquisition rejected (%s)\n", prefix, suffix)
		}

		leases, err := cfg.LeaseProvider.Leases(req.Resource)
		if err != nil {
			// Log the error but return a lease set with our lease, which is the only
			// one that we know of.
			log.Printf("%s: Lease enumeration failed: %v (%s)\n", prefix, err, suffix)
			leases = lease.Set{l}
		}

		response := transport.AcquireResponse{
			Request:  req,
			Accepted: accepted,
			Leases:   leases,
		}

		if !accepted {
			response.Message = fmt.Sprintf("Resource limit of %d has already been met", limit)
		}

		data, err := json.Marshal(response)
		if err != nil {
			log.Printf("%s: Failed to marshal response: %v (%s)\n", prefix, err, suffix)
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
		req, _, err := initRequest(cfg, r)
		if err != nil {
			log.Printf("Bad request: %v\n", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		prefix := fmt.Sprintf("%s - %s", req.Resource, req.Consumer)

		log.Printf("%s: Lease removal requested\n", prefix)

		err = cfg.LeaseProvider.Release(req.Resource, req.Consumer)
		if err != nil {
			log.Printf("%s: Lease removal failed: %v\n", prefix, err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		log.Printf("%s: Lease removal succeeded\n", prefix)

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

	consumer := policies.Consumer()
	if consumer != "" {
		req.Consumer = consumer
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
		default:
			if req.Environment == nil {
				req.Environment = make(environment.Environment)
			}
			req.Environment[k] = value
		}
	}
	return
}
