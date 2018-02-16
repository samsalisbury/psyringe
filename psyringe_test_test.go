package psyringe

import (
	"bytes"
	"testing"
)

func TestPsyringe_Test_fails(t *testing.T) {

	newString := func(*bytes.Buffer) string { return "" }
	newInt := func() (int, error) { return 1, nil }
	newStructPtr := func(s string, b float64, i int) *struct{} { return nil }
	aBuffer := &bytes.Buffer{}

	s, err := NewErr(newString, newInt, newStructPtr, aBuffer)
	if err != nil {
		t.Fatal(err)
	}

	err = s.Test()
	if err == nil {
		t.Fatalf("expected an error, got nil")
	}

	actual := err.Error()
	expected := "unable to satisfy constructor func(string, float64, int) *struct {}: unable to satisfy param 1: no constructor or value for float64"

	if actual != expected {
		t.Errorf("\ngot  %q\nwant %q", actual, expected)
	}
}

func TestPsyringe_Test_succeeds(t *testing.T) {
	if err := New().Test(); err != nil {
		t.Fatalf("unexpected error %q", err)
	}
	newInt := func(b *bytes.Buffer) int { return b.Len() }
	newString := func(int) string { return "" }
	aBuffer := &bytes.Buffer{}
	if err := New(aBuffer, newInt, newString).Test(); err != nil {
		t.Fatalf("unexpected error %q", err)
	}
}

func TestPsyringe_Test_dependencyCycle(t *testing.T) {
	type (
		A *struct{}
		B *struct{}
		C *struct{}
		D *struct{}
		E *struct{}
		F *struct{}
	)
	p := New(
		func(F) A { return nil },
		func(A) B { return nil },
		func(B) C { return nil },
		func(C) D { return nil },
		func(D) E { return nil },
		func(E) F { return nil },
	)
	err := p.Test()
	if err == nil {
		t.Fatal("expected error")
	}
	actualErr := err.Error()
	expectedErr := "dependency cycle: psyringe.A: depends on psyringe.F: depends on psyringe.E: depends on psyringe.D: depends on psyringe.C: depends on psyringe.B: depends on psyringe.A"
	if actualErr != expectedErr {
		t.Errorf("got error:\n%q\nwant:\n%q", actualErr, expectedErr)
	}
}

func TestPsyringe_Test_dependencyCycle_outer(t *testing.T) {
	type (
		A *struct{}
		B *struct{}
		C *struct{}
		D *struct{}
		E *struct{}
		F *struct{}
	)
	p := New(
		func(B) A { return nil },
		func(C) B { return nil },
		func(D) C { return nil },
		func(E) D { return nil },
		func(F) E { return nil },
		func(E) F { return nil },
	)
	err := p.Test()
	if err == nil {
		t.Fatal("expected error")
	}
	actualErr := err.Error()
	expectedErr := "dependency cycle: psyringe.A: depends on psyringe.B: depends on psyringe.C: depends on psyringe.D: depends on psyringe.E: depends on psyringe.F: depends on psyringe.E"
	if actualErr != expectedErr {
		t.Errorf("got error:\n%q\nwant:\n%q", actualErr, expectedErr)
	}
}
