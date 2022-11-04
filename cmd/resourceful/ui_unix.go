//go:build !windows
// +build !windows

package main

import (
	"context"
	"errors"
)

// Run executes the ui command.
func (cmd *UICmd) Run(ctx context.Context) error {
	return errors.New("the resourceful ui can only be run on windows")
}
