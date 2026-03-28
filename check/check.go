package check

import (
	"fmt"
	"os"
	"reflect"

	"github.com/dehwyy/configfx"
	"github.com/dehwyy/configfx/internal/env"
	"github.com/dehwyy/configfx/vault"
)

// Run validates both env config (C) and vault secrets (S).
// Prints a formatted table with checkmark/cross per field.
// Exits with code 1 if any errors found, 0 if all OK.
func Run[C any, S any](vaultAddr, vaultToken string) {
	totalErrors := 0

	// --- Env config validation ---
	fmt.Println("Config validation (env vars):")
	totalErrors += printEnvValidation[C]()

	// --- Vault secrets validation ---
	fmt.Printf("\nSecrets validation (Vault kv://%s):\n", vaultAddr)
	totalErrors += printVaultValidation[S](vaultAddr, vaultToken)

	// --- Summary ---
	fmt.Println()
	if totalErrors == 0 {
		fmt.Println("All checks passed.")
		os.Exit(0)
	} else {
		fmt.Printf("%d error(s) found. Fix before starting the service.\n", totalErrors)
		os.Exit(1)
	}
}

// printEnvValidation prints env var check lines and returns the number of errors.
func printEnvValidation[C any]() int {
	var zero C
	t := reflect.TypeOf(zero)

	if t.Kind() != reflect.Struct {
		fmt.Println("  ! C is not a struct type")
		return 1
	}

	errCount := 0
	envErrors := configfx.Validate[C]()
	errMap := make(map[string]configfx.ValidationError, len(envErrors))
	for _, e := range envErrors {
		errMap[e.EnvKey] = e
		errCount++
	}

	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		tagStr, ok := f.Tag.Lookup("env")
		if !ok {
			continue
		}

		tag := env.ParseTag(tagStr)
		if tag.Key == "" {
			continue
		}

		if e, failed := errMap[tag.Key]; failed {
			fmt.Printf("  \u2717 %-20s %s\n", tag.Key, e.Message)
		} else {
			val := os.Getenv(tag.Key)
			if val == "" && tag.HasDefault {
				val = tag.Default
			}
			fmt.Printf("  \u2713 %-20s = %q\n", tag.Key, val)
		}
	}

	return errCount
}

// printVaultValidation prints vault check lines and returns the number of errors.
func printVaultValidation[S any](addr, token string) int {
	var zero S
	t := reflect.TypeOf(zero)

	if t.Kind() != reflect.Struct {
		fmt.Println("  ! S is not a struct type")
		return 1
	}

	vaultErrors := vault.Validate[S](addr, token)
	errMap := make(map[string]vault.ValidationError, len(vaultErrors))
	for _, e := range vaultErrors {
		errMap[e.Field] = e
	}

	errCount := len(vaultErrors)

	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		tagStr, ok := f.Tag.Lookup("vault")
		if !ok {
			continue
		}

		if e, failed := errMap[f.Name]; failed {
			fmt.Printf("  \u2717 %-40s %s\n", tagStr, e.Message)
			_ = e
		} else {
			fmt.Printf("  \u2713 %-40s \u2192 %s\n", tagStr, f.Name)
		}
	}

	return errCount
}
