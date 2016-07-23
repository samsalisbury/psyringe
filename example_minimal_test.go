package psyringe_test

import (
	"bytes"
	"fmt"
	"io"

	"github.com/samsalisbury/psyringe"
)

type DependentType struct{ Writer io.Writer }

func (dt *DependentType) WriteString(s string) { dt.Writer.Write([]byte(s)) }

func ExampleMinimal() {
	buf := &bytes.Buffer{} // eagerly create an object
	newWriter := func() io.Writer { return buf }
	psyringe.Fill(newWriter)
	obj := DependentType{}
	psyringe.Inject(&obj)
	obj.WriteString("hello?")
	fmt.Println(buf)
	// output:
	// hello?
}
