package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"os/user"
	"strings"
	"syscall"
	"time"

	"github.com/mitchellh/go-ps"
	"github.com/scjalliance/resourceful/environment"
	"github.com/scjalliance/resourceful/guardian"
	"github.com/scjalliance/resourceful/policy"
	"github.com/scjalliance/resourceful/provider/cacheprov"
	"github.com/scjalliance/resourceful/provider/fsprov"
	"github.com/scjalliance/resourceful/provider/memprov"
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

	switch command {
	case "run":
		args := os.Args[2:]
		if len(args) == 0 {
			usage("No executable path provided to run.")
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
	case "list":
		fmt.Println("executing list")
		var criteria policy.Criteria
		for _, target := range os.Args[2:] {
			criteria = append(criteria, policy.Criterion{Component: policy.ComponentResource, Comparison: policy.ComparisonIgnoreCase, Value: target})
		}

		pol := policy.New(1, time.Minute*5, criteria)

		procs, err := ps.Processes()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to retrieve process list: %v\n", err)
			os.Exit(2)
		}

		for _, proc := range procs {
			if pol.Match(proc.Executable(), "user", nil) {
				fmt.Printf("%v\n", proc.Executable())
			}
		}
	case "guardian":
		log.Println("Starting resourceful guardian daemon")

		wd, err := os.Getwd()
		if err != nil {
			log.Print(errors.New("unable to detect working directory"))
		}
		log.Printf("Policy source directory: %s\n", wd)

		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			waitForSignal()
			cancel()
		}()
		cfg := guardian.ServerConfig{
			ListenSpec:     ":5877",
			PolicyProvider: cacheprov.New(fsprov.New(wd)),
			LeaseProvider:  memprov.New(),
		}
		err = guardian.Run(ctx, cfg)
		log.Print(err)
	}
}

func waitForSignal() {
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	s := <-ch
	log.Printf("Got signal: %v, exiting.", s)
}
