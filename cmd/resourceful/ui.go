package main

import "gopkg.in/alecthomas/kingpin.v2"

// UICommand returns a user interface command for app.
func UICommand(app *kingpin.Application) *kingpin.CmdClause {
	return app.Command("ui", "Starts a user interface management daemon.")
}
