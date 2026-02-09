package keyring_test

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"

	"github.com/duboisf/linear/internal/keyring"
)

// --- Helper process for subprocess-based exec.Command mocking ---

func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	fmt.Fprint(os.Stdout, os.Getenv("GO_HELPER_STDOUT"))
	code, _ := strconv.Atoi(os.Getenv("GO_HELPER_EXIT_CODE"))
	os.Exit(code)
}

func fakeCommandRunner(stdout string, exitCode int) func(string, ...string) *exec.Cmd {
	return func(name string, args ...string) *exec.Cmd {
		cs := []string{"-test.run=TestHelperProcess", "--", name}
		cs = append(cs, args...)
		cmd := exec.Command(os.Args[0], cs...)
		cmd.Env = append(os.Environ(),
			"GO_WANT_HELPER_PROCESS=1",
			"GO_HELPER_STDOUT="+stdout,
			fmt.Sprintf("GO_HELPER_EXIT_CODE=%d", exitCode),
		)
		return cmd
	}
}

// --- Mock provider for testing ChainProvider and Resolve ---

type mockProvider struct {
	getKey   string
	getErr   error
	storeErr error
	stored   string
}

func (m *mockProvider) GetAPIKey() (string, error) {
	return m.getKey, m.getErr
}

func (m *mockProvider) StoreAPIKey(key string) error {
	m.stored = key
	return m.storeErr
}

// --- Mock prompter ---

type mockPrompter struct {
	key string
	err error
}

func (m *mockPrompter) PromptForAPIKey(_ io.Reader, _ io.Writer) (string, error) {
	return m.key, m.err
}

// --- EnvProvider Tests ---

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

// --- SecretToolProvider Tests ---

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

// --- ChainProvider Tests ---

func TestChainProvider_GetAPIKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		providers []keyring.Provider
		wantKey   string
		wantError bool
	}{
		{
			name: "first provider succeeds",
			providers: []keyring.Provider{
				&mockProvider{getKey: "key-from-first"},
				&mockProvider{getKey: "key-from-second"},
			},
			wantKey:   "key-from-first",
			wantError: false,
		},
		{
			name: "first fails second succeeds",
			providers: []keyring.Provider{
				&mockProvider{getErr: keyring.ErrNoAPIKey},
				&mockProvider{getKey: "key-from-second"},
			},
			wantKey:   "key-from-second",
			wantError: false,
		},
		{
			name: "all fail",
			providers: []keyring.Provider{
				&mockProvider{getErr: keyring.ErrNoAPIKey},
				&mockProvider{getErr: errors.New("some error")},
			},
			wantKey:   "",
			wantError: true,
		},
		{
			name:      "empty chain",
			providers: []keyring.Provider{},
			wantKey:   "",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			chain := &keyring.ChainProvider{Providers: tt.providers}
			key, err := chain.GetAPIKey()
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

func TestChainProvider_StoreAPIKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		providers []keyring.Provider
		wantError bool
	}{
		{
			name: "first provider succeeds",
			providers: []keyring.Provider{
				&mockProvider{storeErr: nil},
				&mockProvider{storeErr: nil},
			},
			wantError: false,
		},
		{
			name: "first fails second succeeds",
			providers: []keyring.Provider{
				&mockProvider{storeErr: errors.New("cannot store")},
				&mockProvider{storeErr: nil},
			},
			wantError: false,
		},
		{
			name: "all fail",
			providers: []keyring.Provider{
				&mockProvider{storeErr: errors.New("err1")},
				&mockProvider{storeErr: errors.New("err2")},
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			chain := &keyring.ChainProvider{Providers: tt.providers}
			err := chain.StoreAPIKey("test-key")
			if (err != nil) != tt.wantError {
				t.Errorf("StoreAPIKey() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

// --- InteractivePrompter Tests ---

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

// --- EnvProvider nil LookupEnv (default os.LookupEnv) ---

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

// --- SecretToolProvider nil CommandRunner (default exec.Command) ---

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

// --- Resolve Tests ---

func TestResolve(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		provider      keyring.Provider
		prompter      keyring.Prompter
		storeProvider keyring.Provider
		wantKey       string
		wantError     bool
		wantWarning   string
	}{
		{
			name:     "provider succeeds",
			provider: &mockProvider{getKey: "key-from-provider"},
			prompter: &mockPrompter{key: "should-not-be-used"},
			wantKey:  "key-from-provider",
		},
		{
			name:          "provider fails prompt succeeds store succeeds",
			provider:      &mockProvider{getErr: keyring.ErrNoAPIKey},
			prompter:      &mockPrompter{key: "prompted-key"},
			storeProvider: &mockProvider{storeErr: nil},
			wantKey:       "prompted-key",
		},
		{
			name:          "provider fails prompt succeeds store fails warning",
			provider:      &mockProvider{getErr: keyring.ErrNoAPIKey},
			prompter:      &mockPrompter{key: "prompted-key"},
			storeProvider: &mockProvider{storeErr: errors.New("store failed")},
			wantKey:       "prompted-key",
			wantWarning:   "Warning",
		},
		{
			name:      "provider fails prompt fails",
			provider:  &mockProvider{getErr: keyring.ErrNoAPIKey},
			prompter:  &mockPrompter{err: errors.New("prompt error")},
			wantError: true,
		},
		{
			name:          "provider fails prompt succeeds store is nil",
			provider:      &mockProvider{getErr: keyring.ErrNoAPIKey},
			prompter:      &mockPrompter{key: "prompted-key"},
			storeProvider: nil,
			wantKey:       "prompted-key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var stdout bytes.Buffer

			key, err := keyring.Resolve(tt.provider, tt.prompter, tt.storeProvider, nil, &stdout)
			if (err != nil) != tt.wantError {
				t.Errorf("Resolve() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if key != tt.wantKey {
				t.Errorf("Resolve() = %q, want %q", key, tt.wantKey)
			}
			if tt.wantWarning != "" && !strings.Contains(stdout.String(), tt.wantWarning) {
				t.Errorf("stdout %q does not contain warning %q", stdout.String(), tt.wantWarning)
			}
		})
	}
}
