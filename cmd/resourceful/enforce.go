// +build !windows

package main

import (
	"context"
	"fmt"
	"os"
)

func enforce(ctx context.Context, server string, interactive bool, passive bool) {
	fmt.Printf("The resourceful policy enforcer can only be run on windows.\n")
	os.Exit(1)
}
