package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gentlemanautomaton/signaler"

	"github.com/boltdb/bolt"
	"github.com/scjalliance/resourceful/guardian"
	"github.com/scjalliance/resourceful/lease"
	"github.com/scjalliance/resourceful/provider/boltprov"
	"github.com/scjalliance/resourceful/provider/cacheprov"
	"github.com/scjalliance/resourceful/provider/fsprov"
	"github.com/scjalliance/resourceful/provider/logprov"
	"github.com/scjalliance/resourceful/provider/memprov"
	"gopkg.in/alecthomas/kingpin.v2"
)

// GuardianCommand returns a guardian command and configuration for app.
func GuardianCommand(app *kingpin.Application) (*kingpin.CmdClause, *GuardianConfig) {
	cmd := app.Command("guardian", "Runs a guardian policy server.")
	conf := &GuardianConfig{}
	conf.Bind(cmd)
	return cmd, conf
}

// GuardianConfig holds configuration for the guardian command.
type GuardianConfig struct {
	LeaseStorage string
	BoltPath     string
	PolicyPath   string
	TxPath       string
	Schedule     string
}

// Bind binds the guardian configuration to the command.
func (conf *GuardianConfig) Bind(cmd *kingpin.CmdClause) {
	cmd.Flag("leasestore", "lease storage type").Envar("LEASE_STORE").Default(defaultLeaseStorage).EnumVar(&conf.LeaseStorage, "bolt", "memory")
	cmd.Flag("boltpath", "bolt database file path").Envar("BOLT_PATH").Default(defaultBoltPath).StringVar(&conf.BoltPath)
	cmd.Flag("policypath", "policy directory path").Envar("POLICY_PATH").StringVar(&conf.PolicyPath)
	cmd.Flag("txlog", "transaction log file path").Envar("TRANSACTION_LOG").Default(defaultTransactionPath).StringVar(&conf.TxPath)
	cmd.Flag("cpschedule", "transaction checkpoint schedule").Envar("CHECKPOINT_SCHEDULE").StringVar(&conf.Schedule)
}

const (
	defaultLeaseStorage    = "memory"
	defaultBoltPath        = "resourceful.boltdb"
	defaultTransactionPath = "resourceful.tx.log"
)

func daemon(shutdown *signaler.Signaler, conf GuardianConfig) (err error) {
	prepareConsole(false)

	// Prepare a logger that prints to stderr
	logger := log.New(os.Stderr, "", log.LstdFlags)

	// Announce termination
	announcement := shutdown.Then(func() { logger.Println("Received termination signal") })

	// Cancel the context after the announcement
	ctx := announcement.Context()

	if conf.PolicyPath == "" {
		// Use the working directory as the default source for policy files
		conf.PolicyPath, err = os.Getwd()
		if err != nil {
			logger.Printf("Unable to detect working directory: %v", err)
			return
		}
	}

	conf.PolicyPath, err = filepath.Abs(conf.PolicyPath)
	if err != nil {
		logger.Printf("Invalid policy path directory \"%s\": %v", conf.PolicyPath, err)
		return
	}

	var checkpointSchedule []logprov.Schedule
	if conf.Schedule != "" {
		checkpointSchedule, err = logprov.ParseSchedule(conf.Schedule)
		if err != nil {
			logger.Printf("Unable to parse transaction checkpoint schedule: %v", err)
			return
		}
	}

	logger.Println("Starting resourceful guardian daemon")
	defer logger.Printf("Stopped resourceful guardian daemon")

	txFile, err := createTransactionLog(conf.TxPath)
	if err != nil {
		logger.Printf("Unable to open transaction log: %v", err)
		return
	}
	if txFile != nil {
		defer txFile.Close()
	}

	leaseProvider, err := createLeaseProvider(conf.LeaseStorage, conf.BoltPath)
	if err != nil {
		logger.Printf("Unable to create lease provider: %v", err)
		return
	}

	if txFile != nil {
		txLogger := log.New(txFile, "", log.LstdFlags)
		leaseProvider = logprov.New(leaseProvider, txLogger, checkpointSchedule...)
	}

	defer closeProvider(leaseProvider, "lease", logger)

	policyProvider := cacheprov.New(fsprov.New(conf.PolicyPath))

	defer closeProvider(policyProvider, "policy", logger)

	cfg := guardian.ServerConfig{
		ListenSpec:      fmt.Sprintf(":%d", guardian.DefaultPort),
		PolicyProvider:  policyProvider,
		LeaseProvider:   leaseProvider,
		ShutdownTimeout: 5 * time.Second,
		Logger:          logger,
	}

	logger.Printf("Created providers (policy: %s, lease: %s)", policyProvider.ProviderName(), leaseProvider.ProviderName())

	logger.Printf("Policy source directory: %s\n", conf.PolicyPath)
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
