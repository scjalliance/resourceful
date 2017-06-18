package runner

import "github.com/scjalliance/resourceful/lease/leaseui"

// Config holds configuration for a runner.
type Config struct {
	Icon    *leaseui.Icon
	Program string
	Args    []string
	Servers []string
}
