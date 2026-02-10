package keyring_test

import (
	"errors"
	"testing"

	"github.com/duboisf/linear/internal/keyring"
)

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
