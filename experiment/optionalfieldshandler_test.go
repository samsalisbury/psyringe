package experiment

import (
	"strings"
	"testing"

	"github.com/samsalisbury/psyringe"
)

func TestOptionalFieldsHandler(t *testing.T) {

	type TestStruct struct {
		Int    int `inject:"optional"`
		String string
	}

	p := psyringe.New()
	p.Hooks.NoValueForStructField = OptionalFieldHandler()

	target := &TestStruct{}

	actualErr := p.Inject(target)
	expectedErrSuffix := "unable to inject field *experiment.TestStruct.String (string)"
	if actualErr == nil {
		t.Fatalf("got nil; want error ending %q", expectedErrSuffix)
	}
	actual := actualErr.Error()
	if !strings.HasSuffix(actual, expectedErrSuffix) {
		t.Fatalf("got error %q; want suffix %q", actualErr, expectedErrSuffix)
	}

}
