package psyringe

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

type dependent struct {
	Int    int
	String string
	Buffer *bytes.Buffer
}

func TestPsyringe_Inject_objects(t *testing.T) {
	s, err := NewErr(1, "hello", bytes.NewBuffer([]byte("world")))
	if err != nil {
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

func TestPsyringe_Inject_constructors(t *testing.T) {
	newInt := func() int { return 2 }
	newString := func() (string, error) { return "hello", nil }
	newBuffer := func() *bytes.Buffer { return bytes.NewBuffer([]byte("world")) }

	s, err := NewErr(newInt, newString, newBuffer)
	if err != nil {
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

func TestPsyringe_Inject_mixed(t *testing.T) {
	newString := func() (string, error) { return "hello", nil }
	newBuffer := func(s string, i int) *bytes.Buffer {
		return bytes.NewBuffer([]byte(fmt.Sprintf("%s %s %d", s, "world", i)))
	}

	s, err := NewErr(newBuffer, 100, newString)
	if err != nil {
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
	if d.Buffer.String() != "hello world 100" {
		t.Errorf("*bytes.Buffer constructor not injected")
	}
}

var uninjectables = map[string]interface{}{
	"inject into int target failed: target must be a pointer":            1,
	`inject into *int target failed: target must be a pointer to struct`: new(int),
	"inject into *struct {} target failed: target is nil":                ((*struct{})(nil)),
}

func TestPsyringe_Inject_uninjectable(t *testing.T) {
	for expected, uninjectable := range uninjectables {
		err := New().Inject(uninjectable)
		if err == nil {
			t.Errorf("got nil; want error %q", expected)
		}
		actual := err.Error()
		if actual != expected {
			t.Errorf("got error %q; want %q", actual, expected)
		}
	}
}

func TestPsyringe_Inject_customErrors(t *testing.T) {
	newString := func() (string, error) {
		return "", fmt.Errorf("an error")
	}

	s, err := NewErr(newString)
	if err != nil {
		t.Fatal(err)
	}

	d := dependent{}

	err = s.Inject(&d)
	if err == nil {
		t.Fatal("constructor error not returned")
	}

	actual := err.Error()
	expected := `inject into *psyringe.dependent target failed: getting field String (string) failed: invoking string constructor (func() (string, error)) failed: an error`

	if actual != expected {
		t.Errorf("got error %q; want %q", actual, expected)
	}
}

type dependsOnDependent struct {
	Dependency dependent
}

func TestPsyringe_Inject_dependencyTree(t *testing.T) {
	var (
		// We want to monitor that each constructor is only called
		// once. Since we know the tree is resolved concurrently where
		// possible, we need to synchronise the counters...
		newDependentCounter,
		newStringCounter,
		newIntCounter,
		newBufferCounter Counter

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
		newDependentCounter.Increment()
		newDependentBufferRef = b
		return dependent{
			Int:    i,
			String: s,
			Buffer: b,
		}
	}
	newString := func(i int, b *bytes.Buffer) string {
		newStringCounter.Increment()
		newStringBufferRef = b
		return strings.Repeat(b.String(), i)
	}
	newInt := func(b *bytes.Buffer) int {
		newIntCounter.Increment()
		newIntBufferRef = b
		return b.Len()
	}
	newBuffer := func() *bytes.Buffer {
		newBufferCounter.Increment()
		originalBufferRef = bytes.NewBuffer([]byte("yes"))
		return originalBufferRef
	}

	s, err := NewErr(newDependent, newString, newInt, newBuffer)
	if err != nil {
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
	if newDependentCounter.Value() != 1 {
		t.Errorf("newDependent was executed %d times; expected 1", newDependentCounter)
	}
	if newStringCounter.Value() != 1 {
		t.Errorf("newString was executed %d times; expected 1", newStringCounter)
	}
	if newIntCounter.Value() != 1 {
		t.Errorf("newInt was executed %d times; expected 1", newIntCounter)
	}
	if newBufferCounter.Value() != 1 {
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

func TestPsyringe_Inject_dependencyTreeErrors(t *testing.T) {
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

	s, err := NewErr(newDependent, newString, newInt, newBuffer)
	if err != nil {
		t.Fatal(err)
	}

	d := dependsOnDependent{}

	err = s.Inject(&d)
	if err == nil {
		t.Fatalf("expected an error from newString")
	}

	actual := err.Error()
	expected := "inject into *psyringe.dependsOnDependent target failed: getting field Dependency (psyringe.dependent) failed: invoking psyringe.dependent constructor (func(int, string, *bytes.Buffer) psyringe.dependent) failed: getting argument 1 failed: invoking string constructor (func(int, *bytes.Buffer) (string, error)) failed: error from newString"

	if actual != expected {
		t.Fatalf("got %q; want %q", actual, expected)
	}
}

func TestPsyringe_Inject_dependencyCycle(t *testing.T) {
	t.Skipf("This new test currently fails as we have no cycle detection.")
	type A struct{}
	type B struct{}
	type C struct{ A }
	newA := func(b B) A { return A{} }
	newB := func(a A) B { return B{} }
	p := New(newA, newB)
	done := make(chan struct{})
	go func() {
		defer func() {
			if err := recover(); err == nil {
				t.Fatal("expected panic")
			}
			close(done)
		}()
		p.Inject(&C{})
		close(done)
	}()
	<-done
}
