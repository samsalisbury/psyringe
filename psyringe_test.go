package psyringe

import (
	"flag"
	"fmt"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	var debug bool
	flag.BoolVar(&debug, "debug", false, "turn on debug output")
	flag.Parse()
	if debug {
		debugf = func(format string, a ...interface{}) {
			fmt.Fprintf(os.Stderr, "DEBUG: "+format+"\n", a...)
		}
	}
	os.Exit(m.Run())
}
