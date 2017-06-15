package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
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

func daemon(command string, args []string) {
	prepareConsole(false)

	logger := log.New(os.Stderr, "", log.LstdFlags)

	var (
		selectedLeaseProvider string
		boltPath              string
	)

	fs := flag.NewFlagSet(command, flag.ExitOnError)
	fs.StringVar(&selectedLeaseProvider, "lease", "memory", "lease provider type [\"boltdb\", \"memory\"]")
	fs.StringVar(&boltPath, "boltdb", "resourceful.boltdb", "bolt database file path")
	fs.Parse(args)

	// Detect the working directory, which is the source of policy files
	wd, err := os.Getwd()
	if err != nil {
		logger.Printf("Unable to detect working directory: %v", err)
		os.Exit(2)
	}

	logger.Println("Starting resourceful guardian daemon")

	leaseProvider, err := createLeaseProvider(selectedLeaseProvider, boltPath)
	if err != nil {
		logger.Printf("Unable to create lease provider: %v", err)
		os.Exit(2)
	}

	policyProvider := cacheprov.New(fsprov.New(wd))

	cfg := guardian.ServerConfig{
		ListenSpec:      ":5877",
		PolicyProvider:  policyProvider,
		LeaseProvider:   leaseProvider,
		ShutdownTimeout: 5 * time.Second,
		Logger:          logger,
	}

	logger.Printf("Created providers (policy: %s, lease: %s)", policyProvider.ProviderName(), leaseProvider.ProviderName())

	logger.Printf("Policy source directory: %s\n", wd)
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

func createLeaseProvider(prov string, boltPath string) (lease.Provider, error) {
	switch strings.ToLower(prov) {
	case "m", "mem", "memory":
		return memprov.New(), nil
	case "b", "bolt", "boltdb":
		boltdb, err := bolt.Open(boltPath, 0666, nil)
		if err != nil {
			return nil, fmt.Errorf("unable to open or create bolt database \"%s\": %v", boltPath, err)
		}
		return boltprov.New(boltdb), nil
	default:
		return nil, fmt.Errorf("unknown lease provider: %s", prov)
	}
}

func waitForSignal(logger *log.Logger) {
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	s := <-ch
	logger.Printf("Got signal: %v, exiting.", s)
}
