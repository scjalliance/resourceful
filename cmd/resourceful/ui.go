// +build !windows

package main

import (
	"context"
	"fmt"
	"os"
)

func ui(ctx context.Context) {
	fmt.Printf("The resourceful ui can only be run on windows.\n")
	os.Exit(1)
}
