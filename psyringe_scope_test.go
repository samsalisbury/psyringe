package psyringe

import (
	"fmt"
	"testing"
)

func TestScope(t *testing.T) {

	var rootCounter, childCounter Counter

	type RootString string
	type ChildString string

	root := New(func() RootString {
		return RootString(fmt.Sprintf("root called %d time(s)", rootCounter.Increment()))
	})

	child := root.Scope("child")

	child.Add(func() ChildString {
		return ChildString(fmt.Sprintf("child called %d time(s)", childCounter.Increment()))
	})

	var target struct {
		FromRoot  RootString
		FromChild ChildString
	}

	// We always Clone the child before injecting, so we get a new value each
	// time.

	// First injection.
	child.Clone().MustInject(&target)

	// After the first injection, each constructor has been called once.
	{
		actual := target.FromRoot
		expected := RootString("root called 1 time(s)")
		if actual != expected {
			t.Errorf("got %q; want %q", actual, expected)
		}
	}
	{
		actual := target.FromChild
		expected := ChildString("child called 1 time(s)")
		if actual != expected {
			t.Errorf("got %q; want %q", actual, expected)
		}
	}

	// Second injection.
	child.Clone().MustInject(&target)

	// After the second injection, the root constructor has not been called
	// again, but the child one has been.
	{
		actual := target.FromRoot
		expected := RootString("root called 1 time(s)")
		if actual != expected {
			t.Errorf("got %q; want %q", actual, expected)
		}
	}
	{
		actual := target.FromChild
		expected := ChildString("child called 2 time(s)")
		if actual != expected {
			t.Errorf("got %q; want %q", actual, expected)
		}
	}

}