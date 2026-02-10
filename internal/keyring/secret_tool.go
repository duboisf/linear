package keyring

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// SecretToolProvider uses secret-tool (GNOME keyring) for API key storage.
type SecretToolProvider struct {
	// CommandRunner allows overriding exec.Command for testing.
	CommandRunner func(name string, args ...string) *exec.Cmd
}

func (p *SecretToolProvider) commandRunner() func(string, ...string) *exec.Cmd {
	if p.CommandRunner != nil {
		return p.CommandRunner
	}
	return exec.Command
}

// GetAPIKey retrieves the API key from the GNOME keyring via secret-tool.
func (p *SecretToolProvider) GetAPIKey() (string, error) {
	cmd := p.commandRunner()("secret-tool", "lookup", "service", "linear", "account", "default")
	out, err := cmd.Output()
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return "", fmt.Errorf("%w: secret-tool", ErrToolNotFound)
		}
		return "", fmt.Errorf("secret-tool lookup failed: %w", err)
	}
	key := strings.TrimSpace(string(out))
	if key == "" {
		return "", ErrNoAPIKey
	}
	return key, nil
}

// StoreAPIKey stores the API key in the GNOME keyring via secret-tool.
// The key is passed via stdin to avoid exposing it in process arguments.
func (p *SecretToolProvider) StoreAPIKey(key string) error {
	cmd := p.commandRunner()(
		"secret-tool", "store",
		"--label=Linear API key (linear-cli)",
		"service", "linear",
		"account", "default",
	)
	cmd.Stdin = bytes.NewReader([]byte(key))
	if err := cmd.Run(); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return fmt.Errorf("%w: secret-tool", ErrToolNotFound)
		}
		return fmt.Errorf("secret-tool store failed: %w", err)
	}
	return nil
}
