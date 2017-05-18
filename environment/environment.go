package environment

// Environment is a key/value storage for lease properties.
type Environment map[string]string

// Clone returns a deep clone of the environment.
func Clone(from Environment) (to Environment) {
	to = make(Environment, len(from))
	for k, v := range from {
		to[k] = v
	}
	return
}

// Merge returns each of the provided environments merged into a single map.
// The source values are not modified.
func Merge(env ...Environment) (merged Environment) {
	merged = make(Environment)
	for e := range env {
		for k, v := range env[e] {
			merged[k] = v
		}
	}
	return
}
