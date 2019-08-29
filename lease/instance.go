package lease

// An Instance identifies a specific instance of a lease consumer.
type Instance struct {
	Host string `json:"host,omitempty"`
	User string `json:"user,omitempty"`
	ID   string `json:"id,omitempty"`
}

// Empty returns true if the instance is its zero value.
func (i Instance) Empty() bool {
	return i.Host == "" && i.User == "" && i.ID == ""
}

// String returns a string representation of the instance.
func (i Instance) String() string {
	return i.Host + " " + i.User + " " + i.ID
}
