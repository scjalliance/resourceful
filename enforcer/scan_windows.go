// +build windows

package enforcer

import (
	"github.com/gentlemanautomaton/winproc"
	"github.com/scjalliance/resourceful/policy"
)

// Scan returns the set of running processes for which one or more policies
// might be applicable.
//
// TODO: Accept an environment to be used in policy evaluation?
func Scan(policies policy.Set) ([]ProcessData, error) {
	// Detect the current hostname
	/*
		hostname, err := os.Hostname()
		if err != nil {
			return nil, fmt.Errorf("unable to query hostname: %v", err)
		}
	*/

	// Use resource criteria from the policies to build up a set of
	// process filters
	var filters []winproc.Filter
	for _, pol := range policies {
		filter, err := Filter(pol.Criteria)
		if err != nil {
			// Skip policiies with criteria that we couldn't understand
			// TODO: Log the policy error somewhere?
			continue
		}
		if filter != nil {
			filters = append(filters, filter)
		}
	}

	// Exit early if no policies with resource criteria are in effect
	if len(filters) == 0 {
		return nil, nil
	}

	// Prepare a composite filter that matches any policy
	filter := winproc.MatchAny(filters...)

	// Perform the process list collection
	procs, err := winproc.List(
		winproc.Include(filter), // 1. Collect processes that match policy criteria
		winproc.CollectCommands, // 2. Collect command lines for each process
		winproc.CollectSessions, // 3. Collect session data for each process
		winproc.CollectUsers,    // 4. Collect user data for each process
		winproc.CollectTimes,    // 5. Collect timing data for each process
	)
	if err != nil {
		return nil, err
	}
	return procs, nil

	// Exit early if we found no matching processes
	/*
		if len(procs) == 0 {
			return nil, nil
		}

		// Collect resource, consumer and instance information about each process
		out := make([]Process, 0, len(procs))
		for _, proc := range procs {
			out = append(out, newProcess(hostname, proc))
		}

		return out, nil
	*/
}
