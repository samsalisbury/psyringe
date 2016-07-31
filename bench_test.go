package psyringe

import (
	"bytes"
	"io"
	"testing"
	"time"
)

type BenchStruct struct {
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

// Some package-level variables to make sure compiler doesn't optimise away
// calls to New, Clone or Inject.
// See http://dave.cheney.net/2013/06/30/how-to-write-benchmarks-in-go
var (
	P *Psyringe
	S BenchStruct
)

func BenchmarkNew_WorstCase(b *testing.B) {
	for i := 0; i < b.N; i++ {
		P = New(worstCaseConstructors...)
	}
}

func BenchmarkNew_BestCase(b *testing.B) {
	for i := 0; i < b.N; i++ {
		P = New(bestCaseConstructors...)
	}
}

func BenchmarkClone_WorstCase(b *testing.B) {
	P = New(worstCaseConstructors...)
	for i := 0; i < b.N; i++ {
		P = P.Clone()
	}
}

func BenchmarkClone_BestCase(b *testing.B) {
	P = New(bestCaseConstructors...)
	for i := 0; i < b.N; i++ {
		P = P.Clone()
	}
}

// BenchmarkMustInject is testing the same psyringe each time. We do a single
// injection before the loop starts to initialise all values. This probably
// isn't a very likely scenario in real life.
func BenchmarkMustInject_WorstCase(b *testing.B) {
	P = New(worstCaseConstructors...)
	S = BenchStruct{}
	for i := 0; i < b.N; i++ {
		P.MustInject(&S)
	}
}

func BenchmarkMustInject_BestCase(b *testing.B) {
	P = New(bestCaseConstructors...)
	S = BenchStruct{}
	for i := 0; i < b.N; i++ {
		P.MustInject(&S)
	}
}

// BenchmarkNewMustInject is testing the complete cycle of creating a new
// psyringe and injecting with it. This benchmark exists primarily to compare
// New with Clone.
func BenchmarkNewMustInject_WorstCase(b *testing.B) {
	S = BenchStruct{}
	for i := 0; i < b.N; i++ {
		P = New(worstCaseConstructors...)
		P.MustInject(&S)
	}
}

func BenchmarkNewMustInject_BestCase(b *testing.B) {
	S = BenchStruct{}
	for i := 0; i < b.N; i++ {
		P = New(bestCaseConstructors...)
		P.MustInject(&S)
	}
}

// BenchmarkCloneMustInject is testing the cycle of cloning a new psyringe from
// an existing one and injecting with it. This is a likely scenario in a web
// server, for example, where certain resources must be created on each request.
func BenchmarkCloneMustInject_WorstCase(b *testing.B) {
	P = New(worstCaseConstructors...)
	S = BenchStruct{}
	for i := 0; i < b.N; i++ {
		P.Clone().MustInject(&S)
	}
}

func BenchmarkCloneMustInject_BestCase(b *testing.B) {
	P = New(bestCaseConstructors...)
	S = BenchStruct{}
	for i := 0; i < b.N; i++ {
		P.Clone().MustInject(&S)
	}
}
