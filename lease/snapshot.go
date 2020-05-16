package lease

// Snapshot includes a set of leases for a particular resource at a revision.
type Snapshot struct {
	Resource string `json:"resource"`
	Revision uint64 `json:"revision"`
	Leases   Set    `json:"leases"`
}
