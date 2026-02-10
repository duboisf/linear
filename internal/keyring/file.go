package keyring

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FileSystem abstracts filesystem operations needed by FileProvider.
type FileSystem interface {
	ReadFile(name string) ([]byte, error)
	WriteFile(name string, data []byte, perm os.FileMode) error
	MkdirAll(path string, perm os.FileMode) error
}

// osFileSystem implements FileSystem using the real filesystem.
type osFileSystem struct{}

var _ FileSystem = osFileSystem{}

func (osFileSystem) ReadFile(name string) ([]byte, error)                       { return os.ReadFile(name) }
func (osFileSystem) WriteFile(name string, data []byte, perm os.FileMode) error { return os.WriteFile(name, data, perm) }
func (osFileSystem) MkdirAll(path string, perm os.FileMode) error              { return os.MkdirAll(path, perm) }

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
