package psyringe_test

import (
	"fmt"

	"github.com/samsalisbury/psyringe"
)

type SomeStruct struct {
	Message    string
	MessageLen int
}

func Example() {
	p := psyringe.MustNew(
		func() string { return "Hi!" },
		func(s string) int { return len(s) },
	)
	v := SomeStruct{}
	if err := p.Inject(&v); err != nil {
		panic(err)
	}
	fmt.Printf("SomeStruct says %q in %d characters.", v.Message, v.MessageLen)
	// output:
	// SomeStruct says "Hi!" in 3 characters.
}
