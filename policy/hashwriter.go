package policy

import (
	"bufio"
	"encoding/binary"
	"time"
)

type hashWriter struct {
	*bufio.Writer
}

func (w hashWriter) WriteInt(v int) {
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], uint64(v))
	w.Write(buf[:])
}

func (w hashWriter) WriteDuration(v time.Duration) {
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], uint64(v))
	w.Write(buf[:])
}

func (w hashWriter) WriteString(s string) {
	w.WriteInt(len(s))
	w.Writer.WriteString(s)
}
