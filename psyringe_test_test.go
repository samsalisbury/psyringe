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
