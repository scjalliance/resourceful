package policy

// Provider is a source of policies.
type Provider interface {
	Policies() (Set, error)
}
