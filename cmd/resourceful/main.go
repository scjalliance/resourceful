package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"syscall"

	"github.com/gentlemanautomaton/signaler"
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
	if len(os.Args) < 2 {
		usage("No command specified.")
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

	command := strings.ToLower(os.Args[1])
	args := os.Args[2:]

	switch command {
	case "run":
		run(ctx, args)
	case "list":
		list(ctx, args)
	case "daemon", "guardian":
		err := daemon(ctx, logger, strings.Join(os.Args[0:2], " "), args)
		if err != nil {
			os.Exit(2)
		}
	default:
		usage(fmt.Sprintf("\"%s\" is an unrecognized command.", command))
	}
}
