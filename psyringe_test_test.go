package psyringe_test

import (
	"testing"

	"github.com/samsalisbury/psyringe"
)

func TestTest(t *testing.T) {

	newString := func() string { return "" }
	newInt := func() (int, error) { return 1, nil }
	newStructPtr := func(s string, b float64, i int) *struct{} { return nil }

	s, err := psyringe.New(newString, newInt, newStructPtr)
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
