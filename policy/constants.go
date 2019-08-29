package policy

import (
	"time"

	"github.com/scjalliance/resourceful/strategy"
)

// TODO: Consider changing these into integer-based enumerations with a custom
//       JSON codec.

// Comparison types for matching policy criteria.
const (
	ComparisonExact      = "exact"
	ComparisonIgnoreCase = "ignorecase"
	ComparisonRegex      = "regex"
)

const (
	// DefaultLimit is the limit returned for empty policy sets.
	DefaultLimit = ^uint(0)
	// DefaultDuration is the duration returned for empty policy sets.
	DefaultDuration = time.Minute * 15
	// DefaultStrategy is the default resource counting strategy.
	DefaultStrategy = strategy.Instance
)
