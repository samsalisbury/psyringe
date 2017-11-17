package psyringe

import (
	"sync"
	"testing"
)

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

	// Create some clones concurrently (this is mainly for the benefit of the
	// race detector; You should be able to call Clone concurrently.
	var clone1, clone2, clone3, clone4 *Psyringe
	var wg sync.WaitGroup
	wg.Add(4)
	go func() {
		defer wg.Done()
		clone1 = original.Clone()
	}()
	go func() {
		defer wg.Done()
		clone2 = original.Clone()
	}()
	go func() {
		defer wg.Done()
		clone3 = original.Clone()
	}()
	go func() {
		defer wg.Done()
		clone4 = original.Clone()
	}()
	wg.Wait()

	// It's clones all the way down.
	clone2 = clone1.Clone().Clone().Clone().Clone()

	// Add the same thing to both.
	clone1.Add(thingToInject)
	clone2.Add(thingToInject)

	// Add different things with same injection type to both.
	clone1.Add(func() func() { return nil })
	clone2.Add(func() func() { return nil })

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

}

func TestPsyringe_Clone_identity(t *testing.T) {
	var target struct {
		StringPtr *string
	}
	var ctor = func() *string { s := "A String."; return &s }

	p := New(ctor)
	clone1, clone2 := p.Clone(), p.Clone()

	// clone1 and clone2 should inject different string pointers...
	clone1.Inject(&target)
	sp1 := target.StringPtr
	clone2.Inject(&target)
	sp2 := target.StringPtr

	if sp1 == sp2 {
		t.Errorf("clones injected the same pointer when they shouldn't")
	}

	clone1.MustInject(&target)
	sp11 := target.StringPtr

	clone2.MustInject(&target)
	sp22 := target.StringPtr

	if sp11 != sp1 {
		t.Errorf("cloned psyringe is not stable, ctor called twice")
	}
	if sp22 != sp2 {
		t.Errorf("cloned psyringe is not stable, ctor called twice")
	}

	// Now clone the clones after Inject was called, should still be stable.
	clone11 := clone1.Clone()
	clone22 := clone2.Clone()

	clone11.MustInject(&target)
	sp111 := target.StringPtr
	clone22.MustInject(&target)
	sp222 := target.StringPtr

	if sp111 != sp11 {
		t.Errorf("Clone after Inject did not clone the value")
	}

	if sp222 != sp22 {
		t.Errorf("Clone after Inject did not clone the value")
	}

}
