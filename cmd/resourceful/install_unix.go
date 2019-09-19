// +build !windows

package main

import (
	"context"
	"fmt"
	"os"
)

func install(ctx context.Context, program string, conf EnforceConfig) {
	fmt.Printf("The resourceful program can only be installed on windows.\n")
	os.Exit(1)
}
