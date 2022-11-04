//go:build windows
// +build windows

package main

import (
	"context"
	"io"
	"os"

	"github.com/lxn/walk"
	"github.com/scjalliance/resourceful/enforcerui"
)

// Run executes the ui command.
func (cmd *UICmd) Run(ctx context.Context) error {
	// Prepare the icon used by the user interface
	icon, err := walk.NewIconFromResourceId(IconResourceID)
	if err != nil {
		os.Exit(1)
	}
	defer icon.Dispose()

	// Create the user interface and close it when we're done
	ui, err := enforcerui.New(icon, ProgramName, Version)
	if err != nil {
		os.Exit(2)
	}
	defer ui.Close()

	// Close stdin when we receive a shutdown signal so that we interrupt the
	// reader loop
	ctx, shutdown := context.WithCancel(ctx)
	defer shutdown()
	go func() {
		<-ctx.Done()
		os.Stdin.Close()
	}()

	r := enforcerui.NewReader(os.Stdin)
	//w := enforcerui.NewWriter(os.Stdout)

	for {
		msg, err := r.Read()
		if err != nil {
			//fmt.Printf("enforcer ui read: %v\n", err)
			if err != io.EOF {
				os.Exit(3)
			}
			return nil
		}
		ui.Handle(msg)
	}
}
