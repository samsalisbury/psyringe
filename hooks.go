package psyringe

import "reflect"

// Hooks describe a set of event hooks which are called under certain
// circumstances during injection.
//
// All hooks may be called concurrently.
type Hooks struct {
	NoValueForStructField NoValueForStructFieldFunc
}

// NoValueForStructFieldFunc is called for each field in a struct passed to
// Inject for which there is no value or constructor in the graph.
//
// parentTypeName is the name of the type of struct that owns the field.
//
// field is the field in question.
//
// If you return an error, that is an injection error and returned by
// Inject.
type NoValueForStructFieldFunc func(parentTypeName string, field reflect.StructField) error

// newHooks returns noop hooks to avoid the need to check for nil during
// injection.
func newHooks() Hooks {
	return Hooks{
		NoValueForStructField: func(string, reflect.StructField) error { return nil },
	}
}
