package cache

import (
	"os"
	"path/filepath"
	"time"
)

// Cache is a simple file-based cache with TTL expiry based on file mtime.
type Cache struct {
	Dir string
	TTL time.Duration
}

// New creates a Cache rooted at dir with the given TTL.
func New(dir string, ttl time.Duration) *Cache {
	return &Cache{Dir: dir, TTL: ttl}
}

// Get reads the cached value for key. It returns the content and true if the
// file exists and its mtime is within the TTL window. Otherwise it returns
// empty string and false.
func (c *Cache) Get(key string) (string, bool) {
	return c.GetWithTTL(key, c.TTL)
}

// GetWithTTL is like Get but uses the provided TTL instead of the cache default.
// This allows callers to use longer or shorter TTLs for specific keys.
func (c *Cache) GetWithTTL(key string, ttl time.Duration) (string, bool) {
	path := filepath.Join(c.Dir, key)
	info, err := os.Stat(path)
	if err != nil {
		return "", false
	}
	if time.Since(info.ModTime()) > ttl {
		return "", false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	return string(data), true
}

// Clear removes all cached files by deleting and re-creating the cache
// directory. It returns the number of files removed or an error.
func (c *Cache) Clear() (int, error) {
	entries, err := os.ReadDir(c.Dir)
	if os.IsNotExist(err) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	count := countFiles(entries, c.Dir)
	if err := os.RemoveAll(c.Dir); err != nil {
		return 0, err
	}
	return count, os.MkdirAll(c.Dir, 0o700)
}

// countFiles recursively counts regular files under dir.
func countFiles(entries []os.DirEntry, dir string) int {
	n := 0
	for _, e := range entries {
		if e.IsDir() {
			sub := filepath.Join(dir, e.Name())
			if children, err := os.ReadDir(sub); err == nil {
				n += countFiles(children, sub)
			}
			continue
		}
		n++
	}
	return n
}

// Set atomically writes content to the cache file for key, creating parent
// directories as needed. It writes to a temp file first then renames, so
// concurrent readers never see a partial write.
func (c *Cache) Set(key, content string) error {
	path := filepath.Join(c.Dir, key)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return err
	}
	if _, err := tmp.WriteString(content); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	return os.Rename(tmp.Name(), path)
}
