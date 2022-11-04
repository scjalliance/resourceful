//go:build !windows
// +build !windows

package main

import (
	"context"
	"errors"
)

// Run executes the uninstall command.
func (cmd *UninstallCmd) Run(ctx context.Context) error {
	return errors.New("the resourceful enforcement service can only be installed on windows")
}
