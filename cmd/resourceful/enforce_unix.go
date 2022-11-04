//go:build !windows
// +build !windows

package main

import (
	"context"
	"errors"
	"fmt"
	"os"
)

// Run executes the enforce command.
func (cmd *EnforceCmd) Run(ctx context.Context) error {
	return errors.New("the resourceful policy enforcer can only be run on windows")
}

func runServiceHandler() {
	fmt.Printf("The resourceful policy enforcer can only be run on windows.\n")
	os.Exit(1)
}
