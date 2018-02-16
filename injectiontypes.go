package psyringe

import (
	"fmt"
	"reflect"
	"sort"
)

type injectionTypes map[reflect.Type]*injectionType

type injectionType struct {
	Ctor               *ctor
	Value              reflect.Value
	DebugAddedLocation string
}

// Keys returns a sorted slice of the reflect.Type keys of this collection.
func (its injectionTypes) Keys() []reflect.Type {
	types := make([]reflect.Type, len(its))
	i := 0
	for t := range its {
		types[i] = t
		i++
	}
	sort.Slice(types, func(i, j int) bool {
		return types[i].Name() < types[j].Name()
	})
	return types
}

func (its injectionTypes) Contains(t reflect.Type) bool {
	_, ok := its[t]
	return ok
}

func (its injectionTypes) Add(t reflect.Type, it *injectionType) error {
	if its.Contains(t) {
		return fmt.Errorf("type %s already registered", t)
	}
	its[t] = it
	return nil
}

func (its injectionTypes) Clone() injectionTypes {
	clone := make(injectionTypes, len(its))
	for t, it := range its {
		clone[t] = it.Clone()
	}
	return clone
}

func (it *injectionType) Clone() *injectionType {
	clone := *it
	if it.Ctor != nil {
		clone.Ctor = it.Ctor.clone()
	}
	if (it.Value != reflect.Value{}) {
		clone.Value = it.Value
	}
	return &clone
}

// Where filters injectionTypes by the predicate.
func (its injectionTypes) Where(predicate func(reflect.Type, *injectionType) bool) injectionTypes {
	filtered := injectionTypes{}
	for t, it := range its {
		if predicate(t, it) {
			filtered[t] = it
		}
	}
	return filtered
}

// AddedAsCtors filters injection types to those representing constructors (note: they
// may have already been called and also have a value if so.
func (its injectionTypes) AddedAsCtors() injectionTypes {
	return its.Where(func(_ reflect.Type, it *injectionType) bool {
		return it.Ctor != nil
	})
}

// AddedAsValues filters injection types to those representing values (note: this
// excludes values created internally by constructors.
func (its injectionTypes) AddedAsValues() injectionTypes {
	return its.Where(func(_ reflect.Type, it *injectionType) bool {
		return it.Ctor == nil
	})
}

func (its injectionTypes) WithRealisedValues() injectionTypes {
	return its.Where(func(_ reflect.Type, it *injectionType) bool {
		return it.Value.IsValid()
	})
}
