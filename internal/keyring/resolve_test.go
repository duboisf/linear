package keyring_test

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/duboisf/linear/internal/keyring"
)

func TestResolve(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		provider    keyring.Provider
		prompter    keyring.Prompter
		nativeStore keyring.Provider
		fileStore   keyring.Provider
		readLine    func() (string, error)
		wantKey     string
		wantError   bool
		wantOutput  string
	}{
		{
			name:     "provider succeeds",
			provider: &mockProvider{getKey: "key-from-provider"},
			prompter: &mockPrompter{key: "should-not-be-used"},
			wantKey:  "key-from-provider",
		},
		{
			name:        "native store succeeds",
			provider:    &mockProvider{getErr: keyring.ErrNoAPIKey},
			prompter:    &mockPrompter{key: "prompted-key"},
			nativeStore: &mockProvider{},
			wantKey:     "prompted-key",
		},
		{
			name:        "native store fails falls back to file with confirmation",
			provider:    &mockProvider{getErr: keyring.ErrNoAPIKey},
			prompter:    &mockPrompter{key: "prompted-key"},
			nativeStore: &mockProvider{storeErr: errors.New("store failed")},
			fileStore:   &mockProvider{},
			readLine:    func() (string, error) { return "y", nil },
			wantKey:     "prompted-key",
			wantOutput:  "Warning",
		},
		{
			name:        "native tool not found shows install hint",
			provider:    &mockProvider{getErr: keyring.ErrNoAPIKey},
			prompter:    &mockPrompter{key: "prompted-key"},
			nativeStore: &mockProvider{storeErr: fmt.Errorf("%w: secret-tool", keyring.ErrToolNotFound)},
			fileStore:   &mockProvider{},
			readLine:    func() (string, error) { return "y", nil },
			wantKey:     "prompted-key",
			wantOutput:  "Install",
		},
		{
			name:        "file fallback declined",
			provider:    &mockProvider{getErr: keyring.ErrNoAPIKey},
			prompter:    &mockPrompter{key: "prompted-key"},
			nativeStore: &mockProvider{storeErr: errors.New("store failed")},
			fileStore:   &mockProvider{},
			readLine:    func() (string, error) { return "n", nil },
			wantKey:     "prompted-key",
			wantOutput:  "not saved",
		},
		{
			name:      "prompt fails",
			provider:  &mockProvider{getErr: keyring.ErrNoAPIKey},
			prompter:  &mockPrompter{err: errors.New("prompt error")},
			wantError: true,
		},
		{
			name:     "both stores nil",
			provider: &mockProvider{getErr: keyring.ErrNoAPIKey},
			prompter: &mockPrompter{key: "prompted-key"},
			wantKey:  "prompted-key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var msgBuf bytes.Buffer

			key, err := keyring.Resolve(keyring.ResolveOptions{
				Provider:    tt.provider,
				Prompter:    tt.prompter,
				NativeStore: tt.nativeStore,
				FileStore:   tt.fileStore,
				Stdin:       strings.NewReader(""),
				MsgWriter:   &msgBuf,
				ReadLine:    tt.readLine,
			})
			if (err != nil) != tt.wantError {
				t.Errorf("Resolve() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if key != tt.wantKey {
				t.Errorf("Resolve() = %q, want %q", key, tt.wantKey)
			}
			if tt.wantOutput != "" && !strings.Contains(msgBuf.String(), tt.wantOutput) {
				t.Errorf("output %q does not contain %q", msgBuf.String(), tt.wantOutput)
			}
		})
	}
}
