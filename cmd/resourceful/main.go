package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/alecthomas/kong"
)

// To test the enforcement service without installing it, run
// "resourceful enforce" via "psexec -s -i"

func main() {
	if isService, err := isWindowsService(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to detect service invocation: %v\n", err)
		os.Exit(1)
	} else if isService {
		runServiceHandler()
		os.Exit(0)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var cli struct {
		List      ListCmd      `kong:"cmd,help='Lists running processes that match current policies.'"`
		Install   InstallCmd   `kong:"cmd,help='Installs the resourceful enforcer service on the local machine.'"`
		Uninstall UninstallCmd `kong:"cmd,help='Uninstalls the resourceful enforcer service from the local machine.'"`
		Enforce   EnforceCmd   `kong:"cmd,help='Enforces resourceful policies on the local machine.'"`
		Guardian  GuardianCmd  `kong:"cmd,help='Runs a guardian policy server.'"`
		UI        UICmd        `kong:"cmd,help='Starts a user interface agent.'"`
	}

	parser := kong.Must(&cli,
		kong.Description("Provides lease-based management of running programs."),
		kong.BindTo(ctx, (*context.Context)(nil)),
		kong.UsageOnError())

	app, parseErr := parser.Parse(os.Args[1:])
	parser.FatalIfErrorf(parseErr)

	appErr := app.Run()
	app.FatalIfErrorf(appErr)
}
