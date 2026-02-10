package keyring

import "errors"

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
