package experiment

import (
	"fmt"
	"reflect"

	"github.com/samsalisbury/psyringe"
)

// OptionalFieldHandler validates that all injectable fields in a struct value
// must be filled
// unless they are marked optional by a field tag `inject:"optional"`.
func OptionalFieldHandler() psyringe.NoValueForStructFieldFunc {
	return func(parentType string, field reflect.StructField) error {
		if field.Tag.Get("inject") == "optional" {
			return nil
		}
		return fmt.Errorf("no constructor or value of type %s available (for field %s.%s)",
			field.Type, parentType, field.Name)
	}
}
