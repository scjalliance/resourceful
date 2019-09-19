package main

import "gopkg.in/alecthomas/kingpin.v2"

// InstallCommand returns an install command and configuration for app.
func InstallCommand(app *kingpin.Application) (*kingpin.CmdClause, *EnforceConfig) {
	cmd := app.Command("install", "Installs the resourceful enforcer service on the local machine.")
	conf := &EnforceConfig{}
	conf.Bind(cmd)
	return cmd, conf
}
