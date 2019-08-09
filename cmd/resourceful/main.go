package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/gentlemanautomaton/signaler"
	"gopkg.in/alecthomas/kingpin.v2"
)

func usage(errmsg string) {
	fmt.Fprintf(os.Stderr,
		"%s\n\n"+
			"usage: %s <command>\n"+
			"       where <command> is one of\n"+
			"       run, list, guardian.\n",
		errmsg, os.Args[0])
	os.Exit(2)
}

func main() {
	app := kingpin.New(filepath.Base(os.Args[0]), "Provides lease-based management of running programs.")
	app.Interspersed(false)

	var (
		listCmd    = app.Command("list", "Lists running processes that match current policies.")
		listServer = listCmd.Flag("server", "Guardian policy server host and port.").Short('s').String()
	)

	var (
		enforceCmd     = app.Command("enforce", "Enforces resourceful policies on the local machine.")
		enforceServer  = enforceCmd.Flag("server", "Guardian policy server host and port.").Short('s').String()
		enforcePassive = enforceCmd.Flag("passive", "Run passively without killing processes.").Bool()
	)

	var (
		runCmd     = app.Command("run", "Runs a program if a lease can be procured for it.")
		runServer  = runCmd.Flag("server", "Guardian policy server host and port.").Short('s').String()
		runProgram = runCmd.Arg("program", "program to run").Required().String()
		runArgs    = runCmd.Arg("arguments", "program arguments").Strings()
	)

	var (
		guardianCmd  = app.Command("guardian", "Runs a guardian policy server.")
		leaseStorage = guardianCmd.Flag("leasestore", "lease storage type").Envar("LEASE_STORE").Default(defaultLeaseStorage).Enum("bolt", "memory")
		boltPath     = guardianCmd.Flag("boltpath", "bolt database file path").Envar("BOLT_PATH").Default(defaultBoltPath).String()
		policyPath   = guardianCmd.Flag("policypath", "policy directory path").Envar("POLICY_PATH").String()
		txPath       = guardianCmd.Flag("txlog", "transaction log file path").Envar("TRANSACTION_LOG").Default(defaultTransactionPath).String()
		schedule     = guardianCmd.Flag("cpschedule", "transaction checkpoint schedule").Envar("CHECKPOINT_SCHEDULE").String()
	)

	command, err := app.Parse(os.Args[1:])
	if err != nil {
		// Special GUI-based handling for run
		if len(os.Args) > 1 && strings.EqualFold(os.Args[1], "run") {
			runError(err)
		}
		prepareConsole(false)
		app.Fatalf("%s, try --help", err)
	}

	interactive, err := isInteractive()
	if err != nil {
		app.Fatalf("%s", err)
	}

	// Prepare a logger that prints to stderr
	logger := log.New(os.Stderr, "", log.LstdFlags)

	// Shutdown when we receive a termination signal
	shutdown := signaler.New().Capture(os.Interrupt, syscall.SIGTERM)

	// Ensure that we cleanup even if we panic
	defer shutdown.Trigger()

	// Announce termination
	announcement := shutdown.Then(func() { logger.Println("Received termination signal") })

	// Cancel a context after the announcement
	ctx := announcement.Context()

	switch command {
	case runCmd.FullCommand():
		run(ctx, *runServer, *runProgram, *runArgs)
	case listCmd.FullCommand():
		list(ctx, *listServer)
	case enforceCmd.FullCommand():
		enforce(ctx, *enforceServer, interactive, *enforcePassive)
	case guardianCmd.FullCommand():
		err := daemon(ctx, logger, *leaseStorage, *boltPath, *policyPath, *txPath, *schedule)
		if err != nil {
			os.Exit(2)
		}
	}
}
