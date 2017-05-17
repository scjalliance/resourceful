package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/user"
	"strings"

	"github.com/scjalliance/resourceful/environment"
	"github.com/scjalliance/resourceful/guardian"
)

func runUsage(errmsg string) {
	fmt.Fprintf(os.Stderr,
		"%s\n\n"+
			"usage: %s %s <program>\n"+
			"       where <program> is the path to an executable program\n",
		errmsg, os.Args[0], os.Args[1])
	os.Exit(2)
}

func run(args []string) {
	if len(args) == 0 {
		runUsage("No executable path provided to run.")
	}
	resource := args[0]

	hostname, err := os.Hostname()
	if err != nil {
		log.Fatal(err)
	}

	u, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}

	consumer := fmt.Sprintf("%s %s", hostname, u.Username)
	env := make(environment.Environment)
	env["user.uid"] = u.Uid
	env["user.username"] = u.Username
	env["user.name"] = u.Name

	client, err := guardian.NewClient("resourceful")
	if err != nil {
		log.Fatal(err)
	}

	acquisition, err := client.Acquire(resource, consumer, env)
	if err != nil {
		log.Fatal(err)
	}
	consumers := strings.Join(acquisition.Leases.Environment("user.name"), ", ")
	if acquisition.Accepted {
		log.Printf("Resource request granted")
		log.Printf("Users: %s", consumers)
	} else {
		log.Printf("Resource request rejected")
		log.Printf("Users: %s", consumers)
		os.Exit(0)
	}

	cmd := exec.Command(args[0], args[1:]...)
	err = cmd.Start()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Waiting for command to finish...")
	err = cmd.Wait()
	if err != nil {
		log.Printf("Command finished with error: %v\n", err)
	} else {
		log.Printf("Command finished: %v\n", err)
	}
	release, err := client.Release(acquisition.Resource, acquisition.Consumer)
	if err != nil {
		log.Fatal(err)
	}
	if release.Success {
		log.Printf("Resource released")
	}
}
