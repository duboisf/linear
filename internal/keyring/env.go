package keyring

import (
	"errors"
	"os"
)

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
