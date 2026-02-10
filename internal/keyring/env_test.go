package keyring_test

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/duboisf/linear/internal/keyring"
)

func TestEnvProvider_GetAPIKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		envValue  string
		envSet    bool
		wantKey   string
		wantError bool
	}{
		{
			name:      "env var set with value",
			envValue:  "lin_api_test123",
			envSet:    true,
			wantKey:   "lin_api_test123",
			wantError: false,
		},
		{
			name:      "env var set but empty",
			envValue:  "",
			envSet:    true,
			wantKey:   "",
			wantError: true,
		},
		{
			name:      "env var not set",
			envValue:  "",
			envSet:    false,
			wantKey:   "",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			provider := &keyring.EnvProvider{
				LookupEnv: func(key string) (string, bool) {
					if key == "LINEAR_API_KEY" {
						return tt.envValue, tt.envSet
					}
					return "", false
				},
			}

			key, err := provider.GetAPIKey()
			if (err != nil) != tt.wantError {
				t.Errorf("GetAPIKey() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if key != tt.wantKey {
				t.Errorf("GetAPIKey() = %q, want %q", key, tt.wantKey)
			}
			if tt.wantError && !errors.Is(err, keyring.ErrNoAPIKey) {
				t.Errorf("expected ErrNoAPIKey, got %v", err)
			}
		})
	}
}

func TestEnvProvider_StoreAPIKey(t *testing.T) {
	t.Parallel()

	provider := &keyring.EnvProvider{}
	err := provider.StoreAPIKey("some-key")
	if err == nil {
		t.Fatal("expected error from StoreAPIKey, got nil")
	}
	if !strings.Contains(err.Error(), "cannot store") {
		t.Errorf("expected error about inability to store, got: %v", err)
	}
}

func TestEnvProvider_NilLookupEnv_UsesOsLookupEnv(t *testing.T) {
	// When LookupEnv is nil, EnvProvider should fall back to os.LookupEnv.
	// We set the env var to exercise this path.
	t.Setenv("LINEAR_API_KEY", "env-test-key-from-os")

	provider := &keyring.EnvProvider{LookupEnv: nil}
	key, err := provider.GetAPIKey()
	if err != nil {
		t.Fatalf("GetAPIKey() error = %v", err)
	}
	if key != "env-test-key-from-os" {
		t.Errorf("GetAPIKey() = %q, want %q", key, "env-test-key-from-os")
	}
}

func TestEnvProvider_NilLookupEnv_NotSet(t *testing.T) {
	// Ensure LINEAR_API_KEY is not set so the nil-LookupEnv path returns ErrNoAPIKey.
	os.Unsetenv("LINEAR_API_KEY")

	provider := &keyring.EnvProvider{LookupEnv: nil}
	_, err := provider.GetAPIKey()
	if err == nil {
		t.Fatal("expected error when LINEAR_API_KEY is not set")
	}
	if !errors.Is(err, keyring.ErrNoAPIKey) {
		t.Errorf("expected ErrNoAPIKey, got %v", err)
	}
}
