package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/user"
	"strings"

	"github.com/scjalliance/resourceful/environment"
	"github.com/scjalliance/resourceful/guardian"
)

func runError(err error) {
	msgBox("resourceful run error", err.Error())
	os.Exit(2)
}

func run(args []string) {
	//freeConsole()

	if len(args) == 0 {
		runError(errors.New("no executable path provided to run"))
	}
	resource := args[0]

	hostname, err := os.Hostname()
	if err != nil {
		runError(fmt.Errorf("unable to query hostname: %v", err))
	}

	u, err := user.Current()
	if err != nil {
		runError(fmt.Errorf("unable to determine current user: %v", err))
	}

	consumer := fmt.Sprintf("%s %s", hostname, u.Username)
	env := make(environment.Environment)
	env["host.name"] = hostname
	env["user.uid"] = u.Uid
	env["user.username"] = u.Username
	env["user.name"] = u.Name

	client, err := guardian.NewClient("resourceful")
	if err != nil {
		runError(fmt.Errorf("unable to create resourceful guardian client: %v", err))
	}

	acquisition, err := client.Acquire(resource, consumer, env)
	if err != nil {
		runError(fmt.Errorf("unable to request lease: %v", err))
	}
	consumers := strings.Join(acquisition.Leases.Environment("user.name"), ", ")
	if acquisition.Accepted {
		log.Printf("Resource request for %s granted", acquisition.Resource)
		log.Printf("Users: %s", consumers)
	} else {
		log.Printf("Resource request for %s rejected", acquisition.Resource)
		log.Printf("Users: %s", consumers)
		leaseRejectedDlg(resource, acquisition)
		os.Exit(0)
	}

	cmd := exec.Command(args[0], args[1:]...)
	err = cmd.Start()
	if err != nil {
		runError(err)
	}

	log.Printf("Waiting for command to finish...")

	// TODO: Renew lease every duration/2

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
