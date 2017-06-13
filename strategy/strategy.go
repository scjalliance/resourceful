package strategy

// Strategy is a resource counting strategy for lease statistics.
type Strategy string

const (
	// Empty indicates that a resource counting strategy has not been specified.
	Empty Strategy = ""

	// Instance is a resource counting strategy that tallies the number of
	// instances.
	Instance Strategy = "instance"

	// Consumer is a resource counting strategy that tallies the number of
	// consumers. Consumers with more than one instance are only counted once.
	Consumer Strategy = "consumer"
)

// Valid returns true if the resource counting strategy is valid.
// The empty strategy is considered valid.
func Valid(strategy Strategy) bool {
	switch strategy {
	case Empty, Consumer, Instance:
		return true
	default:
		return false
	}
}
