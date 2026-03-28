package configfx

import (
	"fmt"
	"os"
	"reflect"

	"github.com/dehwyy/configfx/internal/env"
	"github.com/dehwyy/configfx/internal/field"
)

// Load reads env vars into T using struct tags.
// Supported tags: env:"KEY", env:"KEY,default=VALUE", env:"KEY,required"
// Fields without env tag are skipped.
// Returns error if a required field is missing and has no default.
func Load[T any]() (*T, error) {
	var zero T
	t := reflect.TypeOf(zero)
	v := reflect.New(t).Elem()

	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("configfx: Load expects a struct type, got %s", t.Kind())
	}

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

		raw := os.Getenv(tag.Key)

		if raw == "" {
			if tag.HasDefault {
				raw = tag.Default
			} else if tag.Required {
				return nil, fmt.Errorf("configfx: required env var %q is not set (field %s)", tag.Key, f.Name)
			} else {
				// optional, no default — leave zero value
				continue
			}
		}

		coerced, err := env.Coerce(raw, f.Type)
		if err != nil {
			return nil, fmt.Errorf("configfx: field %s (env %q): %w", f.Name, tag.Key, err)
		}

		if err := field.Set(v.Field(i), coerced); err != nil {
			return nil, fmt.Errorf("configfx: field %s: %w", f.Name, err)
		}
	}

	result := v.Interface().(T)
	return &result, nil
}
