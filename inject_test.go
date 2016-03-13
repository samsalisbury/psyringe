package syringe_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/samsalisbury/syringe"
)

type dependent struct {
	Int    int
	String string
	Buffer *bytes.Buffer
}

func TestInject_Objects(t *testing.T) {
	s := syringe.New()
	if err := s.Fill(1, "hello", bytes.NewBuffer([]byte("world"))); err != nil {
		t.Fatal(err)
	}

	d := dependent{}
	if err := s.Inject(&d); err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	if d.Int != 1 {
		t.Errorf("int not injected")
	}
	if d.String != "hello" {
		t.Errorf("string not injected")
	}
	if d.Buffer.String() != "world" {
		t.Errorf("*bytes.Buffer not injected")
	}
}

func TestInject_Constructors(t *testing.T) {
	newInt := func() int { return 2 }
	newString := func() (string, error) { return "hello", nil }
	newBuffer := func() *bytes.Buffer { return bytes.NewBuffer([]byte("world")) }

	s := syringe.New()
	if err := s.Fill(newInt, newString, newBuffer); err != nil {
		t.Fatal(err)
	}

	d := dependent{}

	if err := s.Inject(&d); err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	if d.Int != 2 {
		t.Errorf("int constructor not injected")
	}
	if d.String != "hello" {
		t.Errorf("string constructor not injected")
	}
	if d.Buffer.String() != "world" {
		t.Errorf("*bytes.Buffer constructor not injected")
	}
}

func TestInject_Mixed(t *testing.T) {
	newString := func() (string, error) { return "hello", nil }
	newBuffer := func() *bytes.Buffer { return bytes.NewBuffer([]byte("world")) }

	s := syringe.New()
	if err := s.Fill(newBuffer, 100, newString); err != nil {
		t.Fatal(err)
	}

	d := dependent{}

	if err := s.Inject(&d); err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	if d.Int != 100 {
		t.Errorf("int constructor not injected")
	}
	if d.String != "hello" {
		t.Errorf("string constructor not injected")
	}
	if d.Buffer.String() != "world" {
		t.Errorf("*bytes.Buffer constructor not injected")
	}
}

func TestInject_CustomErrors(t *testing.T) {
	newString := func() (string, error) {
		return "", fmt.Errorf("an error")
	}

	s := syringe.New()
	if err := s.Fill(newString); err != nil {
		t.Fatal(err)
	}

	d := dependent{}

	err := s.Inject(&d)
	if err == nil {
		t.Fatalf("constructor error not returned")
	}

	actual := err.Error()
	expected := "an error"

	if actual != expected {
		t.Errorf("got error %s; want %q", actual, expected)
	}
}
