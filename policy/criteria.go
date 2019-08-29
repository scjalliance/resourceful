package policy

import (
	"regexp"
	"strings"

	"github.com/scjalliance/resourceful/lease"
)

// Criteria describes a set of conditions for a policy to be applied.
type Criteria []Criterion

// Match returns true if the criteria match the provided lease properties.
func (c Criteria) Match(props lease.Properties) bool {
	if len(c) == 0 {
		return false
	}
	for _, criterion := range c {
		if !criterion.Match(props) {
			return false
		}
	}
	return true
}

// String returns a string representation of the criteria.
func (c Criteria) String() string {
	parts := make([]string, 0, len(c))
	for i := range c {
		parts = append(parts, c[i].String())
	}
	return strings.Join(parts, "⋅")
}

// Criterion describes a single condition required for a policy to match.
type Criterion struct {
	Key        string `json:"key"`        // The lease property to be examined
	Comparison string `json:"comparison"` // The operator of the comparison
	Value      string `json:"value"`      // The value the property will be compared with

	// TODO: cache compiled regular expressions?
}

// Match returns true if the given process is a match.
func (c *Criterion) Match(props lease.Properties) bool {
	value := props[c.Key]

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

// String returns a string representation of the criterion.
func (c *Criterion) String() string {
	// Key
	output := c.Key

	// Operator
	switch c.Comparison {
	case ComparisonExact:
		output += "="
	case ComparisonIgnoreCase:
		output += "≈"
	case ComparisonRegex:
		output += "~"
	default:
		output += "." + c.Comparison + "."
	}

	// Value
	output += c.Value

	return output
}
