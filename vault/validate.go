package vault

import (
	"context"
	"fmt"
	"reflect"

	vaultclient "github.com/hashicorp/vault-client-go"
	vaultinternal "github.com/dehwyy/configfx/vault/internal"
)

// ValidationError describes a single validation failure for a vault secret.
type ValidationError struct {
	Field    string
	VaultKey string
	Message  string
}

// Validate checks Vault connectivity and existence of all keys defined in T.
// Returns a list of errors: connection failure OR missing keys.
func Validate[T any](addr, token string, opts ...LoadOption) []ValidationError {
	cfg := newLoadConfig(opts)

	client, err := vaultclient.New(vaultclient.WithAddress(addr))
	if err != nil {
		return []ValidationError{{
			Message: fmt.Sprintf("failed to create vault client: %v", err),
		}}
	}

	if err := client.SetToken(token); err != nil {
		return []ValidationError{{
			Message: fmt.Sprintf("failed to set vault token: %v", err),
		}}
	}

	var zero T
	t := reflect.TypeOf(zero)

	if t.Kind() != reflect.Struct {
		return []ValidationError{{
			Message: "Validate expects a struct type",
		}}
	}

	type fieldRef struct {
		name string
		tag  vaultinternal.VaultTag
	}
	pathFields := make(map[pathKey][]fieldRef)

	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		tagStr, ok := f.Tag.Lookup("vault")
		if !ok {
			continue
		}

		tag, err := vaultinternal.ParseTag(tagStr)
		if err != nil {
			return []ValidationError{{
				Field:    f.Name,
				VaultKey: tagStr,
				Message:  fmt.Sprintf("invalid vault tag: %v", err),
			}}
		}

		key := pathKey{mount: tag.Mount, path: tag.Path}
		pathFields[key] = append(pathFields[key], fieldRef{name: f.Name, tag: tag})
	}

	ctx := context.Background()
	var errs []ValidationError

	for key, refs := range pathFields {
		data, err := readPath(ctx, client, key, cfg.kvVersion)
		if err != nil {
			for _, ref := range refs {
				errs = append(errs, ValidationError{
					Field:    ref.name,
					VaultKey: fmt.Sprintf("%s.%s.%s", ref.tag.Mount, ref.tag.Path, ref.tag.Field),
					Message:  fmt.Sprintf("failed to read %s/%s: %v", key.mount, key.path, err),
				})
			}
			continue
		}

		for _, ref := range refs {
			if _, ok := data[ref.tag.Field]; !ok {
				errs = append(errs, ValidationError{
					Field:    ref.name,
					VaultKey: fmt.Sprintf("%s.%s.%s", ref.tag.Mount, ref.tag.Path, ref.tag.Field),
					Message:  fmt.Sprintf("key %q not found in %s/%s", ref.tag.Field, key.mount, key.path),
				})
			}
		}
	}

	return errs
}
