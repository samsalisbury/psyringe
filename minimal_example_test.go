package syringe_test

import (
	"bytes"
	"fmt"
	"io"

	"github.com/samsalisbury/syringe"
)

type DependentType struct{ Writer io.Writer }

func (dt *DependentType) WriteString(s string) { dt.Writer.Write([]byte(s)) }

func ExampleMinimal() {
	buf := &bytes.Buffer{} // eagerly create an object
	newWriter := func() io.Writer { return buf }
	syringe.Fill(newWriter)
	obj := DependentType{}
	syringe.Inject(&obj)
	obj.WriteString("hello?")
	fmt.Println(buf)
	// output:
	// hello?
}
