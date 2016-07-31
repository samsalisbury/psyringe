package psyringe

import (
	"fmt"
	"testing"
)

type HasIntField struct {
	Int int
}

var panickers = map[string]func(){
	// MustNew
	"Add failed: adding constructor func() int failed: injection type int already registered": func() {
		New(func() int { return 0 }, func() int { return 1 }) // panics
	},
	// MustAdd
	"adding constructor func() struct {} failed: injection type struct {} already registered": func() {
		p, err := NewErr(func() (struct{}, error) { return struct{}{}, nil })
		if err != nil {
			panic("inconclusive; New failed: " + err.Error())
		}
		p.Add(func() (s struct{}) { return }) // panics
	},
	// MustInject
	"inject into *psyringe.HasIntField target failed: getting field Int (int) failed: invoking int constructor (func(string) int) failed: no constructor or value for string": func() {
		p, err := NewErr(func(s string) int { return len(s) })
		if err != nil {
			panic("inconclusive; New failed: " + err.Error())
		}
		p.MustInject(&HasIntField{})
	},
}

func TestPsyringe_panics(t *testing.T) {
	for expectedErr, f := range panickers {
		if err := panicsWithError(f, expectedErr); err != nil {
			t.Error(err)
		}
	}
}

func panicsWithError(f func(), expected string) (err error) {
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
			if actual.Error() != expected {
				return fmt.Errorf("panicked with error %q; want %q", actual, expected)
			}
			return nil
		}()
	}()
	f()
	return
}
