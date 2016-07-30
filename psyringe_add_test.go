package psyringe

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"
)

type named string

func TestPsyringe_Add_success(t *testing.T) {
	_, err := New(

		// objects
		//
		"a plain string",                      // string
		int(256),                              // int
		float64(256),                          // float64
		float32(256),                          // float32
		named("string"),                       // named
		&bytes.Buffer{},                       // *bytes.Buffer{}
		func() (int, int) { return 1, 2 },     // func() (int, int)
		func() (error, int) { return nil, 1 }, // func() (error, int)

		// constructors
		//
		func() io.Reader { return nil },               // io.Reader
		func() (io.Writer, error) { return nil, nil }, // io.Writer
		func() error { return nil },                   // error
	)
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
}

func TestPsyringe_Add_alreadyRegisteredError(t *testing.T) {
	e := func(a, b interface{}, expectedInjectionType string) {
		_, err := New(a, b)
		what := fmt.Sprintf("New(%T, %T)", a, b)
		if err == nil {
			t.Errorf("nil error for %s", what)
		}
		expectedError := fmt.Sprintf("injection type %s already registered",
			expectedInjectionType)
		errMessage := err.Error()
		if !strings.Contains(errMessage, expectedError) {
			t.Errorf("got error %q for %s; want %q", err.Error(),
				what, expectedError)
		}
	}

	// objects
	//
	e("hello", "world", "string")
	e(1, 2, "int")
	e(func() {}, func() {}, "func()")
	e(struct{}{}, struct{}{}, "struct {}")
	// Note: func() (string, string) is not a constructor type, thus its
	// injection type is itself. (Constructors either have one return value, or
	// one plus an error.
	e(
		func() (string, string) { return "", "" },
		func() (string, string) { return "hello", "world" },
		"func() (string, string)",
	)
	// This one illustrates that for non constructors,
	// only the value of the item counts, not any particular
	// interface it is asserted as.
	e(
		io.Reader(&bytes.Buffer{}).(io.Reader),
		io.Writer(&bytes.Buffer{}).(io.Writer),
		"*bytes.Buffer",
	)

	// constructors
	//
	e(
		func() int { return 1 },
		func() int { return 1 },
		"int",
	)
	e(
		func() int { return 1 },
		func() (int, error) { return 1, nil },
		"int",
	)
	e(
		func() (int, error) { return 1, nil },
		func() int { return 1 },
		"int",
	)
	e(
		func() interface{} { return nil },
		func() interface{} { return nil },
		"interface {}",
	)
	e(
		func() (interface{}, error) { return nil, nil },
		func() interface{} { return nil },
		"interface {}",
	)
	e(
		func() interface{} { return nil },
		func() (interface{}, error) { return nil, nil },
		"interface {}",
	)
}

func TestPsyringe_Add_nil(t *testing.T) {
	expected := "cannot add nil (argument 3)"
	err := MustNew().Add(1, "", struct{}{}, nil, func(int) interface{} { return nil })
	if err == nil {
		t.Fatalf("got nil; want error %q", expected)
	}
	actual := err.Error()
	if actual != expected {
		t.Fatalf("got error %q; want error %q", actual, expected)
	}
}
