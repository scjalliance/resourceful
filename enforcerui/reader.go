//go:build windows
// +build windows

package enforcerui

import (
	"encoding/json"
	"io"
)

// Reader reads user interface messages from an underlying reader.
type Reader struct {
	decoder *json.Decoder
}

// NewReader returns a new user interface message reader.
func NewReader(r io.Reader) Reader {
	return Reader{decoder: json.NewDecoder(r)}
}

// Read returns the next message from r, or an error.
func (r Reader) Read() (Message, error) {
	var msg Message
	err := r.decoder.Decode(&msg)
	return msg, err
}
