//go:build windows
// +build windows

package enforcerui

import (
	"encoding/json"
	"io"
)

// Writer writes messages to an underlying writer.
type Writer struct {
	encoder *json.Encoder
}

// NewWriter returns a new user interface message writer.
func NewWriter(w io.Writer) Writer {
	return Writer{encoder: json.NewEncoder(w)}
}

// Read returns the next message from r, or an error.
func (w Writer) Write(msg Message) error {
	return w.encoder.Encode(&msg)
}
