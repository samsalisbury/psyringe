package psyringe

import (
	"fmt"
	"reflect"

	"github.com/pkg/errors"
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
		if err := tp.Psyringe.add(thing); err != nil {
			panic(err)
		}
	}
}

// Realise takes a pointer (target) and tries to populate it with a value of the
// same type from the graph. It uses the same mechanism as populating a struct
// field when Inject is called, except the NoValueForStructField hook is never
// called.
//
// Example usage:
//
//     var target *int
//     tp.Realise(target)
//
// This can be used in tests to examine a single item in the graph.
//
// Note: Realise can only be used for pointer types.
func (tp *TestPsyringe) Realise(target interface{}) error {
	return tp.realise(target)
}

func (tp *TestPsyringe) realise(target interface{}) error {
	targetVal := reflect.ValueOf(target)
	if targetVal.Kind() != reflect.Ptr {
		return fmt.Errorf("target must be a pointer, was a %T", target)
	}
	targetType := targetVal.Type()
	if !targetVal.Elem().IsValid() {
		return fmt.Errorf("target must not be nil")
	}
	fakeParentTypeName := "<TestPsyringe.Realise>"
	fakeStructField := reflect.StructField{
		Name: fmt.Sprintf("<%T>", target),
		Type: targetType,
	}
	val, got, err := tp.Psyringe.getValueForStructField(
		newHooks(), fakeParentTypeName, fakeStructField)
	if err != nil {
		return err
	}

	if !got {
		// Try getting a value of type targetType.Elem().
		fakeStructField.Type = targetType.Elem()
		var errElem error
		val, got, errElem = tp.Psyringe.getValueForStructField(
			newHooks(), fakeParentTypeName, fakeStructField)
		if errElem != nil {
			return errors.Wrapf(err, "attempting to realise %s", targetType.Elem())
		}
		if !got {
			return fmt.Errorf("no value or constructor for %s nor %s",
				targetType, targetType.Elem())
		}
	}
	if val.Kind() == reflect.Ptr {
		targetVal.Elem().Set(val.Elem())
	} else {
		targetVal.Elem().Set(val)
	}
	return nil
}

func testGetInjectionType(constructorOrValue interface{}) reflect.Type {
	v := reflect.ValueOf(constructorOrValue)
	t := v.Type()
	if c := newCtor(t, v); c != nil {
		return c.outType
	}
	return t
}
