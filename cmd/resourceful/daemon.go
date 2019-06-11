package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/boltdb/bolt"
	"github.com/scjalliance/resourceful/guardian"
	"github.com/scjalliance/resourceful/lease"
	"github.com/scjalliance/resourceful/provider/boltprov"
	"github.com/scjalliance/resourceful/provider/cacheprov"
	"github.com/scjalliance/resourceful/provider/fsprov"
	"github.com/scjalliance/resourceful/provider/logprov"
	"github.com/scjalliance/resourceful/provider/memprov"
)

const (
	defaultLeaseStorage    = "memory"
	defaultBoltPath        = "resourceful.boltdb"
	defaultTransactionPath = "resourceful.tx.log"
)

func daemon(ctx context.Context, logger *log.Logger, leaseStorage, boltPath, policyPath, txPath, schedule string) (err error) {
	prepareConsole(false)

	if policyPath == "" {
		// Use the working directory as the default source for policy files
		policyPath, err = os.Getwd()
		if err != nil {
			logger.Printf("Unable to detect working directory: %v", err)
			return
		}
	}

	policyPath, err = filepath.Abs(policyPath)
	if err != nil {
		logger.Printf("Invalid policy path directory \"%s\": %v", policyPath, err)
		return
	}

	var checkpointSchedule []logprov.Schedule
	if schedule != "" {
		checkpointSchedule, err = logprov.ParseSchedule(schedule)
		if err != nil {
			logger.Printf("Unable to parse transaction checkpoint schedule: %v", err)
			return
		}
	}

	logger.Println("Starting resourceful guardian daemon")
	defer logger.Printf("Stopped resourceful guardian daemon")

	txFile, err := createTransactionLog(txPath)
	if err != nil {
		logger.Printf("Unable to open transaction log: %v", err)
		return
	}
	if txFile != nil {
		defer txFile.Close()
	}

	leaseProvider, err := createLeaseProvider(leaseStorage, boltPath)
	if err != nil {
		logger.Printf("Unable to create lease provider: %v", err)
		return
	}

	if txFile != nil {
		txLogger := log.New(txFile, "", log.LstdFlags)
		leaseProvider = logprov.New(leaseProvider, txLogger, checkpointSchedule...)
	}

	defer closeProvider(leaseProvider, "lease", logger)

	policyProvider := cacheprov.New(fsprov.New(policyPath))

	defer closeProvider(policyProvider, "policy", logger)

	cfg := guardian.ServerConfig{
		ListenSpec:      fmt.Sprintf(":%d", guardian.DefaultPort),
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
		return
	}

	count := len(policies)
	switch count {
	case 1:
		logger.Printf("1 policy loaded")
	default:
		logger.Printf("%d policies loaded", count)
	}

	err = guardian.Run(ctx, cfg)

	if err != http.ErrServerClosed {
		logger.Printf("Stopped resourceful guardian daemon due to error: %v", err)
	}
	return
}

func createTransactionLog(path string) (file *os.File, err error) {
	if path == "" {
		return nil, nil
	}
	return os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
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

type closer interface {
	Close() error
}

func closeProvider(prov closer, name string, logger *log.Logger) {
	if err := prov.Close(); err != nil {
		if logger != nil {
			logger.Printf("The %s provider did not shut down correctly: %v", name, err)
		}
	}
}
