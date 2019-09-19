// +build !windows

package main

import (
	"context"
	"fmt"
	"os"
)

func enforceService(conf EnforceConfig, confErr error) {
	fmt.Printf("The resourceful policy enforcer can only be run on windows.\n")
	os.Exit(1)
}

func enforceInteractive(ctx context.Context, conf EnforceConfig) {
	fmt.Printf("The resourceful policy enforcer can only be run on windows.\n")
	os.Exit(1)
}
