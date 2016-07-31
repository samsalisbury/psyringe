package psyringe

import "testing"

func TestPsyringe_SetDebugFunc(t *testing.T) {
	p := New()

	var callCount Counter

	p.SetDebugFunc(func(...interface{}) { callCount.Increment() })

	p.AddErr("", 1)
	if callCount.Value() < 2 {
		t.Errorf("debug called %d times, want >1", callCount.Value())
	}

	callCountSnapshot := callCount.Value()

	p.SetDebugFunc(nil)

	p.AddErr(float64(1), uint(1))
	if callCount.Value() != callCountSnapshot {
		t.Errorf("nil did not set debug func to noop")
	}
}
