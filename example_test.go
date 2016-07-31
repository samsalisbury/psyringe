package psyringe_test

import (
	"fmt"

	"github.com/samsalisbury/psyringe"
)

type Speaker struct {
	Message    string
	MessageLen int
}

// Contrived example showing how to create a Psyringe with interdependent
// constructors and then inject their values into a struct that depends on them.
func Example() {
	p := psyringe.New(
		func() string { return "Hi!" },       // string constructor
		func(s string) int { return len(s) }, // int constructor (needs string)
	)
	v := Speaker{}
	if err := p.Inject(&v); err != nil {
		panic(err) // a little drastic I'm sure
	}
	fmt.Printf("Speaker says %q in %d characters.", v.Message, v.MessageLen)
	// output:
	// Speaker says "Hi!" in 3 characters.
}
