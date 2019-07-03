package lease

// Subject describes the subject of a lease.
type Subject struct {
	Resource string
	Consumer string
	Instance string
}

// String returns a string representation of the subject.
func (s Subject) String() string {
	return s.Instance + " " + s.Consumer + " " + s.Resource
}
