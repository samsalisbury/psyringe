package psyringe_test

import (
	"bytes"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/samsalisbury/psyringe"
)

type dependent struct {
	Int    int
	String string
	Buffer *bytes.Buffer
}

func TestInject_Objects(t *testing.T) {
	s := psyringe.New()
	if err := s.Fill(1, "hello", bytes.NewBuffer([]byte("world"))); err != nil {
		t.Fatal(err)
	}

	d := dependent{}
	if err := s.Inject(&d); err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	if d.Int != 1 {
		t.Errorf("int not injected")
	}
	if d.String != "hello" {
		t.Errorf("string not injected")
	}
	if d.Buffer.String() != "world" {
		t.Errorf("*bytes.Buffer not injected")
	}
}

func TestInject_Constructors(t *testing.T) {
	newInt := func() int { return 2 }
	newString := func() (string, error) { return "hello", nil }
	newBuffer := func() *bytes.Buffer { return bytes.NewBuffer([]byte("world")) }

	s := psyringe.New()
	if err := s.Fill(newInt, newString, newBuffer); err != nil {
		t.Fatal(err)
	}

	d := dependent{}

	if err := s.Inject(&d); err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	if d.Int != 2 {
		t.Errorf("int constructor not injected")
	}
	if d.String != "hello" {
		t.Errorf("string constructor not injected")
	}
	if d.Buffer.String() != "world" {
		t.Errorf("*bytes.Buffer constructor not injected")
	}
}

func TestInject_Mixed(t *testing.T) {
	newString := func() (string, error) { return "hello", nil }
	newBuffer := func() *bytes.Buffer { return bytes.NewBuffer([]byte("world")) }

	s := psyringe.New()
	if err := s.Fill(newBuffer, 100, newString); err != nil {
		t.Fatal(err)
	}

	d := dependent{}

	if err := s.Inject(&d); err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	if d.Int != 100 {
		t.Errorf("int constructor not injected")
	}
	if d.String != "hello" {
		t.Errorf("string constructor not injected")
	}
	if d.Buffer.String() != "world" {
		t.Errorf("*bytes.Buffer constructor not injected")
	}
}

func TestInject_CustomErrors(t *testing.T) {
	newString := func() (string, error) {
		return "", fmt.Errorf("an error")
	}

	s := psyringe.New()
	if err := s.Fill(newString); err != nil {
		t.Fatal(err)
	}

	d := dependent{}

	err := s.Inject(&d)
	if err == nil {
		t.Fatal("constructor error not returned")
	}

	actual := err.Error()
	expected := "an error"

	if actual != expected {
		t.Errorf("got error %q; want %q", actual, expected)
	}
}

type dependsOnDependent struct {
	Dependency dependent
}

func TestInject_DependencyTree(t *testing.T) {
	var (
		// We want to monitor that each constructor is only called
		// once. Since we know the tree is resolved concurrently where
		// possible, we need to synchronise the counters...
		newDependentCounter,
		newStringCounter,
		newIntCounter,
		newBufferCounter uint64

		// Also monitor the reference to buffer that each constructor sees
		newDependentBufferRef,
		newStringBufferRef,
		newIntBufferRef,
		originalBufferRef *bytes.Buffer
	)

	// Here are 4 constructors. Notice how all but newBuffer take
	// a buffer as an argument. This means we can test that only
	// one buffer is ever created, even though it is a dependency
	// of many nodes in the tree.
	//
	// Each constructor atomically increments its call counter, and
	// makes a note of the buffer reference it was passed.
	newDependent := func(i int, s string, b *bytes.Buffer) dependent {
		atomic.AddUint64(&newDependentCounter, 1)
		newDependentBufferRef = b
		return dependent{
			Int:    i,
			String: s,
			Buffer: b,
		}
	}
	newString := func(i int, b *bytes.Buffer) string {
		atomic.AddUint64(&newStringCounter, 1)
		newStringBufferRef = b
		return strings.Repeat(b.String(), i)
	}
	newInt := func(b *bytes.Buffer) int {
		atomic.AddUint64(&newIntCounter, 1)
		newIntBufferRef = b
		return b.Len()
	}
	newBuffer := func() *bytes.Buffer {
		atomic.AddUint64(&newBufferCounter, 1)
		originalBufferRef = bytes.NewBuffer([]byte("yes"))
		return originalBufferRef
	}

	s := psyringe.New()
	if err := s.Fill(newDependent, newString, newInt, newBuffer); err != nil {
		t.Fatal(err)
	}

	d := dependsOnDependent{}

	if err := s.Inject(&d); err != nil {
		t.Fatal(err)
	}

	// Assert values are correct in final graph
	if d.Dependency.Int != 3 {
		t.Errorf("int not injected correctly")
	}
	if d.Dependency.Buffer.String() != "yes" {
		t.Errorf("buffer not injected correctly")
	}
	if d.Dependency.String != "yesyesyes" {
		t.Errorf("string not injected correctly")
	}

	// Assert that each constructor was called exactly once
	if newDependentCounter != 1 {
		t.Errorf("newDependent was executed %d times; expected 1", newDependentCounter)
	}
	if newStringCounter != 1 {
		t.Errorf("newString was executed %d times; expected 1", newStringCounter)
	}
	if newIntCounter != 1 {
		t.Errorf("newInt was executed %d times; expected 1", newIntCounter)
	}
	if newBufferCounter != 1 {
		t.Errorf("newBuffer was executed %d times; expected 1", newBufferCounter)
	}

	// Assert that each instance of buffer is exactly the same and not nil.
	if originalBufferRef == nil {
		t.Errorf("originalBuffer is nil")
	}
	if newDependentBufferRef != originalBufferRef {
		t.Errorf("newDependent did not receive the original buffer")
	}
	if newStringBufferRef != originalBufferRef {
		t.Errorf("newString did not receive the original buffer")
	}
	if newIntBufferRef != originalBufferRef {
		t.Errorf("newInt did not receive the original buffer")
	}
}

func TestInject_DependencyTree_Errors(t *testing.T) {
	// Here are 4 constructors. One of them returns an error...
	newDependent := func(i int, s string, b *bytes.Buffer) dependent {
		return dependent{
			Int:    i,
			String: s,
			Buffer: b,
		}
	}
	newString := func(i int, b *bytes.Buffer) (string, error) {
		return "", fmt.Errorf("error from newString")
	}
	newInt := func(b *bytes.Buffer) int {
		return b.Len()
	}
	newBuffer := func() *bytes.Buffer {
		return bytes.NewBuffer([]byte("yes"))
	}

	s := psyringe.New()
	if err := s.Fill(newDependent, newString, newInt, newBuffer); err != nil {
		t.Fatal(err)
	}

	d := dependsOnDependent{}

	err := s.Inject(&d)
	if err == nil {
		t.Fatalf("expected an error from newString")
	}

	actual := err.Error()
	expected := "error from newString"

	if actual != expected {
		t.Fatalf("got %q; want %q", actual, expected)
	}
}
