package env

import "strings"

// Tag represents a parsed env struct tag.
type Tag struct {
	Key        string
	Default    string
	HasDefault bool
	Required   bool
}

// ParseTag parses an env struct tag string.
// Supported formats:
//   - "KEY"
//   - "KEY,default=VALUE"
//   - "KEY,required"
//   - "KEY,default=VALUE,required"
func ParseTag(tag string) Tag {
	parts := strings.Split(tag, ",")
	t := Tag{
		Key: strings.TrimSpace(parts[0]),
	}

	for _, part := range parts[1:] {
		part = strings.TrimSpace(part)
		if part == "required" {
			t.Required = true
		} else if strings.HasPrefix(part, "default=") {
			t.Default = strings.TrimPrefix(part, "default=")
			t.HasDefault = true
		}
	}

	return t
}
