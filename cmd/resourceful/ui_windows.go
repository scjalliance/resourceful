// +build windows

package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/lxn/walk"
	"github.com/scjalliance/resourceful/enforcerui"
)

func ui(ctx context.Context) {
	ctx, shutdown := context.WithCancel(ctx)

	fmt.Printf("Starting user interface\n")
	icon, err := walk.NewIconFromResourceId(2)
	if err != nil {
		fmt.Printf("Failed to load icon from resource: %v\n", err)
		os.Exit(1)
	}
	defer icon.Dispose()

	ui := enforcerui.New(icon)

	/*
		go func() {
			pols, err := collectPolicies(ctx, "localhost:5877")
			if err != nil {
				shutdown()
				return
			}
			ui.Handle(enforcerui.Message{
				Type: "policy.update",
				PolicyChange: enforcerui.PolicyChange{
					New: pols,
				},
			})
		}()
	*/

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer shutdown()
		defer os.Stdin.Close()
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			var msg enforcerui.Message
			if err := json.Unmarshal([]byte(scanner.Text()), &msg); err != nil {
				fmt.Printf("Failed to unmarshal message\n")
			}
			ui.Handle(msg)
		}
		fmt.Printf("Scanner stopped")
	}()

	if err := ui.Run(ctx); err != nil {
		fmt.Printf("UI: %v\n", err)
		return
	}

	wg.Wait()

	fmt.Printf("Stopped user interface\n")
}
