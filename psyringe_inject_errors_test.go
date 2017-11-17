package psyringe

import (
	"fmt"
	"testing"
)

type TestFormatter struct{}

func (tf *TestFormatter) Format(f fmt.State, c rune) {
	flagTest := int('#')
	fmt.Fprintf(f, "Format called with verb %c; flag %c %t",
		c, flagTest, f.Flag(flagTest))
}

func TestInjectErrors(t *testing.T) {
	p := New(
		func() (int, error) { return 0, fmt.Errorf("deep error") },
		func(int) string { return "a" },
		func(string) byte { return 1 },
	)

	x := struct {
		Byte byte
	}{}

	err := p.Inject(&x)

	t.Logf("Whole error: %#s", err)
	//t.Logf("Cause: %s", errors.Cause(err))

	//tf := &TestFormatter{}
	//t.Logf("%#s", tf)
}
