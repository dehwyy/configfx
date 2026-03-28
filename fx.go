package configfx

import (
	"fmt"

	"go.uber.org/fx"
)

// FxModule returns an fx.Option that provides *T loaded from env vars.
// Panics on load error (fail-fast at startup).
func FxModule[T any]() fx.Option {
	return fx.Provide(func() (*T, error) {
		cfg, err := Load[T]()
		if err != nil {
			return nil, fmt.Errorf("configfx.FxModule: %w", err)
		}
		return cfg, nil
	})
}
