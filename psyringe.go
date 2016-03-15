// psyringe is a lazy dependency injector for Go
package psyringe

import (
	"fmt"
	"reflect"
	"sync"
)

type (
	Psyringe struct {
		values         map[reflect.Type]reflect.Value
		ctors          map[reflect.Type]*ctor
		injectionTypes map[reflect.Type]struct{}
		ctorMutex      sync.Mutex
		debug          chan string
	}
	ctor struct {
		outType   reflect.Type
		inTypes   []reflect.Type
		construct func(in []reflect.Value) (reflect.Value, error)
		errChan   chan error
		once      sync.Once
		value     *reflect.Value
	}
	NoConstructorOrValue struct {
		ForType               reflect.Type
		ConstructorType       *reflect.Type
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

func (s *Psyringe) init() *Psyringe {
	if s.values != nil {
		return s
	}
	s.values = map[reflect.Type]reflect.Value{}
	s.ctors = map[reflect.Type]*ctor{}
	s.injectionTypes = map[reflect.Type]struct{}{}
	return s
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
	s.init()
	for _, thing := range things {
		if thing == nil {
			return fmt.Errorf("Fill requires non-nil items")
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

// DebugFunc allows you to pass a func(string) which will be sent debugging
// information as it arises. Note that this func has the ability to block
// Fill and Inject calls, so be careful, and make sure you return from the
// passed func as soon as possible.
func (s *Psyringe) DebugFunc(f func(string)) {
	s.debug = make(chan string)
	go func() {
		for {
			f(<-s.debug)
		}
	}()
}

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

func (s *Psyringe) debugf(format string, a ...interface{}) {
	if s.debug == nil {
		return
	}
	s.debug <- fmt.Sprintf(format, a...)
}
