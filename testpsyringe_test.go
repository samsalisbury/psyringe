package psyringe

import "testing"

func TestTestPsyringe_Replace(t *testing.T) {

	type TestStruct struct {
		Int int
	}

	// Normal behaviour...
	p := New(func() int { return 1 })
	target := &TestStruct{}
	p.Inject(target)

	if target.Int != 1 {
		t.Fatalf("inconclusive, Inject broken")
	}

	// TestPsyringe behaviour...
	tp := TestPsyringe{p}
	tp.Replace(func() int { return 2 })
	tp.Inject(target)

	const expectedInt = 2
	if target.Int != expectedInt {
		t.Fatalf("got %d; want %d", target.Int, expectedInt)
	}
}
