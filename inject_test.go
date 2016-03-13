package syringe_test

import (
	"bytes"
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
	s.Fill(1, "hello", bytes.NewBuffer([]byte("world")))
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
	s.Fill(newInt, newString, newBuffer)

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
