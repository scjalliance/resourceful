package main

import (
	"time"

	"github.com/gentlemanautomaton/stathat"
)

// StatHatRecipient is a stat recipient that sends statistics to StatHat.
type StatHatRecipient struct {
	reporter stathat.StatHat
	prefix   string
}

// NewStatHatRecipient creates a new StatHat stat recipient with the given
// key.
func NewStatHatRecipient(statNamePrefix string, ezkey string) StatHatRecipient {
	return StatHatRecipient{
		reporter: stathat.New().EZKey(ezkey),
		prefix:   statNamePrefix,
	}
}

// SendResource sends the given resource statistics to StatHat.
func (r StatHatRecipient) SendResource(resource string, stats ResourceStats) error {
	if err := r.send(resource, "consumed", stats.Consumed, stats.Time); err != nil {
		return err
	}
	if err := r.send(resource, "limit", stats.Limit, stats.Time); err != nil {
		return err
	}
	if err := r.send(resource, "active", stats.Active, stats.Time); err != nil {
		return err
	}
	if err := r.send(resource, "released", stats.Released, stats.Time); err != nil {
		return err
	}
	if err := r.send(resource, "queued", stats.Queued, stats.Time); err != nil {
		return err
	}

	for user, count := range stats.Users {
		if user != "" {
			r.SendUser(resource, user, count, stats.Time)
		}
	}
	return nil
}

// SendUser sends individual user statistics to StatHat.
func (r StatHatRecipient) SendUser(resource, user string, count uint, t time.Time) error {
	return r.send(resource, "user "+user, count, t)
}

func (r StatHatRecipient) send(resource, name string, value uint, t time.Time) error {
	name = r.prefix + " " + resource + " " + name
	var err error
	for i := 0; i < 3; i++ {
		if i > 0 {
			time.Sleep(200 * time.Millisecond * time.Duration(i))
		}
		err = r.reporter.PostEZ(name, stathat.KindValue, float64(value), &t)
		if err == nil {
			return nil
		}
	}
	return err
}
