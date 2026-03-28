package configfx_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/dehwyy/configfx"
	"github.com/dehwyy/configfx/internal/env"
)

// --- Test structs ---

type basicConfig struct {
	AppEnv  string `env:"APP_ENV"`
	Port    int    `env:"PORT"`
	Debug   bool   `env:"DEBUG"`
	Timeout int64  `env:"TIMEOUT"`
}

type requiredConfig struct {
	DBUrl string `env:"DB_URL,required"`
	Host  string `env:"HOST,required,default=localhost"`
}

type defaultConfig struct {
	LogLevel string `env:"LOG_LEVEL,default=info"`
	Workers  int    `env:"WORKERS,default=4"`
}

type sliceConfig struct {
	AllowedIPs []string `env:"ALLOWED_IPS"`
}

type durationConfig struct {
	Timeout time.Duration `env:"REQUEST_TIMEOUT"`
}

type mixedConfig struct {
	AppName    string        `env:"APP_NAME,required"`
	Port       int           `env:"PORT,default=8080"`
	Debug      bool          `env:"DEBUG,default=false"`
	Tags       []string      `env:"TAGS"`
	MaxTimeout time.Duration `env:"MAX_TIMEOUT,default=30s"`
	NoTag      string        // no env tag — should be skipped
}

// --- Load tests ---

func TestLoad_BasicTypes(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("PORT", "9090")
	t.Setenv("DEBUG", "true")
	t.Setenv("TIMEOUT", "12345")

	cfg, err := configfx.Load[basicConfig]()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.AppEnv != "production" {
		t.Errorf("AppEnv: got %q, want %q", cfg.AppEnv, "production")
	}
	if cfg.Port != 9090 {
		t.Errorf("Port: got %d, want 9090", cfg.Port)
	}
	if !cfg.Debug {
		t.Errorf("Debug: got false, want true")
	}
	if cfg.Timeout != 12345 {
		t.Errorf("Timeout: got %d, want 12345", cfg.Timeout)
	}
}

func TestLoad_RequiredMissing(t *testing.T) {
	t.Setenv("DB_URL", "")
	t.Setenv("HOST", "")

	_, err := configfx.Load[requiredConfig]()
	if err == nil {
		t.Fatal("expected error for missing required field DB_URL, got nil")
	}
}

func TestLoad_RequiredWithDefault(t *testing.T) {
	t.Setenv("DB_URL", "postgres://localhost/mydb")
	t.Setenv("HOST", "")

	cfg, err := configfx.Load[requiredConfig]()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Host != "localhost" {
		t.Errorf("Host: got %q, want %q", cfg.Host, "localhost")
	}
}

func TestLoad_Defaults(t *testing.T) {
	t.Setenv("LOG_LEVEL", "")
	t.Setenv("WORKERS", "")

	cfg, err := configfx.Load[defaultConfig]()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel: got %q, want %q", cfg.LogLevel, "info")
	}
	if cfg.Workers != 4 {
		t.Errorf("Workers: got %d, want 4", cfg.Workers)
	}
}

func TestLoad_DefaultsOverriddenByEnv(t *testing.T) {
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("WORKERS", "16")

	cfg, err := configfx.Load[defaultConfig]()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("LogLevel: got %q, want %q", cfg.LogLevel, "debug")
	}
	if cfg.Workers != 16 {
		t.Errorf("Workers: got %d, want 16", cfg.Workers)
	}
}

func TestLoad_SliceOfStrings(t *testing.T) {
	t.Setenv("ALLOWED_IPS", "10.0.0.1, 10.0.0.2, 192.168.1.1")

	cfg, err := configfx.Load[sliceConfig]()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.AllowedIPs) != 3 {
		t.Fatalf("AllowedIPs: got len %d, want 3", len(cfg.AllowedIPs))
	}
	if cfg.AllowedIPs[0] != "10.0.0.1" {
		t.Errorf("AllowedIPs[0]: got %q, want %q", cfg.AllowedIPs[0], "10.0.0.1")
	}
	if cfg.AllowedIPs[2] != "192.168.1.1" {
		t.Errorf("AllowedIPs[2]: got %q, want %q", cfg.AllowedIPs[2], "192.168.1.1")
	}
}

func TestLoad_Duration(t *testing.T) {
	t.Setenv("REQUEST_TIMEOUT", "5m30s")

	cfg, err := configfx.Load[durationConfig]()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := 5*time.Minute + 30*time.Second
	if cfg.Timeout != expected {
		t.Errorf("Timeout: got %v, want %v", cfg.Timeout, expected)
	}
}

func TestLoad_Mixed(t *testing.T) {
	t.Setenv("APP_NAME", "myservice")
	t.Setenv("PORT", "")
	t.Setenv("DEBUG", "1")
	t.Setenv("TAGS", "web, api, v2")
	t.Setenv("MAX_TIMEOUT", "")

	cfg, err := configfx.Load[mixedConfig]()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.AppName != "myservice" {
		t.Errorf("AppName: got %q, want %q", cfg.AppName, "myservice")
	}
	if cfg.Port != 8080 {
		t.Errorf("Port: got %d, want 8080 (default)", cfg.Port)
	}
	if !cfg.Debug {
		t.Errorf("Debug: got false, want true")
	}
	if len(cfg.Tags) != 3 || cfg.Tags[1] != "api" {
		t.Errorf("Tags: got %v", cfg.Tags)
	}
	if cfg.MaxTimeout != 30*time.Second {
		t.Errorf("MaxTimeout: got %v, want 30s (default)", cfg.MaxTimeout)
	}
	if cfg.NoTag != "" {
		t.Errorf("NoTag should be zero value (skipped), got %q", cfg.NoTag)
	}
}

// --- Validate tests ---

func TestValidate_NoErrors(t *testing.T) {
	t.Setenv("DB_URL", "postgres://localhost/mydb")

	errs := configfx.Validate[requiredConfig]()
	if len(errs) != 0 {
		t.Errorf("expected no errors, got %v", errs)
	}
}

func TestValidate_MissingRequired(t *testing.T) {
	t.Setenv("DB_URL", "")

	errs := configfx.Validate[requiredConfig]()
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d: %v", len(errs), errs)
	}
	if errs[0].EnvKey != "DB_URL" {
		t.Errorf("expected error for DB_URL, got %q", errs[0].EnvKey)
	}
}

func TestValidate_OptionalFieldsIgnored(t *testing.T) {
	errs := configfx.Validate[defaultConfig]()
	if len(errs) != 0 {
		t.Errorf("expected no errors for optional-only config, got %v", errs)
	}
}

// --- Coerce tests ---

func TestCoerce_String(t *testing.T) {
	result, err := env.Coerce("hello", reflect.TypeOf(""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.(string) != "hello" {
		t.Errorf("got %q, want %q", result, "hello")
	}
}

func TestCoerce_Int(t *testing.T) {
	result, err := env.Coerce("42", reflect.TypeOf(0))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.(int) != 42 {
		t.Errorf("got %v, want 42", result)
	}
}

func TestCoerce_Int64(t *testing.T) {
	result, err := env.Coerce("9999999999", reflect.TypeOf(int64(0)))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.(int64) != 9999999999 {
		t.Errorf("got %v, want 9999999999", result)
	}
}

func TestCoerce_BoolTrue(t *testing.T) {
	for _, input := range []string{"true", "1"} {
		result, err := env.Coerce(input, reflect.TypeOf(false))
		if err != nil {
			t.Fatalf("input %q: unexpected error: %v", input, err)
		}
		if result.(bool) != true {
			t.Errorf("input %q: got false, want true", input)
		}
	}
}

func TestCoerce_BoolFalse(t *testing.T) {
	for _, input := range []string{"false", "0"} {
		result, err := env.Coerce(input, reflect.TypeOf(false))
		if err != nil {
			t.Fatalf("input %q: unexpected error: %v", input, err)
		}
		if result.(bool) != false {
			t.Errorf("input %q: got true, want false", input)
		}
	}
}

func TestCoerce_BoolInvalid(t *testing.T) {
	_, err := env.Coerce("yes", reflect.TypeOf(false))
	if err == nil {
		t.Fatal("expected error for invalid bool value, got nil")
	}
}

func TestCoerce_SliceOfStrings(t *testing.T) {
	result, err := env.Coerce("a, b , c", reflect.TypeOf([]string{}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	s := result.([]string)
	if len(s) != 3 || s[0] != "a" || s[1] != "b" || s[2] != "c" {
		t.Errorf("got %v", s)
	}
}

func TestCoerce_Duration(t *testing.T) {
	result, err := env.Coerce("2h30m", reflect.TypeOf(time.Duration(0)))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	d := result.(time.Duration)
	expected := 2*time.Hour + 30*time.Minute
	if d != expected {
		t.Errorf("got %v, want %v", d, expected)
	}
}

func TestCoerce_DurationInvalid(t *testing.T) {
	_, err := env.Coerce("notaduration", reflect.TypeOf(time.Duration(0)))
	if err == nil {
		t.Fatal("expected error for invalid duration, got nil")
	}
}
