package main

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"os/user"
	"strings"
	"time"

	"github.com/scjalliance/resourceful/environment"
	"github.com/scjalliance/resourceful/guardian"
	"github.com/scjalliance/resourceful/lease"
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

	consumer, instance, env, err := detectEnvironment()
	if err != nil {
		runError(err)
	}

	client, err := guardian.NewClient("resourceful")
	if err != nil {
		runError(fmt.Errorf("unable to create resourceful guardian client: %v", err))
	}

	acquisition, err := client.Acquire(resource, consumer, instance, env)
	if err != nil {
		runError(fmt.Errorf("unable to request lease: %v", err))
	}

	consumers := strings.Join(acquisition.Leases.Environment("user.name"), ", ")
	switch acquisition.Lease.Status {
	case lease.Active:
		log.Printf("Resource request for %s granted", acquisition.Resource)
		log.Printf("Users: %s", consumers)

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
	case lease.Queued:
		log.Printf("Resource request for %s queued", acquisition.Resource)
		log.Printf("Users: %s", consumers)

		leasePendingDlg(resource, acquisition)
	default:
		log.Printf("Resource request for %s rejected", acquisition.Resource)
		log.Printf("Users: %s", consumers)
		os.Exit(-2)
	}

	release, err := client.Release(acquisition.Resource, acquisition.Consumer, acquisition.Instance)
	if err != nil {
		log.Fatal(err)
	}
	if release.Success {
		log.Printf("Resource released")
	}
}

func detectEnvironment() (consumer, instance string, env environment.Environment, err error) {
	hostname, err := os.Hostname()
	if err != nil {
		err = fmt.Errorf("unable to query hostname: %v", err)
		return
	}

	u, err := user.Current()
	if err != nil {
		err = fmt.Errorf("unable to determine current user: %v", err)
		return
	}

	consumer = fmt.Sprintf("%s %s", hostname, u.Username)
	env = make(environment.Environment)
	env["host.name"] = hostname
	env["user.uid"] = u.Uid
	env["user.username"] = u.Username
	env["user.name"] = u.Name

	instance = randomInstance(12)

	return
}

// randomInstance generates a random instance identifier of length n.
//
// Code provided by icza on Stack Overflow.
// See: https://stackoverflow.com/questions/22892120/how-to-generate-a-random-string-of-a-fixed-length-in-golang
func randomInstance(n int) string {
	const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	const (
		letterIdxBits = 6                    // 6 bits to represent a letter index
		letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
		letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
	)

	src := rand.NewSource(time.Now().UnixNano())

	b := make([]byte, n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return string(b)
}
