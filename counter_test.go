package psyringe

import (
	"strconv"
	"sync/atomic"
)

// Counter wraps a uint64 and provides methods for atomically incrementing and
// reading that value.
type Counter struct {
	value uint64
}

// Increment atomically adds one to the counter.
func (c *Counter) Increment() {
	atomic.AddUint64(&c.value, 1)
}

// Value atomically gets the current value of the counter.
func (c *Counter) Value() uint64 {
	return atomic.LoadUint64(&c.value)
}

// String atomically returns the current value as a base 10 integer.
func (c *Counter) String() string {
	return strconv.FormatUint(c.Value(), 10)
}
