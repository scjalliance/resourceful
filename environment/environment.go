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
