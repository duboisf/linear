package keyring_test

import (
	"testing"

	"github.com/duboisf/linear/internal/keyring"
)

func TestKeychainProvider_GetAPIKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		stdout    string
		exitCode  int
		wantKey   string
		wantError bool
	}{
		{
			name:      "success",
			stdout:    "lin_api_keychain123",
			exitCode:  0,
			wantKey:   "lin_api_keychain123",
			wantError: false,
		},
		{
			name:      "success with whitespace",
			stdout:    "  lin_api_trimmed  \n",
			exitCode:  0,
			wantKey:   "lin_api_trimmed",
			wantError: false,
		},
		{
			name:      "empty output",
			stdout:    "",
			exitCode:  0,
			wantKey:   "",
			wantError: true,
		},
		{
			name:      "command failure",
			stdout:    "",
			exitCode:  1,
			wantKey:   "",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			provider := &keyring.KeychainProvider{
				CommandRunner: fakeCommandRunner(tt.stdout, tt.exitCode),
			}

			key, err := provider.GetAPIKey()
			if (err != nil) != tt.wantError {
				t.Errorf("GetAPIKey() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if key != tt.wantKey {
				t.Errorf("GetAPIKey() = %q, want %q", key, tt.wantKey)
			}
		})
	}
}

func TestKeychainProvider_StoreAPIKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		exitCode  int
		wantError bool
	}{
		{
			name:      "success",
			exitCode:  0,
			wantError: false,
		},
		{
			name:      "command failure",
			exitCode:  1,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			provider := &keyring.KeychainProvider{
				CommandRunner: fakeCommandRunner("", tt.exitCode),
			}

			err := provider.StoreAPIKey("test-key")
			if (err != nil) != tt.wantError {
				t.Errorf("StoreAPIKey() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}
