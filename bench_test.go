package psyringe

import (
	"bytes"
	"io"
	"testing"
	"time"
)

type benchStruct struct {
	String    string
	Int       int
	Float     float64
	Duration  time.Duration
	Interface interface{}
	Struct    struct{}
	Reader    io.Reader
	Writer    io.Writer
}

// worstCaseConstructors form a completely linear graph.
var worstCaseConstructors = []interface{}{
	func(int, float64, time.Duration, interface{}, struct{}, io.Reader, io.Writer) (string, error) {
		return "", nil
	},
	func(float64, time.Duration, interface{}, struct{}, io.Reader, io.Writer) (int, error) { return 1, nil },
	func(time.Duration, interface{}, struct{}, io.Reader, io.Writer) (float64, error) { return 2.2, nil },
	func(interface{}, struct{}, io.Reader, io.Writer) (time.Duration, error) { return 1 * time.Second, nil },
	func(struct{}, io.Reader, io.Writer) (interface{}, error) { return nil, nil },
	func(io.Reader, io.Writer) (struct{}, error) { return struct{}{}, nil },
	func(io.Reader) (io.Writer, error) { return &bytes.Buffer{}, nil },
	func() (io.Reader, error) { return &bytes.Buffer{}, nil },
}

// bestCaseConstructors form a very shallow graph.
var bestCaseConstructors = []interface{}{
	func() (string, error) { return "", nil },
	func() (int, error) { return 1, nil },
	func() (float64, error) { return 2.2, nil },
	func() (time.Duration, error) { return 1 * time.Second, nil },
	func() (interface{}, error) { return nil, nil },
	func() (struct{}, error) { return struct{}{}, nil },
	func() (io.Writer, error) { return &bytes.Buffer{}, nil },
	func() (io.Reader, error) { return &bytes.Buffer{}, nil },
}

func BenchmarkNew_WorstCase(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := New(worstCaseConstructors...)
		if err != nil {
			b.Fatal(err)
		}
	}
}
func BenchmarkNew_BestCase(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := New(bestCaseConstructors...)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkClone_WorstCase(b *testing.B) {
	p, err := New(worstCaseConstructors...)
	if err != nil {
		b.Fatal(err)
	}
	for i := 0; i < b.N; i++ {
		p.Clone()
	}
}
func BenchmarkClone_BestCase(b *testing.B) {
	p, err := New(bestCaseConstructors...)
	if err != nil {
		b.Fatal(err)
	}
	for i := 0; i < b.N; i++ {
		p.Clone()
	}
}

// BenchmarkMustInject is testing the same psyringe each time. We do a single
// injection before the loop starts to initialise all values. This probably
// isn't a very likely scenario in real life.
func BenchmarkMustInject_WorstCase(b *testing.B) {
	p := MustNew(worstCaseConstructors...)
	s := benchStruct{}
	p.MustInject(&s)
	for i := 0; i < b.N; i++ {
		p.MustInject(&s)
	}
}

func BenchmarkMustInject_BestCase(b *testing.B) {
	p := MustNew(bestCaseConstructors...)
	s := benchStruct{}
	p.MustInject(&s)
	for i := 0; i < b.N; i++ {
		p.MustInject(&s)
	}
}

// BenchmarkNewMustInject is testing the complete cycle of creating a new
// psyringe and injecting with it. This benchmark exists primarily to compare
// New with Clone.
func BenchmarkNewMustInject_WorstCase(b *testing.B) {
	s := benchStruct{}
	for i := 0; i < b.N; i++ {
		p := MustNew(worstCaseConstructors...)
		p.MustInject(&s)
	}
}

func BenchmarkNewMustInject_BestCase(b *testing.B) {
	s := benchStruct{}
	for i := 0; i < b.N; i++ {
		p := MustNew(bestCaseConstructors...)
		p.MustInject(&s)
	}
}

// BenchmarkCloneMustInject is testing the cycle of cloning a new psyringe from
// an existing one and injecting with it. This is a likely scenario in a web
// server, for example, where certain resources must be created on each request.
func BenchmarkCloneMustInject_WorstCase(b *testing.B) {
	p := MustNew(worstCaseConstructors...)
	s := benchStruct{}
	for i := 0; i < b.N; i++ {
		p.Clone().MustInject(&s)
	}
}

func BenchmarkCloneMustInject_BestCase(b *testing.B) {
	p := MustNew(bestCaseConstructors...)
	s := benchStruct{}
	for i := 0; i < b.N; i++ {
		p.Clone().MustInject(&s)
	}
}
