package main

import (
	"errors"
	"os"

	"github.com/scjalliance/resourceful/runner"
)

func runError(err error) {
	msgBox("resourceful run error", err.Error())
	os.Exit(2)
}

func run(args []string) {
	if len(args) == 0 {
		runError(errors.New("no executable path provided to run"))
	}
	program := args[0]
	args = args[1:]
	icon := programIcon()
	err := runner.RunWithIcon(program, args, icon)
	if err != nil {
		runError(err)
	}
}
