package main

import (
	"fmt"
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
	interactive, err := isInteractive()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to determine interactive session status: %v\n", err)
		os.Exit(1)
	}

	var (
		app                       = App()
		listCmd, listConf         = ListCommand(app)
		installCmd, installConf   = InstallCommand(app)
		uninstallCmd              = UninstallCommand(app)
		enforceCmd, enforceConf   = EnforceCommand(app)
		guardianCmd, guardianConf = GuardianCommand(app)
		uiCmd                     = UICommand(app)
		runCmd, runConf           = RunCommand(app)
	)

	command, err := app.Parse(os.Args[1:])

	// Non-interactive means we're running as LocalSystem. This typically means
	// we're being invoked as a service, but it could also mean we're being
	// run via "psexec -s -i". We check our own "-i" flag to override
	// invocation via the service framework.
	if !interactive && !enforceConf.Interactive && command == enforceCmd.FullCommand() {
		enforceService(*enforceConf, err)
		return
	}

	if err != nil {
		// Special GUI-based error handling for run
		if len(os.Args) > 1 && strings.EqualFold(os.Args[1], "run") {
			runError(err)
		}
		prepareConsole(false)
		app.Fatalf("%s, try --help", err)
	}

	// Shutdown when we receive a termination signal
	shutdown := signaler.New().Capture(os.Interrupt, syscall.SIGTERM)

	// Ensure that we cleanup even if we panic
	defer shutdown.Trigger()

	switch command {
	case uiCmd.FullCommand():
		ui(shutdown.Context())
	case runCmd.FullCommand():
		run(shutdown.Context(), *runConf)
	case listCmd.FullCommand():
		list(shutdown.Context(), *listConf)
	case installCmd.FullCommand():
		install(shutdown.Context(), os.Args[0], *installConf)
	case uninstallCmd.FullCommand():
		uninstall(shutdown.Context())
	case enforceCmd.FullCommand():
		enforceInteractive(shutdown.Context(), *enforceConf)
	case guardianCmd.FullCommand():
		// Run the server
		err := daemon(shutdown, *guardianConf)
		if err != nil {
			os.Exit(2)
		}
	}
}
