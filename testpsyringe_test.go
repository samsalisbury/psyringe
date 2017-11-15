package psyringe

import (
	"fmt"
	"testing"
)

func TestTestPsyringe_Replace(t *testing.T) {

	type TestStruct struct {
		Int int
	}

	// Normal behaviour...
	p := New(func() int { return 1 })
	target := &TestStruct{}
	p.Inject(target)

	if target.Int != 1 {
		t.Fatalf("inconclusive, Inject broken")
	}

	// TestPsyringe behaviour...
	tp := TestPsyringe{p}
	tp.Replace(func() int { return 2 })
	tp.Inject(target)

	const expectedInt = 2
	if target.Int != expectedInt {
		t.Fatalf("got %d; want %d", target.Int, expectedInt)
	}
}

func TestTestPsyringe_Realise(t *testing.T) {

	testCases := []struct {
		Title              string
		Psyringe           *Psyringe
		TargetAndAssertion func() (interface{}, func() error)
		Err                error
	}{
		{
			Title: "inject int pointer",
			Psyringe: New(func() *int {
				result := 1
				return &result
			}),
			TargetAndAssertion: func() (interface{}, func() error) {
				v := 2
				val := &v
				return val, func() error {
					if val == nil {
						return fmt.Errorf("got nil; want 1")
					}
					if *val != 1 {
						return fmt.Errorf("got %d; want 1", *val)
					}
					return nil
				}
			},
		},

		{
			Title:    "inject int",
			Psyringe: New(int(1)),
			TargetAndAssertion: func() (interface{}, func() error) {
				v := 2
				val := &v
				return val, func() error {
					if val == nil {
						return fmt.Errorf("got nil; want 1")
					}
					if *val != 1 {
						return fmt.Errorf("got %d; want 1", *val)
					}
					return nil
				}
			},
		},
	}

	for i, tc := range testCases {
		if tc.Title == "" {
			t.Fatalf("test case index %d has an empty title", i)
		}
		t.Run(tc.Title, func(t *testing.T) {
			tp := TestPsyringe{Psyringe: tc.Psyringe}
			target, assertion := tc.TargetAndAssertion()
			if err := tp.Realise(target); err != nil {
				if tc.Err == nil {
					t.Fatal(err)
				}
				actualErr := err.Error()
				expectedErr := tc.Err.Error()
				if actualErr != expectedErr {
					t.Fatalf("got error %q; want %q", actualErr, expectedErr)
				}
				return
			}
			if tc.Err != nil {
				t.Fatalf("got nil error; want %q", tc.Err.Error())
			}
			if err := assertion(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
