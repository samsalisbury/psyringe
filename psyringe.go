// Package psyringe is an easy to use, lazy and concurrent dependency injector.
// It's really fast too.
//
// Psyringe is unlike other Dependency injectors in the following ways:
//
//     - No need to reference psyringe everywhere.
//     - No reliance on struct field tags.
//     - Uses simple named types to define well-known instances.
//     - Dependencies are realised in parallel, automatically.
//     - Scoping is achieved via standard Go code.
//     - Supports one-hit CLI applications by having blazingly fast
//       initialisation.
//     - Supports long-running applications like HTTP servers via trivial
//       scoping and intelligent caching of reflected initialisation plans.
//     - In all cases is super easy to use.
//
// Basic usage looks something like this:
//
//     type SomeStruct {
//         Widget
//         Thing
//     }
//     p := psyringe.New()
//     p.Fill(newWidget, newThing)
//     v := SomeStruct{}
//     err := p.Inject(&v)
//     if err != nil {
//         handle(err)
//     }
package psyringe

import (
	"fmt"
	"reflect"
	"sync"
)

type (
	// A Psyringe should be filled with constructors and fully realised values.
	// It an then be used to inject these as dependencies into struct values.
	Psyringe struct {
		initOnce       sync.Once
		values         map[reflect.Type]reflect.Value
		ctors          map[reflect.Type]*ctor
		injectionTypes map[reflect.Type]struct{}
		ctorMutex      sync.Mutex
		debug          func(...interface{})
		debugf         func(string, ...interface{})
	}
	// ctor is a constructor for a single value.
	ctor struct {
		outType   reflect.Type
		inTypes   []reflect.Type
		construct func(in []reflect.Value) (reflect.Value, error)
		errChan   chan error
		once      sync.Once
		value     *reflect.Value
	}
	// NoConstructorOrValue is an error returned when Psyringe has no way of
	// injecting a value of a specific type into one of its constructors, due to
	// no constructor or value of that injection type being put into a Psyringe
	// with Psyringe.Fill.
	NoConstructorOrValue struct {
		// ForType is the type for which no constructor or value is available.
		ForType reflect.Type
		// ConstructorType is the constructor function requiring a value of type
		// ForType.
		ConstructorType *reflect.Type
		// ConstructorParamIndex is the zero-based index of the first parameter
		// in ConstructorType of type ForType.
		ConstructorParamIndex *int
	}
)

func (e NoConstructorOrValue) Error() string {
	message := ""
	if e.ConstructorType != nil {
		message += fmt.Sprintf("unable to construct %s", *e.ConstructorType)
	}
	if e.ConstructorParamIndex != nil {
		message += fmt.Sprintf(" (missing param %d)", *e.ConstructorParamIndex)
	}
	if message != "" {
		message += ": "
	}
	return message + fmt.Sprintf("no constructor or value for %s", e.ForType)
}

var (
	globalPs = &Psyringe{}
	terror   = reflect.TypeOf((*error)(nil)).Elem()
)

// New returns a new Psyringe. It is equivalent to simply using &Psyringe{}
// and may be removed soon.
func New() *Psyringe {
	return &Psyringe{}
}

// init is called exactly once, and makes sure the maps and debug funcs are not
// nil.
func (s *Psyringe) init() {
	s.values = map[reflect.Type]reflect.Value{}
	s.ctors = map[reflect.Type]*ctor{}
	s.injectionTypes = map[reflect.Type]struct{}{}
	if s.debug == nil {
		s.debug = func(...interface{}) {}
	}
	if s.debugf == nil {
		s.debugf = func(string, ...interface{}) {}
	}
}

// Fill calls Fill on the default, global Psyringe.
func Fill(things ...interface{}) error { return globalPs.Fill(things...) }

// Inject calls Inject on the default, global Psyringe.
func Inject(targets ...interface{}) error { return globalPs.Inject(targets...) }

// Fill fills the psyringe with values and constructors. Any function that
// returns a single value, or two return values, the second of which is an
// error, is considered to be a constructor. Everything else is considered to be
// a fully realised value.
func (s *Psyringe) Fill(things ...interface{}) error {
	s.initOnce.Do(s.init)
	for i, thing := range things {
		if thing == nil {
			return fmt.Errorf("Fill passed nil as argument %d", i)
		}
		if err := s.add(thing); err != nil {
			return err
		}
	}
	return nil
}

// Clone is not yet implemented. It will eventually return a deep copy of this
// psyringe.
func (s *Psyringe) Clone() *Psyringe {
	panic("Clone is not yet implemented")
}

// SetDebugFunc allows you to pass a debug function which will be sent debug
// level logs. The debug function has the same signature as log.Println from the
// standard library, so you could pass that if you wanted.
func (s *Psyringe) SetDebugFunc(f func(...interface{})) { s.debug = f }

// SetDebugfFunc allows you to pass a debug function which will be sent debug
// level logs. The debug function has the same signature as log.Printf from the
// standard library, so you could pass that if you wanted.
func (s *Psyringe) SetDebugfFunc(f func(string, ...interface{})) { s.debugf = f }

// Inject takes a list of targets, which must be pointers to struct types. It
// tries to inject a value for each field in each target, if a value is known
// for that field's type. All targets, and all fields in each target, are
// resolved concurrently.
func (s *Psyringe) Inject(targets ...interface{}) error {
	if s.values == nil {
		return fmt.Errorf("Inject called before Fill")
	}
	wg := sync.WaitGroup{}
	wg.Add(len(targets))
	errs := make(chan error)
	go func() {
		wg.Wait()
		close(errs)
	}()
	for _, t := range targets {
		go func(target interface{}) {
			defer wg.Done()
			if err := s.inject(target); err != nil {
				s.debugf("error injecting into %T: %s", target, err)
				errs <- err
			}
			s.debugf("finished injecting into %T", target)
		}(t)
	}
	return <-errs
}

// Test checks that all constructors' parameters are satisfied within this
// Psyringe. It does not invoke those constructors, it only checks that the
// structure is valid. If any constructor parameters are not satisfiable, an
// error is returned. This func should only be used in tests.
func (s *Psyringe) Test() error {
	for _, c := range s.ctors {
		if err := c.testParametersAreRegisteredIn(s); err != nil {
			return err
		}
	}
	return nil
}

func (c *ctor) testParametersAreRegisteredIn(s *Psyringe) error {
	for paramIndex, paramType := range c.inTypes {
		if _, constructorExists := s.ctors[paramType]; constructorExists {
			continue
		}
		if _, valueExists := s.values[paramType]; valueExists {
			continue
		}
		return NoConstructorOrValue{
			ForType:               paramType,
			ConstructorParamIndex: &paramIndex,
			ConstructorType:       &c.outType,
		}
	}
	return nil
}

// inject just tries to inject a value for each field, no errors if it
// fails, as maybe those other fields are just not meant to receive
// injected values
func (s *Psyringe) inject(target interface{}) error {
	v := reflect.ValueOf(target)
	ptr := v.Type()
	if ptr.Kind() != reflect.Ptr {
		return fmt.Errorf("got a %s; want a pointer", ptr)
	}
	t := ptr.Elem()
	if t.Kind() != reflect.Struct {
		return fmt.Errorf("got a %s, but %s is not a struct", ptr, t)
	}
	if v.IsNil() {
		return fmt.Errorf("got a %s, but it was nil", ptr)
	}
	nfs := t.NumField()
	wg := sync.WaitGroup{}
	wg.Add(nfs)
	errs := make(chan error)
	go func() {
		wg.Wait()
		close(errs)
	}()
	for i := 0; i < nfs; i++ {
		go func(f reflect.Value, fieldName string) {
			defer wg.Done()
			fv, err := s.getValue(f.Type())
			if err == nil {
				f.Set(fv)
				s.debugf("Inject: populated %s.%s with %v", t, fieldName, fv)
			} else if _, ok := err.(NoConstructorOrValue); ok {
				s.debugf("Inject: not populating %s.%s: %s", t, fieldName, err)
			} else {
				errs <- err
			}
		}(v.Elem().Field(i), t.Field(i).Name)
	}
	return <-errs
}

func (s *Psyringe) add(thing interface{}) error {
	v := reflect.ValueOf(thing)
	t := v.Type()
	var err error
	var what string
	if c := s.tryMakeCtor(t, v); c != nil {
		what = "constructor for " + c.outType.Name()
		err = s.addCtor(c)
	} else {
		what = "fully realised value"
		err = s.addValue(t, v)
	}
	if err != nil {
		s.debugf("Fill: error adding %s (%T): %s", what, thing, err)
	} else {
		s.debugf("Fill: added %s (%T)", what, thing)
	}
	return err
}

func (s *Psyringe) getValue(t reflect.Type) (reflect.Value, error) {
	if v, ok := s.values[t]; ok {
		return v, nil
	}
	c, ok := s.ctors[t]
	if !ok {
		return reflect.Value{}, NoConstructorOrValue{ForType: t}
	}
	return c.getValue(s)
}

func (s *Psyringe) tryMakeCtor(t reflect.Type, v reflect.Value) *ctor {
	if t.Kind() != reflect.Func || t.IsVariadic() {
		return nil
	}
	if v.IsNil() {
		panic("psyringe internal error: tryMakeCtor received a nil value")
	}
	if !v.IsValid() {
		panic("psyringe internal error: tryMakeCtor received a zero Value value")
	}
	numOut := t.NumOut()
	if numOut == 0 || numOut > 2 || (numOut == 2 && t.Out(1) != terror) {
		return nil
	}
	outType := t.Out(0)
	numIn := t.NumIn()
	inTypes := make([]reflect.Type, numIn)
	for i := range inTypes {
		inTypes[i] = t.In(i)
	}
	construct := func(in []reflect.Value) (reflect.Value, error) {
		for i, arg := range in {
			if !arg.IsValid() {
				return reflect.Value{}, fmt.Errorf("unable to create arg %d (%s) of %s constructor", i, inTypes[i], outType)
			}
		}
		out := v.Call(in)
		var err error
		if len(out) == 2 && !out[1].IsNil() {
			err = out[1].Interface().(error)
		}
		return out[0], err
	}
	return &ctor{
		outType:   outType,
		inTypes:   inTypes,
		construct: construct,
		errChan:   make(chan error),
	}
}

func (c *ctor) getValue(s *Psyringe) (reflect.Value, error) {
	if c.value != nil {
		return *c.value, nil
	}
	go c.once.Do(func() {
		defer close(c.errChan)
		wg := sync.WaitGroup{}
		numArgs := len(c.inTypes)
		wg.Add(numArgs)
		args := make([]reflect.Value, numArgs)
		for i, t := range c.inTypes {
			i, t := i, t
			go func() {
				defer wg.Done()
				v, err := s.getValue(t)
				if err != nil {
					c.errChan <- err
				}
				args[i] = v
			}()
		}
		wg.Wait()
		v, err := c.construct(args)
		if err != nil {
			c.errChan <- err
		}
		c.value = &v
	})
	if err := <-c.errChan; err != nil {
		return reflect.Value{}, err
	}
	return *c.value, nil
}

func (s *Psyringe) addCtor(c *ctor) error {
	if err := s.registerInjectionType(c.outType); err != nil {
		return err
	}
	s.ctors[c.outType] = c
	return nil
}

func (s *Psyringe) addValue(t reflect.Type, v reflect.Value) error {
	if err := s.registerInjectionType(t); err != nil {
		return err
	}
	s.values[t] = v
	return nil
}

func (s *Psyringe) registerInjectionType(t reflect.Type) error {
	if _, alreadyRegistered := s.injectionTypes[t]; alreadyRegistered {
		return fmt.Errorf("injection type %s already registered", t)
	}
	s.injectionTypes[t] = struct{}{}
	return nil
}
