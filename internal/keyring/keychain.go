package keyring

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// KeychainProvider uses macOS Keychain (security CLI) for API key storage.
type KeychainProvider struct {
	// CommandRunner allows overriding exec.Command for testing.
	CommandRunner func(name string, args ...string) *exec.Cmd
}

func (p *KeychainProvider) commandRunner() func(string, ...string) *exec.Cmd {
	if p.CommandRunner != nil {
		return p.CommandRunner
	}
	return exec.Command
}

// GetAPIKey retrieves the API key from the macOS Keychain via the security CLI.
func (p *KeychainProvider) GetAPIKey() (string, error) {
	cmd := p.commandRunner()("security", "find-generic-password", "-s", "linear", "-a", "default", "-w")
	out, err := cmd.Output()
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return "", fmt.Errorf("%w: security", ErrToolNotFound)
		}
		return "", fmt.Errorf("security find-generic-password failed: %w", err)
	}
	key := strings.TrimSpace(string(out))
	if key == "" {
		return "", ErrNoAPIKey
	}
	return key, nil
}

// StoreAPIKey stores the API key in the macOS Keychain via the security CLI.
// The -U flag updates the entry if it already exists.
func (p *KeychainProvider) StoreAPIKey(key string) error {
	cmd := p.commandRunner()("security", "add-generic-password", "-s", "linear", "-a", "default", "-w", key, "-U")
	if err := cmd.Run(); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return fmt.Errorf("%w: security", ErrToolNotFound)
		}
		return fmt.Errorf("security add-generic-password failed: %w", err)
	}
	return nil
}
