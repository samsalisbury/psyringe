package psyringe_test

import (
	"bytes"
	"testing"

	"github.com/samsalisbury/psyringe"
)

func TestPsyringe_Test(t *testing.T) {

	newString := func() string { return "" }
	newInt := func() (int, error) { return 1, nil }
	newStructPtr := func(s string, b float64, i int) *struct{} { return nil }
	aBuffer := &bytes.Buffer{}

	s, err := psyringe.New(newString, newInt, newStructPtr, aBuffer)
	if err != nil {
		t.Fatal(err)
	}

	err = s.Test()
	if err == nil {
		t.Fatalf("expected an error, got nil")
	}

	actual := err.Error()
	expected := "injection type float64 not known (calling constructor func(string, float64, int) *struct {})"

	if actual != expected {
		t.Errorf("\ngot  %q\nwant %q", actual, expected)
	}
}
