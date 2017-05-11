package policy

import (
	"regexp"
	"strings"

	"github.com/scjalliance/resourceful/environment"
)

// Criteria describes a set of conditions for a policy to be applied.
type Criteria []Criterion

// Match returns true if all of the criteria match the provided resource,
// consumer and environment.
func (c Criteria) Match(resource, consumer string, env environment.Environment) bool {
	if len(c) == 0 {
		return false
	}
	for _, criterion := range c {
		if !criterion.Match(resource, consumer, env) {
			return false
		}
	}
	return true
}

// Criterion describes a single condition required for a policy to match.
type Criterion struct {
	Component  string `json:"component"`  // The operand of the comparison
	Key        string `json:"key"`        // The key when operating on an environmental component
	Comparison string `json:"comparison"` // The operator of the comparison
	Value      string `json:"value"`      // The value for comparison

	// TODO: cache compiled regular expressions?
}

// Match returns true if the given process is a match.
func (c *Criterion) Match(resource string, consumer string, env environment.Environment) bool {
	var value string

	switch c.Component {
	case ComponentResource:
		value = resource
	case ComponentConsumer:
		value = consumer
	case ComponentEnvironment:
		value = env[c.Key]
	default:
		return false
	}

	switch c.Comparison {
	case ComparisonExact:
		return c.Value == value
	case ComparisonIgnoreCase:
		return strings.ToLower(c.Value) == strings.ToLower(value)
	case ComparisonRegex:
		re, err := regexp.Compile(c.Value)
		if err != nil {
			// TODO: Log error
			return false
		}
		return re.MatchString(value)
	default:
		return false
	}
}
