package main

import (
	"os"
	"path/filepath"

	"gopkg.in/alecthomas/kingpin.v2"
)

// App returns a new resourceful kingpin app without any commands.
func App() *kingpin.Application {
	app := kingpin.New(filepath.Base(os.Args[0]), "Provides lease-based management of running programs.")
	app.Interspersed(false)
	return app
}
