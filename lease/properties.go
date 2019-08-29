package lease

// Properties is a key/value map of lease properties.
type Properties map[string]string

// Clone returns a deep clone of the lease properties.
func (p Properties) Clone() Properties {
	to := make(Properties, len(p))
	for k, v := range p {
		to[k] = v
	}
	return to
}

// MergeProperties merges the given properties into a single property map.
// The source properties are not modified.
func MergeProperties(props ...Properties) (merged Properties) {
	merged = make(Properties)
	for p := range props {
		for k, v := range props[p] {
			merged[k] = v
		}
	}
	return
}
