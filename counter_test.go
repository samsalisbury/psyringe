package psyringe

import (
	"strconv"
	"sync/atomic"
)

// Counter wraps an int64 and provides methods for atomically incrementing and
// reading that value. It is used in tests only.
type Counter struct {
	value int64
}

// Increment atomically adds one to the counter and returns its new value.
func (c *Counter) Increment() int64 {
	return atomic.AddInt64(&c.value, 1)
}

// Value atomically gets the current value of the counter.
func (c *Counter) Value() int64 {
	return atomic.LoadInt64(&c.value)
}

// String atomically returns the current value as a base 10 integer.
func (c *Counter) String() string {
	return strconv.FormatInt(c.Value(), 10)
}
