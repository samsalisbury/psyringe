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

// newCtor creates a new ctor for the return type of constructor.
func newCtor(constructor reflect.Type, v reflect.Value) *ctor {
	if constructor.Kind() != reflect.Func || constructor.IsVariadic() {
		return nil
	}
	numOut := constructor.NumOut()
	if numOut == 0 || numOut > 2 || (numOut == 2 && constructor.Out(1) != terror) {
		return nil
	}
	outType := constructor.Out(0)
	numIn := constructor.NumIn()
	inTypes := make([]reflect.Type, numIn)
	for i := range inTypes {
		inTypes[i] = constructor.In(i)
	}
	construct := func(in []reflect.Value) (reflect.Value, error) {
		for i, arg := range in {
			if arg.IsValid() {
				continue
			}
			const format = "unable to create arg %d (%s) of %s constructor"
			return reflect.Value{}, fmt.Errorf(format, i, inTypes[i], outType)
		}
		out := v.Call(in)
		var err error
		if len(out) == 2 && !out[1].IsNil() {
			err = out[1].Interface().(error)
		}
		return out[0], err
	}

	return &ctor{
		funcType:  constructor,
		outType:   outType,
		inTypes:   inTypes,
		construct: construct,
		errChan:   make(chan error),
		once:      &sync.Once{},
	}
}

func (c ctor) clone() *ctor {
	c.once = &sync.Once{}
	c.errChan = make(chan error)
	return &c
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
	c.once.Do(func() { go c.manifest(p) })
	err := <-c.errChan
	if err == nil {
		return *c.value, nil
	}
	const format = "invoking %s constructor (%s) failed"
	return reflect.Value{}, errors.Wrapf(err, format, c.outType, c.funcType)
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
