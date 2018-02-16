package psyringe

import (
	"bytes"
	"testing"
)

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

func TestPsyringe_Test_missing_injectionType(t *testing.T) {

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

func TestPsyringe_Test_dependencyCycle(t *testing.T) {
	type (
		A *struct{}
		B *struct{}
		C *struct{}
	)
	testCases := []struct {
		desc    string
		p       *Psyringe
		wantErr string
	}{
		{
			desc: "full cycle",
			p: New(
				func(A) C { return nil },
				func(B) A { return nil },
				func(C) B { return nil },
			),
			wantErr: "dependency cycle: psyringe.A: depends on psyringe.B: depends on psyringe.C: depends on psyringe.A",
		},
		{
			desc: "partial cycle",
			p: New(
				func(B) A { return nil },
				func(C) B { return nil },
				func(B) C { return nil },
			),
			wantErr: "dependency cycle: psyringe.A: depends on psyringe.B: depends on psyringe.C: depends on psyringe.B",
		},
		{
			desc: "self cycle",
			p: New(
				func(A) A { return nil },
			),
			wantErr: "dependency cycle: psyringe.A: depends on psyringe.A",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			err := tc.p.Test()
			if err == nil {
				t.Fatalf("got nil error; want %q", tc.wantErr)
			}
			gotErr := err.Error()
			if gotErr != tc.wantErr {
				t.Errorf("got error:\n%q\nwant:\n%q", gotErr, tc.wantErr)
			}
		})
	}

}
