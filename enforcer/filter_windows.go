// +build windows

package enforcer

import (
	"errors"
	"fmt"
	"strings"

	"github.com/gentlemanautomaton/winproc"
	"github.com/scjalliance/resourceful/lease"
	"github.com/scjalliance/resourceful/policy"
)

// Filter returns a winproc filter that matches processes that match
// the given criteria.
//
// If no valid criteria are present a nil filter will be returned.
func Filter(criteria policy.Criteria, environment lease.Properties) (filter winproc.Filter, err error) {
	var filters []propertyFilter
	for _, c := range criteria {
		f, err := makeCriterionFilter(c)
		if err != nil {
			return nil, err
		}
		if f != nil {
			filters = append(filters, f)
		}
	}
	if len(filters) == 0 {
		return nil, nil
	}

	return func(p winproc.Process) bool {
		props := Properties(p, environment)
		for _, filter := range filters {
			if !filter(props) {
				return false
			}
		}
		return true
	}, nil
}

func makeCriterionFilter(c policy.Criterion) (filter propertyFilter, err error) {
	matcher, err := makeMatcher(c.Comparison, c.Value)
	if err != nil {
		return nil, err
	}

	return makePropertyFilter(c.Key, matcher), nil
}

type propertyFilter func(lease.Properties) bool

func makePropertyFilter(key string, matcher matcherFunc) propertyFilter {
	return func(p lease.Properties) bool {
		return matcher(p[key])
	}
}

type matcherFunc func(string) bool

func makeMatcher(comparison, value string) (f matcherFunc, err error) {
	switch comparison {
	case policy.ComparisonExact:
		return func(fieldValue string) bool {
			return fieldValue == value
		}, nil
	case policy.ComparisonIgnoreCase:
		return func(fieldValue string) bool {
			return strings.EqualFold(fieldValue, value)
		}, nil
	case policy.ComparisonRegex:
		re, err := compileRegex(value)
		if err != nil {
			return nil, err
		}
		if re == nil {
			return nil, errors.New("empty regular expression")
		}
		return func(fieldValue string) bool {
			return re.MatchString(fieldValue)
		}, nil
	default:
		return nil, fmt.Errorf("policy criteria contains unrecognized comparison type: %s", comparison)
	}
}
