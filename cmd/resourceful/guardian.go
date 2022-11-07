package main

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
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

// GuardianCmd runs a guardian policy server.
type GuardianCmd struct {
	LeaseStorage  string        `kong:"optional,name='leasestore',env='LEASE_STORE',default='memory',help='Lease storage type.'"`
	BoltPath      string        `kong:"optional,name='boltpath',env='BOLT_PATH',default='resourceful.boltdb',help='Bolt database file path.'"`
	PolicyPath    string        `kong:"optional,name='policypath',env='POLICY_PATH',help='Policy directory path.'"`
	TxPath        string        `kong:"optional,name='txlog',env='TRANSACTION_LOG',default='resourceful.tx.log',help='Transaction log file path.'"`
	Schedule      string        `kong:"optional,name='cpschedule',env='CHECKPOINT_SCHEDULE',help='Transaction checkpoint schedule.'"`
	StatHatKey    string        `kong:"optional,name='stathatkey',env='STATHAT_KEY',help='Optional StatHat key for recording statistics.'"`
	StatsInterval time.Duration `kong:"optional,name='stats',env='STATS_INTERVAL',default='1m',help='Optional interval for recording statistics.'"`
}

// Run executes the guardian command.
func (cmd *GuardianCmd) Run(ctx context.Context) (err error) {
	//func daemon(shutdown *signaler.Signaler, conf GuardianConfig) (err error) {
	prepareConsole(false)

	// Prepare a logger that prints to stderr
	logger := log.New(os.Stderr, "", log.LstdFlags)

	// Announce termination
	//announcement := shutdown.Then(func() { logger.Println("Received termination signal") })

	// Cancel the context after the announcement
	//ctx := announcement.Context()

	const minStatsInterval = 5 * time.Second
	if cmd.StatsInterval < minStatsInterval {
		// Don't spam the stats recipient
		cmd.StatsInterval = minStatsInterval
	}

	if cmd.PolicyPath == "" {
		// Use the working directory as the default source for policy files
		cmd.PolicyPath, err = os.Getwd()
		if err != nil {
			logger.Printf("Unable to detect working directory: %v", err)
			return nil
		}
	}

	cmd.PolicyPath, err = filepath.Abs(cmd.PolicyPath)
	if err != nil {
		logger.Printf("Invalid policy path directory \"%s\": %v", cmd.PolicyPath, err)
		return nil
	}

	var checkpointSchedule []logprov.Schedule
	if cmd.Schedule != "" {
		checkpointSchedule, err = logprov.ParseSchedule(cmd.Schedule)
		if err != nil {
			logger.Printf("Unable to parse transaction checkpoint schedule: %v", err)
			return
		}
	}

	logger.Println("Starting resourceful guardian daemon")
	defer logger.Printf("Stopped resourceful guardian daemon")

	txFile, err := createTransactionLog(cmd.TxPath)
	if err != nil {
		logger.Printf("Unable to open transaction log: %v", err)
		return
	}
	if txFile != nil {
		defer txFile.Close()
	}

	leaseProvider, err := createLeaseProvider(cmd.LeaseStorage, cmd.BoltPath)
	if err != nil {
		logger.Printf("Unable to create lease provider: %v", err)
		return
	}

	if txFile != nil {
		txLogger := log.New(txFile, "", log.LstdFlags)
		leaseProvider = logprov.New(leaseProvider, txLogger, checkpointSchedule...)
	}

	defer closeProvider(leaseProvider, "lease", logger)

	policyProvider := cacheprov.New(fsprov.New(cmd.PolicyPath))

	defer closeProvider(policyProvider, "policy", logger)

	// The embeded file system contains all files in a www directory, which
	// is an unnecessary detail we would like to hide form the world.
	fsys, err := fs.Sub(webfiles, "www")
	if err != nil {
		return err
	}

	cfg := guardian.ServerConfig{
		ListenSpec:      fmt.Sprintf(":%d", guardian.DefaultPort),
		PolicyProvider:  policyProvider,
		LeaseProvider:   leaseProvider,
		RefreshInterval: 2 * time.Second,
		ShutdownTimeout: 5 * time.Second,
		Logger:          logger,
		Handler:         http.FileServer(http.FS(fsys)),
	}

	logger.Printf("Created providers (policy: %s, lease: %s)", policyProvider.ProviderName(), leaseProvider.ProviderName())

	logger.Printf("Policy source directory: %s\n", cmd.PolicyPath)
	// Verify that we're starting with a good policy set
	policies, err := cfg.PolicyProvider.Policies()
	if err != nil {
		logger.Printf("Failed to load policy set: %v", err)
		return nil
	}

	count := len(policies)
	switch count {
	case 1:
		logger.Printf("1 policy loaded")
	default:
		logger.Printf("%d policies loaded", count)
	}

	if recipient := createStatRecipient(cmd.StatHatKey); recipient != nil {
		stats := NewStatManager(recipient)
		if err := stats.Init(policyProvider, leaseProvider); err != nil {
			logger.Printf("Failed to collect lease statistics: %v", err)
			return err
		}

		statsCtx, statsCancel := context.WithCancel(context.Background())
		var wg sync.WaitGroup
		wg.Add(1)

		go func() {
			defer wg.Done()

			t := time.NewTicker(cmd.StatsInterval)
			defer t.Stop()

			for {
				select {
				case <-statsCtx.Done():
					// Attempt to send a final set of statistics after the
					// the server has stopped
					stats.CollectAndSend(policyProvider, leaseProvider)
					return
				case <-t.C:
					if err := stats.CollectAndSend(policyProvider, leaseProvider); err != nil {
						logger.Printf("Failed to collect and send lease statistics: %v", err)
					}
				}
			}
		}()

		err = guardian.Run(ctx, cfg)

		statsCancel()
		wg.Wait()
	} else {
		err = guardian.Run(ctx, cfg)
	}

	if err != http.ErrServerClosed {
		logger.Printf("Stopped resourceful guardian daemon due to error: %v", err)
	}
	return
}

func createStatRecipient(statHatKey string) StatRecipient {
	if statHatKey != "" {
		return NewStatHatRecipient("resourceful", statHatKey)
	}
	return nil
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
