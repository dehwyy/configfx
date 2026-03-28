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

// Load reads Vault KV secrets into T using vault struct tags.
// addr: Vault address (e.g. "https://vault.dev.uniteplat.org")
// token: Vault token
// opts: optional LoadOption values (e.g. OptionClientKv2 to use KV v2 API)
//
// Strategy: batch reads — collect unique (mount, path) pairs,
// make one HTTP request per unique path, map fields from response Data map.
func Load[T any](addr, token string, opts ...LoadOption) (*T, error) {
	cfg := newLoadConfig(opts)

	clientOpts := []vaultclient.ClientOption{vaultclient.WithAddress(addr)}
	if cfg.tlsSkipVerify {
		clientOpts = append(
			clientOpts,
			vaultclient.WithTLS(vaultclient.TLSConfiguration{InsecureSkipVerify: true}),
		)
	}

	client, err := vaultclient.New(clientOpts...)
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
		data, err := readPath(ctx, client, key, cfg.kvVersion)
		if err != nil {
			return nil, fmt.Errorf("vault: failed to read %s/%s: %w", key.mount, key.path, err)
		}
		cache[key] = data
	}

	for key, refs := range pathFields {
		data := cache[key]
		for _, ref := range refs {
			fieldName := t.Field(ref.index).Name
			raw, ok := data[ref.tag.Field]
			if !ok {
				return nil, fmt.Errorf("vault: field %s: key %q not found in %s/%s", fieldName, ref.tag.Field, key.mount, key.path)
			}

			fv := v.Field(ref.index)
			if !fv.CanSet() {
				return nil, fmt.Errorf("vault: field %s is not settable", fieldName)
			}

			if err := setField(fv, raw, fieldName); err != nil {
				return nil, err
			}
		}
	}

	result := v.Interface().(T)
	return &result, nil
}

// readPath performs a KV read and returns the data map.
func readPath(ctx context.Context, client *vaultclient.Client, key pathKey, kvVersion int) (map[string]any, error) {
	switch kvVersion {
	case 2:
		resp, err := client.Secrets.KvV2Read(ctx, key.path, vaultclient.WithMountPath(key.mount))
		if err != nil {
			return nil, err
		}
		return resp.Data.Data, nil
	default: // 1
		resp, err := client.Secrets.KvV1Read(ctx, key.path, vaultclient.WithMountPath(key.mount))
		if err != nil {
			return nil, err
		}
		return resp.Data, nil
	}
}

// setField maps a raw interface{} value from Vault into a reflect.Value.
// Supports string, []string, []int (and other int slice variants).
func setField(fv reflect.Value, raw any, fieldName string) error {
	switch fv.Kind() {
	case reflect.String:
		s, ok := raw.(string)
		if !ok {
			return fmt.Errorf("vault: field %s: expected string, got %T", fieldName, raw)
		}
		fv.SetString(s)

	case reflect.Slice:
		arr, ok := raw.([]interface{})
		if !ok {
			return fmt.Errorf("vault: field %s: expected array, got %T", fieldName, raw)
		}
		elemKind := fv.Type().Elem().Kind()
		result := reflect.MakeSlice(fv.Type(), len(arr), len(arr))
		for i, item := range arr {
			elem := result.Index(i)
			switch elemKind {
			case reflect.String:
				s, ok := item.(string)
				if !ok {
					return fmt.Errorf("vault: field %s[%d]: expected string element, got %T", fieldName, i, item)
				}
				elem.SetString(s)
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				n, ok := item.(float64) // JSON numbers decode as float64
				if !ok {
					return fmt.Errorf("vault: field %s[%d]: expected number element, got %T", fieldName, i, item)
				}
				elem.SetInt(int64(n))
			default:
				return fmt.Errorf("vault: field %s: unsupported slice element kind %s", fieldName, elemKind)
			}
		}
		fv.Set(result)

	default:
		return fmt.Errorf("vault: field %s: unsupported field kind %s (only string and []string/[]int are supported)", fieldName, fv.Kind())
	}
	return nil
}
