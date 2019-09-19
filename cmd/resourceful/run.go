package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/scjalliance/resourceful/guardian"
	"github.com/scjalliance/resourceful/lease/leaseui"
	"github.com/scjalliance/resourceful/runner"
	"gopkg.in/alecthomas/kingpin.v2"
)

// RunCommand returns a run command and configuration for app.
func RunCommand(app *kingpin.Application) (*kingpin.CmdClause, *RunConfig) {
	cmd := app.Command("run", "Runs a program if a lease can be procured for it.")
	conf := &RunConfig{}
	conf.Bind(cmd)
	return cmd, conf
}

// RunConfig holds configuration for the run command.
type RunConfig struct {
	Server  string
	Program string
	Args    []string
}

// Bind binds the guardian configuration to the command.
func (conf *RunConfig) Bind(cmd *kingpin.CmdClause) {
	cmd.Flag("server", "Guardian policy server host and port.").Short('s').StringVar(&conf.Server)
	cmd.Arg("program", "program to run").Required().StringVar(&conf.Program)
	cmd.Arg("arguments", "program arguments").StringsVar(&conf.Args)
}

func runError(err error) {
	leaseui.Notify("resourceful run error", err.Error())
	os.Exit(2)
}

func run(ctx context.Context, conf RunConfig) {
	if conf.Program == "" {
		runError(errors.New("no executable path provided to run"))
	}

	var endpoints []guardian.Endpoint
	if conf.Server != "" {
		endpoints = append(endpoints, guardian.Endpoint(conf.Server))
	} else {
		var err error
		endpoints, err = collectEndpoints(ctx)
		if err != nil {
			runError(err)
		}
	}

	client, err := guardian.NewClient(endpoints...)
	if err != nil {
		runError(fmt.Errorf("unable to create resourceful guardian client: %v", err))
	}

	err = runner.Run(ctx, client, runner.Config{
		Icon:    programIcon(),
		Program: conf.Program,
		Args:    conf.Args,
	})
	if err != nil {
		runError(err)
	}
}

// splitEndpointArgs extracts a single -s argument from the start of the arg
// list if present and interpets it as a guardian endpoint. Any remaining
// arguments are returned and will be passed to the executable being run.
func splitEndpointArgs(combined []string) (endpoints []guardian.Endpoint, args []string) {
	args = combined
	for len(args) > 2 && args[0] == "-s" && args[1] != "" {
		endpoints = append(endpoints, guardian.Endpoint(args[1]))
		args = args[2:]
	}
	return
}
