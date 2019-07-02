// +build windows

package enforcer

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/gentlemanautomaton/winproc"
	"github.com/scjalliance/resourceful/policy"
)

// Filter returns a winproc filter that matches processes that match
// the given criteria.
//
// If no valid criteria are present a nil filter will be returned.
func Filter(criteria policy.Criteria) (filter winproc.Filter, err error) {
	var filters []winproc.Filter
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
	return winproc.MatchAll(filters...), nil
}

func makeCriterionFilter(c policy.Criterion) (filter winproc.Filter, err error) {
	matcher, err := makeMatcher(c.Comparison, c.Value)
	if err != nil {
		return nil, err
	}

	switch c.Component {
	case policy.ComponentResource:
		return makeResourceFilter(matcher)
	case policy.ComponentConsumer:
		return makeConsumerFilter(matcher)
	case policy.ComponentEnvironment:
		return makeEnvironmentFilter(matcher, c.Key)
	default:
		return nil, fmt.Errorf("policy criteria contains unrecognized component type: %s", c.Component)
	}
}

func makeResourceFilter(matcher matcherFunc) (filter winproc.Filter, err error) {
	return makeFieldFilter(matcher, func(p winproc.Process) string {
		return p.Name
	}), nil
}

func makeConsumerFilter(matcher matcherFunc) (filter winproc.Filter, err error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, fmt.Errorf("unable to query hostname: %v", err)
	}

	return makeFieldFilter(matcher, func(p winproc.Process) string {
		username := p.User.String()
		if username == "" {
			return hostname
		}
		return hostname + " " + username
	}), nil
}

func makeEnvironmentFilter(matcher matcherFunc, key string) (filter winproc.Filter, err error) {
	switch key {
	case "host.name":
		hostname, err := os.Hostname()
		if err != nil {
			return nil, fmt.Errorf("unable to query hostname: %v", err)
		}
		return makeStaticFilter(matcher, hostname), nil
	case "user.uid", "user.id":
		return makeFieldFilter(matcher, func(p winproc.Process) string {
			return p.User.SID
		}), nil
	case "user.username":
		return makeFieldFilter(matcher, func(p winproc.Process) string {
			return p.User.String()
		}), nil
	default:
		return nil, fmt.Errorf("policy criteria contains unrecognized environment key: %s", key)
	}
}

// matcherFunc returns true if it matches the given value.
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
		re, err := regexp.Compile(value)
		if err != nil {
			return nil, err
		}
		return func(fieldValue string) bool {
			return re.MatchString(fieldValue)
		}, nil
	default:
		return nil, fmt.Errorf("policy criteria contains unrecognized comparison type: %s", comparison)
	}
}

func makeStaticFilter(matcher matcherFunc, value string) winproc.Filter {
	return func(p winproc.Process) bool {
		return matcher(value)
	}
}

// fieldFunc returns the value of a field from a process.
type fieldFunc func(winproc.Process) string

func makeFieldFilter(matcher matcherFunc, field fieldFunc) winproc.Filter {
	return func(p winproc.Process) bool {
		return matcher(field(p))
	}
}
