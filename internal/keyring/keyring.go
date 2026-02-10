package keyring

import (
	"errors"
	"io"
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
