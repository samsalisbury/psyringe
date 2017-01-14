package psyringe

import "testing"

type (
	TestCloneNeedsString struct {
		String string
	}
	TestCloneNeedsInt struct {
		Int int64
	}

	beforeClone string
	afterClone  string

	TestCloneSequence struct {
		Before beforeClone
		After  afterClone
	}
)

func TestPsyringe_Clone(t *testing.T) {

	var stringCounter, intCounter Counter

	original := New(
		1,
		func() string {
			stringCounter.Increment()
			return "#" + stringCounter.String()
		},
		func() int64 {
			intCounter.Increment()
			return intCounter.Value()
		},
	)

	var thingToInject = func() interface{} { return nil }

	// Create a couple of clones.
	clone1 := original.Clone()
	clone2 := clone1.Clone()

	// Add the same thing to both.
	clone1.Add(thingToInject)
	clone2.Add(thingToInject)

	// Add different things with same injection type to both.
	clone1.Add(func() func() { return nil })
	clone2.Add(func() func() { return nil })

}

func TestPsyringe_Clone_Sequencing(t *testing.T) {
	var stringCounter Counter

	original := New()

	original.Add(func() beforeClone {
		stringCounter.Increment()
		return beforeClone("before" + stringCounter.String())
	})

	clone := original.Clone()

	var testTarget TestCloneSequence

	// injections registered before cloning should be identical
	original.MustInject(&testTarget)
	if string(testTarget.Before) != "before1" {
		t.Errorf("Expected original injection to be before1, was %s", testTarget.Before)
	}
	clone.MustInject(&testTarget)
	if string(testTarget.Before) != "before1" {
		t.Errorf("Expected clone injection to be before1, was %s", testTarget.Before)
	}

	// injections registered before cloning should be stable
	original.MustInject(&testTarget)
	if string(testTarget.Before) != "before1" {
		t.Errorf("Expected original injection to be before1, was %s", testTarget.Before)
	}
	clone.MustInject(&testTarget)
	if string(testTarget.Before) != "before1" {
		t.Errorf("Expected clone injection to be before1, was %s", testTarget.Before)
	}

	after := func() afterClone {
		stringCounter.Increment()
		return afterClone("after" + stringCounter.String())
	}

	original.Add(after)
	clone.Add(after)

	// injections registered after cloning should be distinct
	original.MustInject(&testTarget)
	if string(testTarget.After) != "after2" {
		t.Errorf("Expected original injection to be after2, was %s", testTarget.After)
	}
	clone.MustInject(&testTarget)
	if string(testTarget.After) != "after3" {
		t.Errorf("Expected clone injection to be after3, was %s", testTarget.After)
	}

	// should be stable, though
	original.MustInject(&testTarget)
	if string(testTarget.After) != "after2" {
		t.Errorf("Expected original injection to be after2, was %s", testTarget.After)
	}
	clone.MustInject(&testTarget)
	if string(testTarget.After) != "after3" {
		t.Errorf("Expected clone injection to be after3, was %s", testTarget.After)
	}

	/*
		ns := TestCloneNeedsString{}

		// Inject 999 times using the original psyringe, the string should still be
		// #1 since the string constructor is called only once for that psyringe.
		for i := 0; i < 999; i++ {
			original.MustInject(&ns)
		}
		original.MustInject(&ns)
		expected := "#1"
		if ns.String != expected {
			t.Fatalf("got %q; want %q", ns.String, expected)
		}

		// Now inject using the clone 999 times, this time the string should be
		// #2 because we called the clone's constructor, but only once.
		for i := 0; i < 999; i++ {
			clone1.MustInject(&ns)
		}
		expected = "#2"
		if ns.String != expected {
			t.Fatalf("got %q; want %q", ns.String, expected)
		}

		// For fun, let's inject with the original Psyringe again...
		original.MustInject(&ns)
		expected = "#1"
		if ns.String != expected {
			t.Fatalf("got %q; want %q", ns.String, expected)
		}

		// Inject with the second clone.
		clone2.MustInject(&ns)
		expected = "#3"
		if ns.String != expected {
			t.Fatalf("got %q; want %q", ns.String, expected)
		}
	*/

}
