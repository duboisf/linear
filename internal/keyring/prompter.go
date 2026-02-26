package keyring

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

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
	return p.ReadAPIKey(msgWriter)
}

// ReadAPIKey prompts "Enter your Linear API key: " and reads the key without
// terminal echo. Unlike PromptForAPIKey, it does not print any preamble.
func (p *InteractivePrompter) ReadAPIKey(msgWriter io.Writer) (string, error) {
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
