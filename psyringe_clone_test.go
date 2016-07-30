package psyringe

import "testing"

type (
	TestCloneNeedsString struct {
		String string
	}
	TestCloneNeedsInt struct {
		Int int64
	}
)

func TestPsyringe_Clone(t *testing.T) {

	var stringCounter, intCounter Counter

	original := MustNew(
		func() string {
			stringCounter.Increment()
			return "#" + stringCounter.String()
		},
		func() int64 {
			intCounter.Increment()
			return intCounter.Value()
		},
	)

	// Create a couple of clones.
	clone1 := original.Clone()
	clone2 := clone1.Clone()

	ns := TestCloneNeedsString{}

	// Inject 999 times using the original psyringe, the string should still be
	// #1 since the string constructor is called only once for that psyringe.
	for i := 0; i < 999; i++ {
		original.Inject(&ns)
	}
	original.Inject(&ns)
	expected := "#1"
	if ns.String != expected {
		t.Fatalf("got %q; want %q", ns.String, expected)
	}

	// Now inject using the clone 999 times, this time the string should be
	// #2 because we called the clone's constructor, but only once.
	for i := 0; i < 999; i++ {
		clone1.Inject(&ns)
	}
	expected = "#2"
	if ns.String != expected {
		t.Fatalf("got %q; want %q", ns.String, expected)
	}

	// For fun, let's inject with the original Psyringe again...
	original.Inject(&ns)
	expected = "#1"
	if ns.String != expected {
		t.Fatalf("got %q; want %q", ns.String, expected)
	}

	// Inject with the second clone.
	clone2.Inject(&ns)
	expected = "#3"
	if ns.String != expected {
		t.Fatalf("got %q; want %q", ns.String, expected)
	}

}
