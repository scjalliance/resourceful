//go:build !windows
// +build !windows

package main

import (
	"context"
	"fmt"
)

func ui(ctx context.Context) (exit int) {
	fmt.Printf("The resourceful ui can only be run on windows.\n")
	return 1
}
