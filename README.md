# configfx

Typed config and secrets loader for Go services using [Uber FX](https://github.com/uber-go/fx).

Replaces manual `os.Getenv` calls and raw Vault HTTP requests with struct-tag-driven, FX-native config/secrets injection.

---

## Install

```bash
go get github.com/dehwyy/configfx
```

Vault subpackage:

```bash
go get github.com/dehwyy/configfx/vault
```

---

## Usage

### 1. Define your structs

```go
// internal/config/config.go
package config

type Config struct {
    AppEnv       string   `env:"APP_ENV,default=local"`
    AuthURL      string   `env:"AUTH_URL,required"`
    OtelEndpoint string   `env:"OTEL_ENDPOINT,default=localhost:4317"`
    CORSOrigins  []string `env:"CORS_ORIGINS,default=https://example.com"`
}

func (c *Config) IsLocal() bool { return c.AppEnv == "local" }

// internal/config/secrets.go
type Secrets struct {
    PgConnFmt   string `vault:"kv.shared.pg.conn.dev"`
    NatsServers string `vault:"kv.shared.nats.dev.servers"`
    NatsSeedKey string `vault:"kv.shared.nats.seedKey"`
}
```

### 2. Register as FX modules

```go
func main() {
    fx.New(
        configfx.FxModule[config.Config](),
        cfgvault.FxModule[config.Secrets](
            os.Getenv("KEY_VAULT_ADDRESS"),
            os.Getenv("KEY_VAULT_TOKEN"),
        ),
        // both *config.Config and *config.Secrets are now in the DI container
        fx.Provide(func(cfg *config.Config, sec *config.Secrets) (*nats.Conn, error) {
            // ...
        }),
    ).Run()
}
```

### 3. Inject as dependencies

```go
type Opts struct {
    fx.In
    Config  *config.Config
    Secrets *config.Secrets
}

func New(opts Opts) *Service { ... }
```

---

## Struct tag reference

### `env` tag (Config)

```
env:"KEY"                   // read APP_KEY, zero value if not set
env:"KEY,default=VALUE"     // use VALUE if not set
env:"KEY,required"          // error if not set and no default
```

Supported field types: `string`, `int`, `bool`, `[]string` (comma-separated).

### `vault` tag (Secrets)

```
vault:"mount.path.field"
```

- `mount` — KV v1 mount name (e.g. `kv`)
- `path` — secret path within mount (e.g. `shared`)
- `field` — field name inside the secret map (e.g. `pg.conn.dev`)

Tag `vault:"kv.shared.pg.conn.dev"` → reads `GET /v1/kv/shared`, takes `data["pg.conn.dev"]`.

Batch reads: all fields from the same `(mount, path)` share one HTTP request.

---

## Low-level API

If you need to load outside FX (e.g. early init, tests):

```go
cfg, err := configfx.Load[config.Config]()

sec, err := vault.Load[config.Secrets](vaultAddr, vaultToken)
```

---

## Pre-flight check CLI

The `check` subpackage provides a diagnostic binary that validates both env vars and Vault keys before service start.

```go
// cmd/check/main.go
package main

import (
    "os"
    "git.example.com/myservice/internal/config"
    "github.com/dehwyy/configfx/check"
)

func main() {
    check.Run[config.Config, config.Secrets](
        os.Getenv("KEY_VAULT_ADDRESS"),
        os.Getenv("KEY_VAULT_TOKEN"),
    )
}
```

Output example:

```
Config validation (env vars):
  ✓ APP_ENV              = "local"
  ✓ AUTH_URL             = "https://auth.example.com/api/v1"
  ✗ REQUIRED_KEY         missing required env var

Secrets validation (Vault kv://https://vault.example.com):
  ✓ kv.shared.pg.conn.dev                     → PgConnFmt
  ✗ kv.shared.nats.dev.servers                failed to read kv/shared: ...

1 error(s) found. Fix before starting the service.
```

Exits `0` on success, `1` on any error.

---

## Project structure

```
configfx/
├── loader.go          # Load[T]() — env vars → struct
├── validate.go        # Validate[T]() — dry-run, no side effects
├── fx.go              # FxModule[T]() — wraps Load in fx.Provide
├── internal/
│   ├── env/           # tag parser + type coercion
│   └── field/         # reflect-based field setter
├── vault/
│   ├── loader.go      # Load[T](addr, token) — Vault KV v1 → struct
│   ├── validate.go    # Validate[T](addr, token) — dry-run
│   ├── fx.go          # FxModule[T](addr, token) — wraps Load in fx.Provide
│   └── internal/      # vault tag parser
└── check/
    └── check.go       # Run[C, S](addr, token) — CLI validator
```

---

## Requirements

- Go 1.25.5+
- Vault KV v1 (not KV v2)
