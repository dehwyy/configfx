package vaultinternal

import (
	"fmt"
	"strings"
)

// VaultTag represents a parsed vault struct tag.
// Tag format: "mount.path.field" where field can contain dots.
// Example: "kv.shared.pg.conn.dev" → Mount="kv", Path="shared", Field="pg.conn.dev"
type VaultTag struct {
	Mount string // e.g. "kv"
	Path  string // e.g. "shared"
	Field string // e.g. "pg.conn.dev"
}

// ParseTag parses a vault struct tag string.
// The tag must have at least 3 dot-separated parts: mount.path.field[.more...]
func ParseTag(tag string) (VaultTag, error) {
	parts := strings.SplitN(tag, ".", 3)
	if len(parts) < 3 {
		return VaultTag{}, fmt.Errorf("vault tag %q must have format 'mount.path.field' (minimum 3 dot-separated parts)", tag)
	}

	return VaultTag{
		Mount: parts[0],
		Path:  parts[1],
		Field: parts[2],
	}, nil
}
