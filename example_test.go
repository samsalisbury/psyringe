package psyringe_test

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/samsalisbury/psyringe"
)

type SomeStruct struct {
	Message string
}

func newString(r io.Reader) (string, error) {
	b, err := ioutil.ReadAll(r)
	return string(b), err
}
func newReader() io.Reader {
	return bytes.NewBufferString("Hi!")
}

func Example() {
	p := psyringe.MustNew(newString, newReader)
	v := SomeStruct{}
	if err := p.Inject(&v); err != nil {
		panic(err)
	}
	fmt.Printf("SomeStruct says %q", v.Message)
}
