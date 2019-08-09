// +build windows

package enforcer

import (
	"bufio"
	"encoding/base64"

	"github.com/gentlemanautomaton/winproc"
	"golang.org/x/crypto/sha3"
)

// NewInstance generates an instance identifier for p.
//
// The return value is not guaranteed to be deterministic.
func NewInstance(p winproc.Process) string {
	var (
		hash = sha3.New224()
		w    = hashWriter{bufio.NewWriterSize(hash, hash.BlockSize())}
		uid  = p.UniqueID().Bytes()
	)

	w.Write(uid[:])
	w.WriteString(p.User.SID)
	w.WriteString(p.Name)

	if err := w.Flush(); err != nil {
		panic(err)
	}

	var h [28]byte
	hash.Sum(h[:0])

	return base64.RawURLEncoding.EncodeToString(h[:])
}
