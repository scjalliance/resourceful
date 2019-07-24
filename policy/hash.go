package policy

import (
	"encoding/base64"
)

// Hash is a 224-bit policy hash.
type Hash [28]byte

// String returns a string representation of the hash.
func (h Hash) String() string {
	return base64.RawURLEncoding.EncodeToString(h[:])
}
