package psyringe

import (
	"fmt"
	"reflect"
	"sync/atomic"
	"testing"
)

func TestHooks_NoValueForStructField(t *testing.T) {

	type TestTargetStruct struct {
		Int int
	}

	testCases := []struct {
		// Desc is the description of this test to make which test case is
		// running clear.
		Desc string
		// MakePsyringe is called with the generated hooks (see below).
		// It must return a Psyringe which will be used to inject into Target.
		//
		// Because we pass the hooks, they can be inserted into the scope
		// hierarchy or otherwise manipulated.
		MakePsyringe func(Hooks) *Psyringe
		// HandlerError is returned from the generated NoValueForStructField
		// func, and we assert that Inject returns this error.
		HandlerError error
		// A target to inject into (must be a pointer).
		Target interface{}
		// Number of times the generated NoValueForStructFieldFunc should be
		// called.
		ExpectedTimesCalled int64
		// Name of the  parent type when the generated NoValueForStructFieldFunc
		// is called.
		ExpectedParentTypeName string
		// Name of the field when the generated NoValueForStructFieldFunc is
		// called.
		ExpectedFieldName string
	}{
		{
			Desc: "single empty psyringe",
			MakePsyringe: func(hooks Hooks) *Psyringe {
				p := New()
				p.Hooks = hooks
				return p
			},
			HandlerError:           nil,
			Target:                 &TestTargetStruct{},
			ExpectedTimesCalled:    1,
			ExpectedParentTypeName: "*psyringe.TestTargetStruct",
			ExpectedFieldName:      "Int",
		},
		{
			Desc: "single empty psyringe + error",
			MakePsyringe: func(hooks Hooks) *Psyringe {
				p := New()
				p.Hooks = hooks
				return p
			},
			HandlerError:           fmt.Errorf("this error"),
			Target:                 &TestTargetStruct{},
			ExpectedTimesCalled:    1,
			ExpectedParentTypeName: "*psyringe.TestTargetStruct",
			ExpectedFieldName:      "Int",
		},
		{
			Desc: "cloned psyringe inherits hooks",
			MakePsyringe: func(hooks Hooks) *Psyringe {
				p := New()
				p.Hooks = hooks
				return p.Clone()
			},
			HandlerError:           nil,
			Target:                 &TestTargetStruct{},
			ExpectedTimesCalled:    1,
			ExpectedParentTypeName: "*psyringe.TestTargetStruct",
			ExpectedFieldName:      "Int",
		},
		{
			Desc: "scoped psyringe inherits hooks",
			MakePsyringe: func(hooks Hooks) *Psyringe {
				p := New()
				p.Hooks = hooks
				return p.Scope("testScope")
			},
			HandlerError:           nil,
			Target:                 &TestTargetStruct{},
			ExpectedTimesCalled:    1,
			ExpectedParentTypeName: "*psyringe.TestTargetStruct",
			ExpectedFieldName:      "Int",
		},
		{
			Desc: "scoped psyringe overrides hooks",
			MakePsyringe: func(hooks Hooks) *Psyringe {
				p := New()
				p.Hooks = hooks
				scoped := p.Scope("testScope")
				// Override the hook with a noop.
				scoped.Hooks.NoValueForStructField =
					func(string, reflect.StructField) error { return nil }
				return scoped
			},
			HandlerError:           nil,
			Target:                 &TestTargetStruct{},
			ExpectedTimesCalled:    0,
			ExpectedParentTypeName: "",
			ExpectedFieldName:      "",
		},
	}

	assertError := func(target interface{}, actual, expected error) error {

		if actual == nil && expected == nil {
			return nil
		}

		var expectedErrString string
		// Flesh out error to assert on full string.
		if expected != nil {
			expectedErrString = fmt.Sprintf("inject into %T target failed: %s",
				target, expected)
		}

		if actual == nil && expected != nil {
			return fmt.Errorf("got nil; want error %q", expected)
		}

		if expected == nil && actual != nil {
			return fmt.Errorf("got error %q; want nil", actual)
		}

		actualErrString := actual.Error()
		if actualErrString != expectedErrString {
			return fmt.Errorf("got error %q; want %q", actual, expected)
		}
		return nil
	}

	for _, test := range testCases {
		t.Run(test.Desc, func(t *testing.T) {
			var actualParentType string
			var actualField reflect.StructField
			var callCount int64
			handler := func(parentType string, field reflect.StructField) error {
				actualParentType = parentType
				actualField = field
				atomic.AddInt64(&callCount, 1)
				return test.HandlerError
			}

			hooks := newHooks()
			hooks.NoValueForStructField = handler

			p := test.MakePsyringe(hooks)

			target := test.Target

			actualErr := p.Inject(target)

			if err := assertError(target, actualErr, test.HandlerError); err != nil {
				t.Error(err)
			}

			if callCount != test.ExpectedTimesCalled {
				t.Errorf("called %d times; want %d", callCount, test.ExpectedTimesCalled)
			}

			expectedParentType := test.ExpectedParentTypeName
			if actualParentType != expectedParentType {
				t.Errorf("got %q; want %q", actualParentType, expectedParentType)
			}

			expectedFieldName := test.ExpectedFieldName
			if actualField.Name != expectedFieldName {
				t.Errorf("got %q; want %q", actualField.Name, expectedFieldName)
			}
		})

	}
}
