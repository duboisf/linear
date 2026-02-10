package keyring

import (
	"errors"
	"fmt"
	"io"
	"runtime"
	"strings"
)

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
