package keyring

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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

// ErrToolNotFound is returned when the native credential storage tool is not installed.
var ErrToolNotFound = errors.New("credential storage tool not found")

// nativeToolInstallHint returns a user-facing message explaining how to install
// the native credential storage tool for the current platform.
func nativeToolInstallHint() string {
	switch runtime.GOOS {
	case "darwin":
		return "The macOS security CLI should be available by default.\n" +
			"If missing, install Xcode Command Line Tools:\n" +
			"  xcode-select --install"
	default:
		return "Install secret-tool for secure credential storage:\n" +
			"  Ubuntu/Debian: sudo apt install libsecret-tools\n" +
			"  Fedora:        sudo dnf install libsecret\n" +
			"  Arch:          sudo pacman -S libsecret"
	}
}

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

// FileSystem abstracts filesystem operations needed by FileProvider.
type FileSystem interface {
	ReadFile(name string) ([]byte, error)
	WriteFile(name string, data []byte, perm os.FileMode) error
	MkdirAll(path string, perm os.FileMode) error
}

// osFileSystem implements FileSystem using the real filesystem.
type osFileSystem struct{}

var _ FileSystem = osFileSystem{}

func (osFileSystem) ReadFile(name string) ([]byte, error)                        { return os.ReadFile(name) }
func (osFileSystem) WriteFile(name string, data []byte, perm os.FileMode) error  { return os.WriteFile(name, data, perm) }
func (osFileSystem) MkdirAll(path string, perm os.FileMode) error                { return os.MkdirAll(path, perm) }

// FileProvider stores the API key in a file under the user's config directory.
// The file is created with 0600 permissions (owner read/write only).
type FileProvider struct {
	// FS provides filesystem operations. Defaults to the real OS filesystem.
	FS FileSystem
	// ConfigDir returns the user's config directory. Defaults to os.UserConfigDir.
	ConfigDir func() (string, error)
}

func (p *FileProvider) fs() FileSystem {
	if p.FS != nil {
		return p.FS
	}
	return osFileSystem{}
}

func (p *FileProvider) credentialPath() (string, error) {
	configDir := p.ConfigDir
	if configDir == nil {
		configDir = os.UserConfigDir
	}
	dir, err := configDir()
	if err != nil {
		return "", fmt.Errorf("determining config directory: %w", err)
	}
	return filepath.Join(dir, "linear", "credentials"), nil
}

// GetAPIKey reads the API key from the credentials file.
func (p *FileProvider) GetAPIKey() (string, error) {
	path, err := p.credentialPath()
	if err != nil {
		return "", err
	}
	data, err := p.fs().ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", ErrNoAPIKey
		}
		return "", fmt.Errorf("reading credentials file: %w", err)
	}
	key := strings.TrimSpace(string(data))
	if key == "" {
		return "", ErrNoAPIKey
	}
	return key, nil
}

// StoreAPIKey writes the API key to the credentials file with 0600 permissions.
func (p *FileProvider) StoreAPIKey(key string) error {
	path, err := p.credentialPath()
	if err != nil {
		return err
	}
	if err := p.fs().MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}
	if err := p.fs().WriteFile(path, []byte(key+"\n"), 0600); err != nil {
		return fmt.Errorf("writing credentials file: %w", err)
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

// ResolveOptions configures the Resolve function.
type ResolveOptions struct {
	// Provider resolves API keys (typically a ChainProvider).
	Provider Provider
	// Prompter handles interactive API key prompts.
	Prompter Prompter
	// NativeStore is the platform-specific credential store (keyring/keychain).
	NativeStore Provider
	// FileStore is the file-based fallback credential store.
	FileStore Provider
	// Stdin for interactive input.
	Stdin io.Reader
	// MsgWriter receives user-facing messages (prompts, warnings).
	// Callers typically pass stderr so that prompts do not interfere with stdout.
	MsgWriter io.Writer
	// ReadLine reads a line of user input for confirmation prompts.
	// If nil, defaults to reading from Stdin.
	ReadLine func() (string, error)
}

func (o *ResolveOptions) readLine() (string, error) {
	if o.ReadLine != nil {
		return o.ReadLine()
	}
	var buf [256]byte
	n, err := o.Stdin.Read(buf[:])
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(buf[:n])), nil
}

// Resolve returns an API key using the following precedence:
// 1. The given provider chain (env var, then keyring, then file)
// 2. Interactive prompt (stores result for next time)
//
// When storing, Resolve tries the native keyring first. If the native tool
// is not installed, it shows install instructions and asks the user before
// falling back to file-based storage.
func Resolve(opts ResolveOptions) (string, error) {
	key, err := opts.Provider.GetAPIKey()
	if err == nil {
		return key, nil
	}

	key, err = opts.Prompter.PromptForAPIKey(opts.Stdin, opts.MsgWriter)
	if err != nil {
		return "", fmt.Errorf("prompting for API key: %w", err)
	}

	storeKey(key, opts)

	return key, nil
}

func storeKey(key string, opts ResolveOptions) {
	// Try native keyring first.
	if opts.NativeStore != nil {
		if err := opts.NativeStore.StoreAPIKey(key); err == nil {
			return
		} else if errors.Is(err, ErrToolNotFound) {
			fmt.Fprintf(opts.MsgWriter, "\n%s\n\n", nativeToolInstallHint())
		} else {
			fmt.Fprintf(opts.MsgWriter, "Warning: could not store API key in system keyring: %v\n", err)
		}
	}

	// Fall back to file storage with user confirmation.
	if opts.FileStore != nil {
		fmt.Fprint(opts.MsgWriter, "Store API key in a local config file instead? [y/N]: ")
		answer, err := opts.readLine()
		if err != nil {
			fmt.Fprintf(opts.MsgWriter, "Warning: could not read response: %v\n", err)
			return
		}
		if answer != "y" && answer != "Y" && answer != "yes" {
			fmt.Fprintln(opts.MsgWriter, "API key was not saved. You will be prompted again next time.")
			return
		}
		if err := opts.FileStore.StoreAPIKey(key); err != nil {
			fmt.Fprintf(opts.MsgWriter, "Warning: could not store API key in file: %v\n", err)
		}
	}
}
