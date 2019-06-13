// +build windows

package enforcer

import (
	"github.com/gentlemanautomaton/winproc"
	"github.com/scjalliance/resourceful/policy"
)

// Scan returns the set of running processes for which one or more policies
// should be applied.
//
// TODO: Accept an environment to be used in policy evaluation?
func Scan(policies policy.Set) ([]winproc.Node, error) {
	// Use resource criteria from the policies to build up a set of
	// process filters
	var filters []winproc.Filter
	for _, pol := range policies {
		if filter, ok := Filter(pol.Criteria); ok {
			filters = append(filters, filter)
		}
	}

	// Exit early if no policies with resource criteria are in effect
	if len(filters) == 0 {
		return nil, nil
	}

	// Build up a set of process collection options
	opts := []winproc.CollectionOption{
		winproc.IncludeAncestors,
		winproc.CollectCommands,
	}

	for _, filter := range filters {
		opts = append(opts, winproc.Include(filter))
	}

	// Perform the process tree collection
	nodes, err := winproc.Tree(opts...)
	if err != nil {
		return nil, err
	}

	// TODO: Perform full policy evaluation?

	return nodes, nil
}
