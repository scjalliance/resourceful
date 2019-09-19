package main

import "gopkg.in/alecthomas/kingpin.v2"

// EnforceCommand returns an enforce command and configuration for app.
func EnforceCommand(app *kingpin.Application) (*kingpin.CmdClause, *EnforceConfig) {
	cmd := app.Command("enforce", "Enforces resourceful policies on the local machine.")
	conf := &EnforceConfig{}
	conf.Bind(cmd)
	return cmd, conf
}

// EnforceConfig holds configuration for the enforce command.
type EnforceConfig struct {
	Server      string
	Passive     bool
	Interactive bool
}

// Bind binds the enforce configuration to the command.
func (conf *EnforceConfig) Bind(cmd *kingpin.CmdClause) {
	cmd.Flag("server", "Guardian policy server host and port.").Short('s').StringVar(&conf.Server)
	cmd.Flag("passive", "Run passively without killing processes.").Short('p').BoolVar(&conf.Passive)
	cmd.Flag("interactive", "Force service to run interactively.").Short('i').BoolVar(&conf.Interactive)
}
