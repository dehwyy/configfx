package vault

// LoadOption configures vault.Load and vault.Validate behavior.
type LoadOption interface {
	applyLoad(*loadConfig)
}

type loadConfig struct {
	kvVersion int // 1 or 2
}

func newLoadConfig(opts []LoadOption) loadConfig {
	cfg := loadConfig{kvVersion: 1}
	for _, o := range opts {
		o.applyLoad(&cfg)
	}
	return cfg
}

type kvVersionOpt int

func (o kvVersionOpt) applyLoad(c *loadConfig) { c.kvVersion = int(o) }

// OptionClientKv1 uses Vault KV v1 API (default).
var OptionClientKv1 LoadOption = kvVersionOpt(1)

// OptionClientKv2 uses Vault KV v2 API.
var OptionClientKv2 LoadOption = kvVersionOpt(2)
