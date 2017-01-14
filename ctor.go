package psyringe

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/pkg/errors"
)

// ctor is a constructor for a single value.
type ctor struct {
	outType,
	funcType reflect.Type
	inTypes   []reflect.Type
	construct func(in []reflect.Value) (reflect.Value, error)
	errChan   chan error
	once      *sync.Once
	value     *reflect.Value
}

// terror is the type "error"
var terror = reflect.TypeOf((*error)(nil)).Elem()

func newCtor(t reflect.Type, v reflect.Value) *ctor {
	if t.Kind() != reflect.Func || t.IsVariadic() {
		return nil
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
		funcType:  t,
		outType:   outType,
		inTypes:   inTypes,
		construct: construct,
		errChan:   make(chan error),
		once:      &sync.Once{},
	}
}

func (c *ctor) testParametersAreRegisteredIn(s *Psyringe) error {
	for paramIndex, paramType := range c.inTypes {
		if err := s.testValueOrConstructorIsRegistered(paramType); err != nil {
			return errors.Wrapf(err, "unable to satisfy param %d", paramIndex)
		}
	}
	return nil
}

func (c *ctor) getValue(p *Psyringe) (reflect.Value, error) {
	go c.once.Do(func() { c.manifest(p) })
	if err := <-c.errChan; err != nil {
		return reflect.Value{},
			errors.Wrapf(err, "invoking %s constructor (%s) failed",
				c.outType, c.funcType)
	}
	return *c.value, nil
}

// manifest is called exactly once for each constructor to generate its value.
func (c *ctor) manifest(s *Psyringe) {
	defer close(c.errChan)
	wg := sync.WaitGroup{}
	numArgs := len(c.inTypes)
	wg.Add(numArgs)
	args := make([]reflect.Value, numArgs)
	for i, t := range c.inTypes {
		s, c, i, t := s, c, i, t
		go func() {
			defer wg.Done()
			v, err := s.getValueForConstructor(c, i, t)
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
}
