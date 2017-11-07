package psyringe

import (
	"fmt"
	"reflect"
)

// TestPsyringe is a Psyringe for use in testing only.
// It allows individual constructors to be replaced,
// which can be useful in testing scenarios.
type TestPsyringe struct {
	*Psyringe
}

// Replace replaces a constructor or value in the graph.
// Note that if this injection type is not already registered, it panics.
// Replace breaks the singleton semantics of Psyringe, and is not recommended
// outside of testing scenarios.
//
// When replacing a constructor that has already been called, the old value that
// was generated is blown away and the replacement constructor will be called
// next time it's called on to inject.
func (tp *TestPsyringe) Replace(constructorsAndValues ...interface{}) {
	for _, thing := range constructorsAndValues {
		t := testGetInjectionType(thing)
		if _, exists := tp.Psyringe.injectionTypes[t]; !exists {
			panic(fmt.Errorf("attempt to replace injection type %s; but no such type added", t))
		}
		delete(tp.Psyringe.injectionTypes, t)
		delete(tp.Psyringe.values, t)
		delete(tp.Psyringe.debugAddedLocation, t)
		delete(tp.Psyringe.ctors, t)
		if err := tp.Psyringe.add(thing); err != nil {
			panic(err)
		}
	}
}

func testGetInjectionType(constructorOrValue interface{}) reflect.Type {
	v := reflect.ValueOf(constructorOrValue)
	t := v.Type()
	if c := newCtor(t, v); c != nil {
		return c.outType
	}
	return t
}
