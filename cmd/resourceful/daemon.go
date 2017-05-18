package main

import (
	"context"
	"errors"
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

	wd, err := os.Getwd()
	if err != nil {
		log.Print(errors.New("unable to detect working directory"))
	}
	log.Printf("Policy source directory: %s\n", wd)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		waitForSignal()
		cancel()
	}()
	cfg := guardian.ServerConfig{
		ListenSpec:     ":5877",
		PolicyProvider: cacheprov.New(fsprov.New(wd)),
		LeaseProvider:  memprov.New(),
	}
	err = guardian.Run(ctx, cfg)
	log.Print(err)
}

func waitForSignal() {
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	s := <-ch
	log.Printf("Got signal: %v, exiting.", s)
}
