package psyringe

import (
	"fmt"
	"reflect"
	"sync"
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

func (c ctor) clone() *ctor {
	c.once = &sync.Once{}
	c.errChan = make(chan error)
	return &c
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
			ConstructorParamIndex: paramIndex,
			ConstructorType:       c.funcType,
		}
	}
	return nil
}

func (c *ctor) getValue(s *Psyringe) (reflect.Value, error) {
	go c.once.Do(func() { c.manifest(s) })
	if err := <-c.errChan; err != nil {
		return reflect.Value{}, err
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
