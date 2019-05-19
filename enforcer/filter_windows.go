// +build windows

package enforcer

import (
	"regexp"

	"github.com/gentlemanautomaton/winproc"
	"github.com/scjalliance/resourceful/policy"
)

// Filter returns a winproc filter that matches processes that match
// the given criteria. Only resource criteria will be evaluated by
// the filter.
//
// If no resource criteria are present a nil filter and false will be
// returned.
func Filter(criteria policy.Criteria) (filter winproc.Filter, ok bool) {
	var filters []winproc.Filter
	for _, c := range criteria {
		switch c.Component {
		case policy.ComponentResource:
			switch c.Comparison {
			case policy.ComparisonExact, policy.ComparisonIgnoreCase:
				filters = append(filters, winproc.EqualsName(c.Value))
			case policy.ComparisonRegex:
				re, err := regexp.Compile(c.Value)
				if err != nil {
					return nil, false
				}
				filters = append(filters, winproc.MatchName(re.MatchString))
			}
		}
	}
	if len(filters) == 0 {
		return nil, false
	}
	return winproc.MatchAll(filters...), true
}
