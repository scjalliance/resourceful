package main

import (
	"fmt"
	"os"
	"strings"
)

func usage(errmsg string) {
	fmt.Fprintf(os.Stderr,
		"%s\n\n"+
			"usage: %s <command>\n"+
			"       where <command> is one of\n"+
			"       run, list, guardian.\n",
		errmsg, os.Args[0])
	os.Exit(2)
}

func main() {
	if len(os.Args) < 2 {
		usage("No command specified.")
	}

	command := strings.ToLower(os.Args[1])
	args := os.Args[2:]

	switch command {
	case "run":
		run(args)
	case "list":
		list(args)
	case "daemon", "guardian":
		err := daemon(strings.Join(os.Args[0:2], " "), args)
		if err != nil {
			os.Exit(2)
		}
	default:
		usage(fmt.Sprintf("\"%s\" is an unrecognized command.", command))
	}
}
