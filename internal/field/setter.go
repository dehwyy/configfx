package field

import (
	"fmt"
	"reflect"
)

// Set sets a struct field value using reflection.
// val must already be the correct type (from coerce.Coerce).
func Set(field reflect.Value, val interface{}) error {
	if !field.CanSet() {
		return fmt.Errorf("field is not settable")
	}

	v := reflect.ValueOf(val)
	if !v.Type().AssignableTo(field.Type()) {
		return fmt.Errorf("cannot assign %s to %s", v.Type(), field.Type())
	}

	field.Set(v)
	return nil
}
