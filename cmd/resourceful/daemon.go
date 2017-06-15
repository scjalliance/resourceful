package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/boltdb/bolt"
	"github.com/scjalliance/resourceful/guardian"
	"github.com/scjalliance/resourceful/lease"
	"github.com/scjalliance/resourceful/provider/boltprov"
	"github.com/scjalliance/resourceful/provider/cacheprov"
	"github.com/scjalliance/resourceful/provider/fsprov"
	"github.com/scjalliance/resourceful/provider/memprov"
)

const (
	defaultLeaseStorage = "memory"
	defaultBoltPath     = "resourceful.boltdb"
)

func daemon(command string, args []string) {
	prepareConsole(false)

	logger := log.New(os.Stderr, "", log.LstdFlags)

	var (
		leaseStorage = os.Getenv("LEASE_STORE")
		boltPath     = os.Getenv("BOLT_PATH")
		policyPath   = os.Getenv("POLICY_PATH")
		err          error
	)

	if leaseStorage == "" {
		leaseStorage = defaultLeaseStorage
	}
	if boltPath == "" {
		boltPath = defaultBoltPath
	}
	if policyPath == "" {
		// Use the working directory as the default source for policy files
		policyPath, err = os.Getwd()
		if err != nil {
			logger.Printf("Unable to detect working directory: %v", err)
			os.Exit(2)
		}
	}

	fs := flag.NewFlagSet(command, flag.ExitOnError)
	fs.StringVar(&leaseStorage, "leasestore", leaseStorage, "lease storage type [\"bolt\", \"memory\"]")
	fs.StringVar(&boltPath, "boltpath", boltPath, "bolt database file path")
	fs.StringVar(&policyPath, "policypath", policyPath, "policy directory path")
	fs.Parse(args)

	policyPath, err = filepath.Abs(policyPath)
	if err != nil {
		logger.Printf("Invalid policy path directory \"%s\": %v", policyPath, err)
		os.Exit(2)
	}

	logger.Println("Starting resourceful guardian daemon")

	leaseProvider, err := createLeaseProvider(leaseStorage, boltPath)
	if err != nil {
		logger.Printf("Unable to create lease provider: %v", err)
		os.Exit(2)
	}

	policyProvider := cacheprov.New(fsprov.New(policyPath))

	cfg := guardian.ServerConfig{
		ListenSpec:      ":5877",
		PolicyProvider:  policyProvider,
		LeaseProvider:   leaseProvider,
		ShutdownTimeout: 5 * time.Second,
		Logger:          logger,
	}

	logger.Printf("Created providers (policy: %s, lease: %s)", policyProvider.ProviderName(), leaseProvider.ProviderName())

	logger.Printf("Policy source directory: %s\n", policyPath)
	// Verify that we're starting with a good policy set
	policies, err := cfg.PolicyProvider.Policies()
	if err != nil {
		logger.Printf("Failed to load policy set: %v", err)
		os.Exit(2)
	}

	count := len(policies)
	switch count {
	case 1:
		logger.Printf("1 policy loaded")
	default:
		logger.Printf("%d policies loaded", count)
	}

	ctx, shutdown := context.WithCancel(context.Background())
	defer shutdown()
	go func() {
		waitForSignal(logger)
		shutdown()
	}()

	err = guardian.Run(ctx, cfg)

	if provErr := leaseProvider.Close(); provErr != nil {
		logger.Printf("The lease provider did not shut down correctly: %v", provErr)
	}
	if provErr := policyProvider.Close(); provErr != nil {
		logger.Printf("The policy provider did not shut down correctly: %v", provErr)
	}

	if err != http.ErrServerClosed {
		logger.Printf("Stopped resourceful guardian daemon due to error: %v", err)
		os.Exit(2)
	}

	logger.Printf("Stopped resourceful guardian daemon")
}

func createLeaseProvider(storage string, boltPath string) (lease.Provider, error) {
	switch strings.ToLower(storage) {
	case "mem", "memory":
		return memprov.New(), nil
	case "bolt", "boltdb":
		boltdb, err := bolt.Open(boltPath, 0666, nil)
		if err != nil {
			return nil, fmt.Errorf("unable to open or create bolt database \"%s\": %v", boltPath, err)
		}
		return boltprov.New(boltdb), nil
	default:
		return nil, fmt.Errorf("unknown lease storage type: %s", storage)
	}
}

func waitForSignal(logger *log.Logger) {
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	s := <-ch
	logger.Printf("Got signal: %v, exiting.", s)
}
