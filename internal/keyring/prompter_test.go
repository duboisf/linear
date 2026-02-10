package keyring_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/duboisf/linear/internal/keyring"
)

func TestInteractivePrompter_PromptForAPIKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		readPassword func(fd int) ([]byte, error)
		wantKey      string
		wantError    bool
		errContains  string
	}{
		{
			name: "success",
			readPassword: func(fd int) ([]byte, error) {
				return []byte("lin_api_prompt123"), nil
			},
			wantKey:   "lin_api_prompt123",
			wantError: false,
		},
		{
			name: "success with whitespace trimmed",
			readPassword: func(fd int) ([]byte, error) {
				return []byte("  lin_api_trimmed  "), nil
			},
			wantKey:   "lin_api_trimmed",
			wantError: false,
		},
		{
			name: "empty key error",
			readPassword: func(fd int) ([]byte, error) {
				return []byte(""), nil
			},
			wantKey:     "",
			wantError:   true,
			errContains: "empty",
		},
		{
			name: "whitespace only key error",
			readPassword: func(fd int) ([]byte, error) {
				return []byte("   "), nil
			},
			wantKey:     "",
			wantError:   true,
			errContains: "empty",
		},
		{
			name: "ReadPassword error",
			readPassword: func(fd int) ([]byte, error) {
				return nil, errors.New("terminal error")
			},
			wantKey:     "",
			wantError:   true,
			errContains: "reading API key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			prompter := &keyring.InteractivePrompter{
				ReadPassword: tt.readPassword,
			}

			var stdout bytes.Buffer
			key, err := prompter.PromptForAPIKey(nil, &stdout)
			if (err != nil) != tt.wantError {
				t.Errorf("PromptForAPIKey() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if key != tt.wantKey {
				t.Errorf("PromptForAPIKey() = %q, want %q", key, tt.wantKey)
			}
			if tt.wantError && tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
				t.Errorf("error %q does not contain %q", err.Error(), tt.errContains)
			}

			// Verify prompt output was written
			output := stdout.String()
			if !strings.Contains(output, "No Linear API key found") {
				t.Error("expected prompt instructions in stdout")
			}
			if !strings.Contains(output, "linear.app/settings/api") {
				t.Error("expected API key URL in stdout")
			}
		})
	}
}
