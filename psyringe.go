/*
Package psyringe provides an easy to use, lazy and concurrent dependency
injector.

Psyringe makes dependency injection very easy for well-written Go code. It
uses Go's type system to decide what to inject, and uses channels to orchestrate
value construction, automatically being as concurrent as your dependency graph
allows.

Psyringe does not rely on messy struct field tags nor verbose graph construction
syntax, and it keeps "magic" to an absolute minimum. It is very flexible
and has a small interface, allowing you to tailor things like scopes and object
lifetimes very easily using standard Go code.

The examples (below) should speak for themselves, but if you want a deeper
explanation of how Psyringe works, read on.

Injection Type

Values and constructors added to psyringe have an implicit "injection type".
This is the type of value that constructor or value represents in the graph. For
non-constructor values, the injection type is the type of the value itself,
determined by reflect.GetType(). For constructors, it is the type of the first
output (return) value. It is important to understand this concept, since a
single psyringe can have only one value or constructor per injection type.
`Add` will return an error if you try to register multiple values and/or
constructors that have the same injection type.

Constructors

Go does not have an explicit concept of "constructor". In Psyringe, constructors
are defined as any function that returns either a single value, or two values
where the second is an error. They can have any number of input parameters.

How Injection Works

A Psyringe knows how to populate fields in a struct with values of any injection
type that has been added to it.

When called upon to generate a value, via a call to Inject, the Psyringe
implicitly constructs a directed acyclic graph (DAG) from the constructors and
values, channelling values of each injection type into the relevant parameter
of any constructors which require it, and ultimately into any fields of that
type in the target struct which require it.

For a given Psyringe, each constructor can be called at most once. After that,
the generated value is provided directly without calling the constructor again.
Thus every value in a psyringe is effectively a singleton. The Clone method
allows taking snapshots of a Psyringe in order to re-use its constructor graph
whilst generating new values; and it is idiomatic to use multiple Psyringes with
differing scopes to inject different fields into the same object.
*/
package psyringe

import (
	"fmt"
	"reflect"
	"sync"
)

type (
	// A Psyringe holds a collection of constructors and fully realised values
	// which can be injected into structs which depend on them.
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
	// no constructor or value of that injection type being put into the
	// Psyringe.
	NoConstructorOrValue struct {
		// ForType is the type for which no constructor or value is available.
		ForType reflect.Type
		// ConstructorType is the type of the constructor function requiring a
		// value of type ForType.
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
	terror = reflect.TypeOf((*error)(nil)).Elem()
)

// New creates a new Psyringe, and adds the provided constructors and values to
// it. It returns an error if Add returns an error. See Add for more details.
func New(constructorsAndValues ...interface{}) (*Psyringe, error) {
	p := &Psyringe{}
	return p, p.Add(constructorsAndValues...)
}

// MustNew wraps New, and panics if New returns an error.
func MustNew(constructorsAndValues ...interface{}) *Psyringe {
	p, err := New(constructorsAndValues)
	if err != nil {
		panic(err)
	}
	return p
}

func noopDebug(...interface{})          {}
func noopDebugf(string, ...interface{}) {}

// init is called exactly once, and makes sure the Psyringe itself, as well as
// maps and debug funcs are not nil.
func (s *Psyringe) init() {
	if s == nil {
		*s = Psyringe{}
	}
	s.values = map[reflect.Type]reflect.Value{}
	s.ctors = map[reflect.Type]*ctor{}
	s.injectionTypes = map[reflect.Type]struct{}{}
	if s.debug == nil {
		s.debug = noopDebug
	}
	if s.debugf == nil {
		s.debugf = noopDebugf
	}
	s.debugf("Psyringe %v initialised.", s)
}

// Add adds constructors and values to the Psyringe. It returns an error if any
// pair of constructors and values have the same injection type. See package
// documentation for definition of "injection type".
//
// Add uses reflection to determine whether each passed value is a constructor
// or not. For each constructor, it then generates a generic function in terms
// of reflect.Values ready to be used by a call to Inject. As such, Add is a
// relatively expensive call. See Clone for how to avoid calling Add too often.
func (s *Psyringe) Add(constructorsAndValues ...interface{}) error {
	s.initOnce.Do(s.init)
	for i, thing := range constructorsAndValues {
		if thing == nil {
			return fmt.Errorf("cannot add nil (argument %d)", i)
		}
		if err := s.add(thing); err != nil {
			return err
		}
	}
	return nil
}

// MustAdd wraps Add and panics if Add returns an error.
func (s *Psyringe) MustAdd(constructorsAndValues ...interface{}) {
	if err := s.Add(constructorsAndValues); err != nil {
		panic(err)
	}
}

// Clone returns a bytewise clone of this Psyringe.
//
// Clone exists to provide efficiency by allowing you to Add constructors and
// values once, and then invoke them multiple times for different instances.
// This is especially important in long-running applications where the cost of
// calling Add repeatedly may get expensive.
func (s *Psyringe) Clone() *Psyringe {
	panic("Clone is not yet implemented")
}

// SetDebugFunc allows you to pass a debug function which will be sent debug
// level logs. The debug function has the same signature and semantics as
// log.Println from the standard library. If SetDebugFunc is not called, all
// debug messages are passed to a noop.
//
// If you pass nil, SetDebugFunc will revert to using the noop.
func (s *Psyringe) SetDebugFunc(f func(...interface{})) {
	if f != nil {
		s.debug = f
	} else {
		s.debug = noopDebug
	}
}

// SetDebugfFunc allows you to pass a debug function which will be sent debug
// level logs. The debug function has the same signature and semantics as
// log.Printf from the standard library. If SetDebugfFunc is not called, all
// debug messages are passed to a noop.
//
// If you pass nil, SetDebugfFunc will revert to using the noop.
func (s *Psyringe) SetDebugfFunc(f func(string, ...interface{})) {
	if f != nil {
		s.debugf = f
	} else {
		s.debugf = noopDebugf
	}
}

// Inject takes a list of targets, which must be pointers to struct types. It
// tries to inject a value for each field in each target, if a value is known
// for that field's type. All targets, and all fields in each target, are
// resolved concurrently where the graph allows. In the instance that the
// Psyringe knows no injection type for a given field's type, that field is
// passed over, leaving it with whatever value it already had.
//
// See package documentation for details on how the Psyringe injects values.
func (s *Psyringe) Inject(targets ...interface{}) error {
	if s.values == nil {
		return fmt.Errorf("not initialised; call Add before Inject")
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

// MustInject wraps Inject and panics if Inject returns an error.
func (s *Psyringe) MustInject(targets ...interface{}) {
	if err := s.Inject(targets...); err != nil {
		panic(err)
	}
}

// Test checks that all constructors' parameters are satisfied within this
// Psyringe. This method should be used in your own tests to ensure you have a
// complete graph; it should not be used outside of tests.
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

// inject just tries to inject a value for each field in target, no errors if it
// doesn't know how to inject a value for a given field's type, those fields are
// just left as-is.
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
			if fv, err := s.getValue(f.Type()); err == nil {
				f.Set(fv)
				s.debugf("populated %s.%s with %v", t, fieldName, fv)
			} else if _, ok := err.(NoConstructorOrValue); ok {
				s.debugf("not populating %s.%s: %s", t, fieldName, err)
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
		what = "fully realised value " + fmt.Sprint(thing)
		err = s.addValue(t, v)
	}
	if err != nil {
		s.debugf("error adding %s (%T): %s", what, thing, err)
	} else {
		s.debugf("added %s (%T)", what, thing)
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
				return reflect.Value{},
					fmt.Errorf("unable to create arg %d (%s) of %s constructor",
						i, inTypes[i], outType)
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
	s.ctors[c.outType] = c
	return s.registerInjectionType(c.outType)
}

func (s *Psyringe) addValue(t reflect.Type, v reflect.Value) error {
	s.values[t] = v
	return s.registerInjectionType(t)
}

func (s *Psyringe) registerInjectionType(t reflect.Type) error {
	if _, alreadyRegistered := s.injectionTypes[t]; alreadyRegistered {
		return fmt.Errorf("injection type %s already registered", t)
	}
	s.injectionTypes[t] = struct{}{}
	return nil
}
