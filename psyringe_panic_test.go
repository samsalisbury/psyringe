package psyringe

import (
	"fmt"
	"regexp"
	"testing"
)

type HasIntField struct {
	Int int
}

var panickers = map[string]func(){ // These tests are very brittle; see note 1 below.
	// New
	`^adding constructor func\(\) int failed: injection type int already registered at .*/psyringe_panic_test.go:16$`: func() {
		New(func() int { return 0 }, func() int { return 1 }) // panics
	},
	// NewErr
	`^adding constructor func\(\) int failed: injection type int already registered at .*/psyringe_panic_test.go:20$`: func() {
		if _, err := NewErr(func() int { return 0 }, func() int { return 1 }); err != nil {
			panic(err)
		}
		panic("inconclusive: NewErr did not return error as expected")
	},
	// Add
	`^adding constructor func\(\) struct \{\} failed: injection type struct \{\} already registered at .*/psyringe_panic_test.go:27`: func() {
		p, err := NewErr(func() (struct{}, error) { return struct{}{}, nil })
		if err != nil {
			panic("inconclusive; New failed: " + err.Error())
		}
		p.Add(func() (s struct{}) { return }) // panics
	},
	// AddErr
	`^adding constructor func\(\) struct \{\} failed: injection type struct \{\} already registered at .*/psyringe_panic_test.go:35`: func() {
		p, err := NewErr(func() (struct{}, error) { return struct{}{}, nil })
		if err != nil {
			panic("inconclusive; New failed: " + err.Error())
		}
		if err := p.AddErr(func() (s struct{}) { return }); err != nil {
			panic(err) // panics
		}
		panic("inconclusive: AddErr did not return error as expected")
	},

	// MustInject
	`^inject into \*psyringe.HasIntField target failed: getting field Int \(int\) failed: invoking int constructor \(func\(string\) int\) failed: no constructor or value for string`: func() {
		p, err := NewErr(func(s string) int { return len(s) })
		if err != nil {
			panic("inconclusive; NewErr failed: " + err.Error())
		}
		p.MustInject(&HasIntField{})
	},
}

// Note 1: Brittle tests...
//
// These tests ensure the correct file:line is reported in many cases,
// this means that the name of this file as well as the exact line where
// tests are defined above is significant.
//
// Always add new test cases at the bottom of the panickers map above
// to avoid having to fix all the line numbers, and do not rename this
// file.

func TestPsyringe_panics(t *testing.T) {
	for expectedErr, f := range panickers {
		if err := panicsWithError(f, expectedErr); err != nil {
			t.Error(err)
		}
	}
}

func panicsWithError(f func(), expectedPattern string) (err error) {
	expected, err := regexp.Compile(expectedPattern)
	if err != nil {
		return fmt.Errorf("bad test pattern: %s", err)
	}
	defer func() {
		rec := recover()
		err = func() error {
			if rec == nil {
				return fmt.Errorf("got nil or no panic; want %q", expected)
			}
			actual, ok := rec.(error)
			if !ok {
				return fmt.Errorf("panicked with a %T: %#v; want an error: %q", rec, rec, expected)
			}
			if !expected.MatchString(actual.Error()) {
				return fmt.Errorf("panicked with error %q; did not match %q", actual, expectedPattern)
			}
			return nil
		}()
	}()
	f()
	return
}
