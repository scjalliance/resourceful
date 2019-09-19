// +build windows

package main

import (
	"context"
	"fmt"
	"os"

	"github.com/gentlemanautomaton/winservice"
	"github.com/scjalliance/resourceful/enforcer"
)

func uninstall(ctx context.Context) {
	// Check for an existing enforcement service
	exists, err := winservice.Exists(enforcer.ServiceName)
	if err != nil {
		fmt.Printf("Failed to check for existing enforcement service: %v\n", err)
		os.Exit(1)
	}
	if !exists {
		fmt.Printf("An installation of the \"%s\" service could not be found.\n", enforcer.ServiceName)
		return
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
}
