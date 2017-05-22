package lease

// Status indicates the current condition of a lease.
type Status string

const (
	// Queued indicates that a lease is pending and does not yet count against
	// the resource allocation counts.
	Queued Status = "queued"

	// Active indicates that a lease is in use and included in the resource
	// allocation counts.
	Active Status = "active"

	// Released indicates that a lease has ended and is in a state of decay,
	// during which it still is included in the resource allocation counts.
	Released Status = "released"
)

// Order returns an ordinal value reflecting the status' sort order. The order
// is:
//
//   0: Active
//   1: Released
//   2: Queued
//   3: (any invalid or unrecognized status)
func (s Status) Order() int {
	switch s {
	case Active:
		return 0
	case Released:
		return 1
	case Queued:
		return 2
	default:
		return 3
	}
}
