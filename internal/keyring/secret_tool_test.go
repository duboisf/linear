package keyring_test

import (
	"testing"

	"github.com/duboisf/linear/internal/keyring"
)

func TestSecretToolProvider_GetAPIKey(t *testing.T) {
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
			stdout:    "lin_api_secret123",
			exitCode:  0,
			wantKey:   "lin_api_secret123",
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
			name:      "whitespace only output",
			stdout:    "   \n  ",
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

			provider := &keyring.SecretToolProvider{
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

func TestSecretToolProvider_StoreAPIKey(t *testing.T) {
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

			provider := &keyring.SecretToolProvider{
				CommandRunner: fakeCommandRunner("", tt.exitCode),
			}

			err := provider.StoreAPIKey("test-key")
			if (err != nil) != tt.wantError {
				t.Errorf("StoreAPIKey() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestSecretToolProvider_NilCommandRunner_DefaultsToExecCommand(t *testing.T) {
	t.Parallel()

	// With nil CommandRunner, it should fall back to exec.Command.
	// secret-tool is likely not installed in test environments, so we
	// expect an error, but the important thing is it doesn't panic.
	provider := &keyring.SecretToolProvider{CommandRunner: nil}
	_, err := provider.GetAPIKey()
	// We just verify it doesn't panic and returns some error (since
	// secret-tool is not installed in test environment).
	if err == nil {
		// This would only succeed if secret-tool is installed AND has a key.
		// That's fine too - we just care that the nil fallback path was exercised.
		t.Log("secret-tool is available and returned a key")
	}
}
