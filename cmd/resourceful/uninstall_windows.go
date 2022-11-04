//go:build windows
// +build windows

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"github.com/gentlemanautomaton/winservice"
	"github.com/scjalliance/resourceful/enforcer"
	"golang.org/x/sys/windows/svc/eventlog"
)

// Run executes the uninstall command.
func (cmd *UninstallCmd) Run(ctx context.Context) error {
	// Check for an existing enforcement service
	exists, err := winservice.Exists(enforcer.ServiceName)
	if err != nil {
		fmt.Printf("Failed to check for existing enforcement service: %v\n", err)
		os.Exit(1)
	}
	if !exists {
		fmt.Printf("An installation of the \"%s\" service could not be found.\n", enforcer.ServiceName)
		return nil
	}
	fmt.Printf("An installation of the \"%s\" service was found.\n", enforcer.ServiceName)

	// Stop and remove any existing service
	if err := winservice.Delete(context.Background(), enforcer.ServiceName); err != nil {
		if opErr, ok := err.(winservice.OpError); ok && opErr.Err == winservice.ErrServiceMarkedForDeletion {
			fmt.Printf("The service has already been marked for deletion.\n")
		} else {
			fmt.Printf("Removal of existing service failed: %v\n", err)
			os.Exit(1)
		}
	} else {
		fmt.Printf("The \"%s\" service has been uninstalled.\n", enforcer.ServiceName)
	}

	// Remove the event log handler for the service
	if err := eventlog.Remove(enforcer.ServiceName); err != nil {
		// Report the error but press on regardless
		fmt.Printf("Failed to remove event log source for %s: %v\n", enforcer.ServiceName, err)
	} else {
		fmt.Printf("The \"%s\" service event log handler has been uninstalled.\n", enforcer.ServiceName)
	}

	// Remove registry keys
	if err := delUninstallRegKeys(); err != nil {
		fmt.Printf("Failed to remove uninstall entry from the Windows registry: %v\n", err)
	}
	fmt.Printf("Removed uninstall entry from from the Windows registry.\n")

	return nil
}

// uninstallCommand returns a command line string can be run to uninstall
// resourceful. The returned string will be properly quoted.
func uninstallCommand(dir, executable string) string {
	return syscall.EscapeArg(filepath.Join(dir, executable)) + " uninstall"
}
