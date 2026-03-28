package vault

import (
	"context"
	"fmt"
	"reflect"

	vaultclient "github.com/hashicorp/vault-client-go"
	vaultinternal "github.com/dehwyy/configfx/vault/internal"
)

type pathKey struct {
	mount string
	path  string
}

// Load reads Vault KV v1 secrets into T using vault struct tags.
// addr: Vault address (e.g. "https://vault.dev.uniteplat.org")
// token: Vault token
//
// Strategy: batch reads — collect unique (mount, path) pairs,
// make one HTTP request per unique path, map fields from response Data map.
func Load[T any](addr, token string) (*T, error) {
	client, err := vaultclient.New(vaultclient.WithAddress(addr))
	if err != nil {
		return nil, fmt.Errorf("vault: failed to create client: %w", err)
	}

	if err := client.SetToken(token); err != nil {
		return nil, fmt.Errorf("vault: failed to set token: %w", err)
	}

	var zero T
	t := reflect.TypeOf(zero)
	v := reflect.New(t).Elem()

	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("vault: Load expects a struct type, got %s", t.Kind())
	}

	type fieldRef struct {
		index int
		tag   vaultinternal.VaultTag
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
			return nil, fmt.Errorf("vault: field %s: %w", f.Name, err)
		}

		key := pathKey{mount: tag.Mount, path: tag.Path}
		pathFields[key] = append(pathFields[key], fieldRef{index: i, tag: tag})
	}

	ctx := context.Background()
	cache := make(map[pathKey]map[string]any)

	for key := range pathFields {
		resp, err := client.Secrets.KvV1Read(
			ctx,
			key.path,
			vaultclient.WithMountPath(key.mount),
		)
		if err != nil {
			return nil, fmt.Errorf("vault: failed to read %s/%s: %w", key.mount, key.path, err)
		}
		cache[key] = resp.Data
	}

	for key, refs := range pathFields {
		data := cache[key]
		for _, ref := range refs {
			fieldName := t.Field(ref.index).Name
			raw, ok := data[ref.tag.Field]
			if !ok {
				return nil, fmt.Errorf("vault: field %s: key %q not found in %s/%s", fieldName, ref.tag.Field, key.mount, key.path)
			}

			str, ok := raw.(string)
			if !ok {
				return nil, fmt.Errorf("vault: field %s: expected string value, got %T", fieldName, raw)
			}

			fv := v.Field(ref.index)
			if !fv.CanSet() {
				return nil, fmt.Errorf("vault: field %s is not settable", fieldName)
			}

			if fv.Kind() != reflect.String {
				return nil, fmt.Errorf("vault: field %s must be string type, got %s", fieldName, fv.Kind())
			}

			fv.SetString(str)
		}
	}

	result := v.Interface().(T)
	return &result, nil
}
