package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/scjalliance/resourceful/guardian"
	"github.com/scjalliance/resourceful/provider/cacheprov"
	"github.com/scjalliance/resourceful/provider/fsprov"
	"github.com/scjalliance/resourceful/provider/memprov"
)

func daemon(args []string) {
	prepareConsole(false)

	log.Println("Starting resourceful guardian daemon")

	// Detect the working directory, which is the source of policy files
	wd, err := os.Getwd()
	if err != nil {
		log.Printf("Unable to detect working directory: %v", err)
		os.Exit(1)
	}
	log.Printf("Policy source directory: %s\n", wd)

	cfg := guardian.ServerConfig{
		ListenSpec:     ":5877",
		PolicyProvider: cacheprov.New(fsprov.New(wd)),
		LeaseProvider:  memprov.New(),
	}

	// Verify that we're starting with a good policy set
	policies, err := cfg.PolicyProvider.Policies()
	if err != nil {
		log.Printf("Failed to load policy set: %v", err)
		os.Exit(1)
	}

	count := len(policies)
	switch count {
	case 1:
		log.Printf("1 policy loaded")
	default:
		log.Printf("%d policies loaded", count)
	}

	ctx, shutdown := context.WithCancel(context.Background())
	defer shutdown()
	go func() {
		waitForSignal()
		shutdown()
	}()

	err = guardian.Run(ctx, cfg)

	if err != nil {
		log.Printf("Guardian server error: %v", err)
		os.Exit(1)
	}

	log.Printf("Stopped resourceful guardian daemon")
}

func waitForSignal() {
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	s := <-ch
	log.Printf("Got signal: %v, exiting.", s)
}
