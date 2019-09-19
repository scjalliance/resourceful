package main

import "gopkg.in/alecthomas/kingpin.v2"

// ListCommand returns a list command and configuration for app.
func ListCommand(app *kingpin.Application) (*kingpin.CmdClause, *ListConfig) {
	cmd := app.Command("list", "Lists running processes that match current policies.")
	conf := &ListConfig{}
	conf.Bind(cmd)
	return cmd, conf
}

// ListConfig holds configuration for the list command.
type ListConfig struct {
	Server string
}

// Bind binds the list configuration to the command.
func (conf *ListConfig) Bind(cmd *kingpin.CmdClause) {
	cmd.Flag("server", "Guardian policy server host and port.").Short('s').StringVar(&conf.Server)
}
