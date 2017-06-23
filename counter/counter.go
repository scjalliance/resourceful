package counter

import (
	"sync/atomic"
	"unsafe"
)

// Counter is an atomic counter for counting operations. Its zero-value is safe
// for use.
type Counter struct {
	// 64-bit atomic operations require 64-bit alignment, but 32-bit
	// compilers do not ensure it. So we allocate 12 bytes and then use
	// the aligned 8 bytes in them as state.
	state [12]byte
}

func (c *Counter) addr() *uint64 {
	if uintptr(unsafe.Pointer(&c.state))%8 == 0 {
		return (*uint64)(unsafe.Pointer(&c.state))
	}
	return (*uint64)(unsafe.Pointer(&c.state[4]))
}

// Add will add the given delta to the counter and return the resulting sum.
func (c *Counter) Add(delta uint64) (value uint64) {
	return atomic.AddUint64(c.addr(), delta)
}

// Value returns the current value of the counter.
func (c *Counter) Value() uint64 {
	return atomic.LoadUint64(c.addr())
}
