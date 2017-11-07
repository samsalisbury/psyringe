package experiment

import (
	"fmt"
	"reflect"

	"github.com/samsalisbury/psyringe"
)

// OptionalFieldHandler  validates that all fields in an injected struct must be
// filled unless they are marked optional by a field tag `inject:"optional"`.
func OptionalFieldHandler() psyringe.NoValueForStructFieldFunc {
	return func(parentType string, field reflect.StructField) error {
		if field.Tag.Get("inject") == "optional" {
			return nil
		}
		return fmt.Errorf("unable to inject field %s.%s (%s)",
			parentType, field.Name, field.Type)
	}
}
