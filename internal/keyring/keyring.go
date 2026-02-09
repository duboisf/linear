package keyring

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"golang.org/x/term"
)

// Provider abstracts API key storage and retrieval.
type Provider interface {
	// GetAPIKey returns the stored API key, or an error if none is found.
	GetAPIKey() (string, error)
	// StoreAPIKey persists the given API key.
	StoreAPIKey(key string) error
}

// Prompter handles interactive user prompts for API key setup.
type Prompter interface {
	// PromptForAPIKey prompts the user to enter an API key and returns it.
	PromptForAPIKey(stdin io.Reader, stdout io.Writer) (string, error)
}

// ErrNoAPIKey is returned when no API key is found.
var ErrNoAPIKey = errors.New("no API key found")

// EnvProvider resolves the API key from the LINEAR_API_KEY environment variable.
type EnvProvider struct {
	// LookupEnv allows overriding os.LookupEnv for testing.
	LookupEnv func(key string) (string, bool)
}

// GetAPIKey returns the API key from the LINEAR_API_KEY environment variable.
func (p *EnvProvider) GetAPIKey() (string, error) {
	lookup := p.LookupEnv
	if lookup == nil {
		lookup = os.LookupEnv
	}
	val, ok := lookup("LINEAR_API_KEY")
	if !ok || val == "" {
		return "", ErrNoAPIKey
	}
	return val, nil
}

// StoreAPIKey is not supported for environment variables.
func (p *EnvProvider) StoreAPIKey(_ string) error {
	return errors.New("cannot store API key in environment variable")
}

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
		return fmt.Errorf("secret-tool store failed: %w", err)
	}
	return nil
}

// ChainProvider tries multiple providers in order, returning the first success.
type ChainProvider struct {
	Providers []Provider
}

// GetAPIKey tries each provider in order and returns the first successful result.
func (p *ChainProvider) GetAPIKey() (string, error) {
	for _, provider := range p.Providers {
		key, err := provider.GetAPIKey()
		if err == nil {
			return key, nil
		}
	}
	return "", ErrNoAPIKey
}

// StoreAPIKey stores the key using the first provider that supports it.
func (p *ChainProvider) StoreAPIKey(key string) error {
	for _, provider := range p.Providers {
		if err := provider.StoreAPIKey(key); err == nil {
			return nil
		}
	}
	return errors.New("no provider could store the API key")
}

// InteractivePrompter prompts the user for an API key via the terminal.
type InteractivePrompter struct {
	// ReadPassword allows overriding term.ReadPassword for testing.
	ReadPassword func(fd int) ([]byte, error)
}

// PromptForAPIKey displays instructions and reads the API key without echo.
// The stdin parameter is unused because term.ReadPassword requires a file
// descriptor (os.Stdin.Fd()) to disable terminal echo; an io.Reader is
// insufficient for that operation.
func (p *InteractivePrompter) PromptForAPIKey(_ io.Reader, msgWriter io.Writer) (string, error) {
	fmt.Fprintln(msgWriter, "No Linear API key found.")
	fmt.Fprintln(msgWriter, "Create one at: https://linear.app/settings/api")
	fmt.Fprint(msgWriter, "Enter your Linear API key: ")

	readPassword := p.ReadPassword
	if readPassword == nil {
		readPassword = term.ReadPassword
	}

	keyBytes, err := readPassword(int(os.Stdin.Fd()))
	if err != nil {
		return "", fmt.Errorf("reading API key: %w", err)
	}
	fmt.Fprintln(msgWriter) // newline after hidden input

	key := strings.TrimSpace(string(keyBytes))
	if key == "" {
		return "", errors.New("API key cannot be empty")
	}
	return key, nil
}

// Resolve returns an API key using the following precedence:
// 1. The given provider chain (env var, then keyring)
// 2. Interactive prompt (stores result via storeProvider for next time)
//
// The msgWriter parameter receives user-facing messages (prompts, warnings).
// Callers typically pass stderr so that prompts do not interfere with stdout.
func Resolve(provider Provider, prompter Prompter, storeProvider Provider, stdin io.Reader, msgWriter io.Writer) (string, error) {
	key, err := provider.GetAPIKey()
	if err == nil {
		return key, nil
	}

	key, err = prompter.PromptForAPIKey(stdin, msgWriter)
	if err != nil {
		return "", fmt.Errorf("prompting for API key: %w", err)
	}

	if storeProvider != nil {
		if storeErr := storeProvider.StoreAPIKey(key); storeErr != nil {
			fmt.Fprintf(msgWriter, "Warning: could not store API key: %v\n", storeErr)
		}
	}

	return key, nil
}
