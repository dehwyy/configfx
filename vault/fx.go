package vault

import (
	"fmt"

	"go.uber.org/fx"
)

// FxModule returns an fx.Option that provides *T loaded from Vault.
// addr and token are evaluated at call time (not fx startup time).
func FxModule[T any](addr, token string, opts ...LoadOption) fx.Option {
	return fx.Provide(func() (*T, error) {
		cfg, err := Load[T](addr, token, opts...)
		if err != nil {
			return nil, fmt.Errorf("vault.FxModule: %w", err)
		}
		return cfg, nil
	})
}
