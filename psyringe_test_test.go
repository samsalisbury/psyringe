package psyringe

import (
	"bytes"
	"math/rand"
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

// TestPsyringe_Test_dependencyCycle checks that all cases of dependency cycles
// are handled correctly. See testCases for details.
//
// Note that these tests also imply that we begin checking cycles on
// injectionTypes sorted by name.
func TestPsyringe_Test_dependencyCycle(t *testing.T) {
	type (
		A *struct{}
		B *struct{}
		C *struct{}
		D *struct{}
		E *struct{}
		F *struct{}
		G *struct{}
		H *struct{}
		I *struct{}
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
		{
			desc: "self cycle from chord",
			p: New(
				func(B) A { return nil },
				func(C) B { return nil },
				func(C) C { return nil },
			),
			wantErr: "dependency cycle: psyringe.A: depends on psyringe.B: depends on psyringe.C: depends on psyringe.C",
		},
		{
			desc: "two self-cycles",
			p: New(
				func(B) A { return nil },
				func(B) B { return nil },
				func(C) C { return nil }, // This one is not reached.
			),
			wantErr: "dependency cycle: psyringe.A: depends on psyringe.B: depends on psyringe.B",
		},
		{
			desc: "first arg satisfied",
			p: New(
				func(B, C) A { return nil },
				func() B { return nil },
				func(B, C) C { return nil },
			),
			wantErr: "dependency cycle: psyringe.A: depends on psyringe.C: depends on psyringe.C",
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

// TestPsyringe_Test_deterministic_output ensures that Test always returns the
// same error for a given Psyringe. This makes it easier to iteratively correct
// problems whilst running Test.
func TestPsyringe_Test_deterministic_output(t *testing.T) {
	type (
		A *struct{}
		B *struct{}
		C *struct{}
	)

	// Each test case is run 1000 times try to catch nondeterministic output.
	// It is possible but statistically quite unlikely for these tests to
	// pass erroneously.
	testCases := []struct {
		desc string
		// ctors are shuffled before each test
		ctors   []interface{}
		wantErr string
	}{
		{
			desc: "missing ctors before cycles",
			ctors: []interface{}{
				func(B) C { return nil }, // No ctor for B (Want).
				func(A) A { return nil }, // A is a self-cycle.
			},
			wantErr: "unable to satisfy constructor func(psyringe.B) psyringe.C: unable to satisfy param 0: no constructor or value for psyringe.B",
		},
		{
			desc: "missing ctors sorted",
			ctors: []interface{}{
				func(int) A { return nil }, // No ctor for int (want)
				func(int) B { return nil }, // No ctor for int.

			},
			wantErr: "unable to satisfy constructor func(int) psyringe.A: unable to satisfy param 0: no constructor or value for int",
		},
		{
			desc: "cycles sorted",
			ctors: []interface{}{
				func(A) A { return nil },
				func(B) B { return nil },
			},
			wantErr: "dependency cycle: psyringe.A: depends on psyringe.A",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			// Run Test and assertions 1000 times each.
			// This catches random map iterations.
			for i := 0; i < 1000; i++ {
				// Shuffle ctors to ensure their order of being added to the
				// Psyringe is unimportant.
				for i := range tc.ctors {
					j := rand.Intn(i + 1)
					tc.ctors[i], tc.ctors[j] = tc.ctors[j], tc.ctors[i]
				}
				p := New(tc.ctors...)
				err := p.Test()
				if err == nil {
					t.Fatalf("iter %d: got nil err; want %q", i, tc.wantErr)
				}
				gotErr := err.Error()
				if gotErr != tc.wantErr {
					t.Fatalf("iter %d: got error %q; want %q", i, gotErr, tc.wantErr)
				}
			}
		})
	}
}
