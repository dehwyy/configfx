package env

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// Coerce converts a string value to the target reflect.Type.
// Supported types: string, int, int32, int64, bool, []string, time.Duration.
func Coerce(val string, t reflect.Type) (interface{}, error) {
	// Handle time.Duration separately (it's a named type based on int64)
	if t == reflect.TypeOf(time.Duration(0)) {
		d, err := time.ParseDuration(val)
		if err != nil {
			return nil, fmt.Errorf("cannot parse %q as time.Duration: %w", val, err)
		}
		return d, nil
	}

	// Handle slice of strings
	if t.Kind() == reflect.Slice && t.Elem().Kind() == reflect.String {
		parts := strings.Split(val, ",")
		result := make([]string, 0, len(parts))
		for _, p := range parts {
			trimmed := strings.TrimSpace(p)
			if trimmed != "" {
				result = append(result, trimmed)
			}
		}
		return result, nil
	}

	switch t.Kind() {
	case reflect.String:
		return val, nil

	case reflect.Int:
		n, err := strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("cannot parse %q as int: %w", val, err)
		}
		return n, nil

	case reflect.Int32:
		n, err := strconv.ParseInt(val, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("cannot parse %q as int32: %w", val, err)
		}
		return int32(n), nil

	case reflect.Int64:
		n, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("cannot parse %q as int64: %w", val, err)
		}
		return n, nil

	case reflect.Bool:
		switch strings.ToLower(val) {
		case "true", "1":
			return true, nil
		case "false", "0":
			return false, nil
		default:
			return nil, fmt.Errorf("cannot parse %q as bool: expected true/false/1/0", val)
		}

	default:
		return nil, fmt.Errorf("unsupported type: %s", t.Kind())
	}
}
