package lease

// Subject describes what a lease consumes and what consumes it.
type Subject struct {
	Resource string   `json:"resource"` // The resource that is being consumed
	Instance Instance `json:"instance"` // The thing that is consuming the resource
}

// HostUser returns a combination of host and user.
func (s Subject) HostUser() string {
	return s.Instance.Host + " " + s.Instance.User
}

// Empty returns true if the subject is its zero value.
func (s Subject) Empty() bool {
	return s.Resource == "" && s.Instance.Empty()
}

// String returns a string representation of the subject.
func (s Subject) String() string {
	if s.Resource == "" {
		return s.Instance.String()
	}
	return s.Resource + ": " + s.Instance.String()
}
