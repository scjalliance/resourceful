//go:build !windows
// +build !windows

package main

import (
	"context"
	"fmt"
	"os"
)

func uninstall(ctx context.Context) {
	fmt.Printf("The resourceful service can only be installed on windows.\n")
	os.Exit(1)
}
