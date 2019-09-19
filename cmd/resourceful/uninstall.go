package main

import "gopkg.in/alecthomas/kingpin.v2"

// UninstallCommand returns an uninstall command for app.
func UninstallCommand(app *kingpin.Application) *kingpin.CmdClause {
	return app.Command("uninstall", "Uninstalls the resourceful enforcer service from the local machine.")
}
