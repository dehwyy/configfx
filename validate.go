package configfx

import (
	"os"
	"reflect"

	"github.com/dehwyy/configfx/internal/env"
)

// ValidationError describes a single missing required env var.
type ValidationError struct {
	Field   string
	EnvKey  string
	Message string
}

// Validate checks all required env vars without loading values into a struct.
// Returns a list of errors for fields that are required, missing, and have no default.
func Validate[T any]() []ValidationError {
	var zero T
	t := reflect.TypeOf(zero)

	if t.Kind() != reflect.Struct {
		return []ValidationError{{
			Field:   "",
			EnvKey:  "",
			Message: "Validate expects a struct type",
		}}
	}

	var errs []ValidationError

	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		tagStr, ok := f.Tag.Lookup("env")
		if !ok {
			continue
		}

		tag := env.ParseTag(tagStr)
		if tag.Key == "" {
			continue
		}

		if !tag.Required {
			continue
		}

		val := os.Getenv(tag.Key)
		if val == "" && !tag.HasDefault {
			errs = append(errs, ValidationError{
				Field:   f.Name,
				EnvKey:  tag.Key,
				Message: "required env var is not set and has no default",
			})
		}
	}

	return errs
}
