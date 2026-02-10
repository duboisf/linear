package keyring_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/duboisf/linear/internal/keyring"
)

// --- memFS for FileProvider testing ---

type memFS struct {
	files    map[string][]byte
	readErr  error
	writeErr error
	mkdirErr error

	writtenPath string
	writtenData []byte
	writtenPerm os.FileMode
}

var _ keyring.FileSystem = (*memFS)(nil)

func (m *memFS) ReadFile(name string) ([]byte, error) {
	if m.readErr != nil {
		return nil, m.readErr
	}
	data, ok := m.files[name]
	if !ok {
		return nil, os.ErrNotExist
	}
	return data, nil
}

func (m *memFS) WriteFile(name string, data []byte, perm os.FileMode) error {
	if m.writeErr != nil {
		return m.writeErr
	}
	m.writtenPath = name
	m.writtenData = data
	m.writtenPerm = perm
	return nil
}

func (m *memFS) MkdirAll(path string, perm os.FileMode) error {
	return m.mkdirErr
}

// --- FileProvider Tests ---

func TestFileProvider_GetAPIKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		fs        *memFS
		wantKey   string
		wantError bool
	}{
		{
			name:      "success",
			fs:        &memFS{files: map[string][]byte{"/fake/config/linear/credentials": []byte("lin_api_file123\n")}},
			wantKey:   "lin_api_file123",
			wantError: false,
		},
		{
			name:      "success with whitespace",
			fs:        &memFS{files: map[string][]byte{"/fake/config/linear/credentials": []byte("  lin_api_trimmed  \n")}},
			wantKey:   "lin_api_trimmed",
			wantError: false,
		},
		{
			name:      "file not found",
			fs:        &memFS{files: map[string][]byte{}},
			wantKey:   "",
			wantError: true,
		},
		{
			name:      "empty file",
			fs:        &memFS{files: map[string][]byte{"/fake/config/linear/credentials": []byte("")}},
			wantKey:   "",
			wantError: true,
		},
		{
			name:      "read error",
			fs:        &memFS{readErr: errors.New("permission denied")},
			wantKey:   "",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			provider := &keyring.FileProvider{
				FS:        tt.fs,
				ConfigDir: func() (string, error) { return "/fake/config", nil },
			}

			key, err := provider.GetAPIKey()
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

func TestFileProvider_GetAPIKey_ConfigDirError(t *testing.T) {
	t.Parallel()

	provider := &keyring.FileProvider{
		FS:        &memFS{},
		ConfigDir: func() (string, error) { return "", errors.New("no home dir") },
	}
	_, err := provider.GetAPIKey()
	if err == nil {
		t.Fatal("expected error when config dir fails")
	}
}

func TestFileProvider_StoreAPIKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		fs        *memFS
		wantError bool
	}{
		{
			name:      "success",
			fs:        &memFS{},
			wantError: false,
		},
		{
			name:      "mkdir failure",
			fs:        &memFS{mkdirErr: errors.New("permission denied")},
			wantError: true,
		},
		{
			name:      "write failure",
			fs:        &memFS{writeErr: errors.New("disk full")},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			provider := &keyring.FileProvider{
				FS:        tt.fs,
				ConfigDir: func() (string, error) { return "/fake/config", nil },
			}

			err := provider.StoreAPIKey("test-key")
			if (err != nil) != tt.wantError {
				t.Errorf("StoreAPIKey() error = %v, wantError %v", err, tt.wantError)
			}
			if !tt.wantError {
				wantPath := filepath.Join("/fake/config", "linear", "credentials")
				if tt.fs.writtenPath != wantPath {
					t.Errorf("wrote to %q, want %q", tt.fs.writtenPath, wantPath)
				}
				if string(tt.fs.writtenData) != "test-key\n" {
					t.Errorf("wrote data %q, want %q", tt.fs.writtenData, "test-key\n")
				}
				if tt.fs.writtenPerm != 0600 {
					t.Errorf("wrote perm %o, want %o", tt.fs.writtenPerm, 0600)
				}
			}
		})
	}
}

func TestFileProvider_StoreAPIKey_ConfigDirError(t *testing.T) {
	t.Parallel()

	provider := &keyring.FileProvider{
		FS:        &memFS{},
		ConfigDir: func() (string, error) { return "", errors.New("no home dir") },
	}
	err := provider.StoreAPIKey("test-key")
	if err == nil {
		t.Fatal("expected error when config dir fails")
	}
}
