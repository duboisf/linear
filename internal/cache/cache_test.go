package cache_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/duboisf/linear/internal/cache"
)

func TestSetAndGet(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	c := cache.New(dir, 5*time.Minute)

	if err := c.Set("issues/ENG-1", "hello"); err != nil {
		t.Fatalf("Set: %v", err)
	}

	got, ok := c.Get("issues/ENG-1")
	if !ok {
		t.Fatal("expected cache hit")
	}
	if got != "hello" {
		t.Errorf("got %q, want %q", got, "hello")
	}
}

func TestGet_Miss(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	c := cache.New(dir, 5*time.Minute)

	_, ok := c.Get("nonexistent")
	if ok {
		t.Error("expected cache miss for nonexistent key")
	}
}

func TestGet_Expired(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	c := cache.New(dir, 1*time.Millisecond)

	if err := c.Set("key", "value"); err != nil {
		t.Fatalf("Set: %v", err)
	}

	// Wait for TTL to expire.
	time.Sleep(5 * time.Millisecond)

	_, ok := c.Get("key")
	if ok {
		t.Error("expected cache miss after TTL expiry")
	}
}

func TestSet_CreatesSubdirectories(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	c := cache.New(dir, 5*time.Minute)

	if err := c.Set("a/b/c", "deep"); err != nil {
		t.Fatalf("Set: %v", err)
	}

	path := filepath.Join(dir, "a", "b", "c")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "deep" {
		t.Errorf("got %q, want %q", string(data), "deep")
	}
}

func TestGetWithTTL_LongerThanDefault(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	c := cache.New(dir, 1*time.Millisecond)

	if err := c.Set("key", "value"); err != nil {
		t.Fatalf("Set: %v", err)
	}

	// Wait for the default TTL to expire.
	time.Sleep(5 * time.Millisecond)

	// Default Get should miss.
	if _, ok := c.Get("key"); ok {
		t.Error("expected cache miss with default TTL")
	}

	// GetWithTTL with a longer TTL should still hit.
	got, ok := c.GetWithTTL("key", 1*time.Hour)
	if !ok {
		t.Fatal("expected cache hit with longer TTL")
	}
	if got != "value" {
		t.Errorf("got %q, want %q", got, "value")
	}
}

func TestGetWithTTL_ShorterThanDefault(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	c := cache.New(dir, 1*time.Hour)

	if err := c.Set("key", "value"); err != nil {
		t.Fatalf("Set: %v", err)
	}

	// Wait for the short TTL to expire.
	time.Sleep(5 * time.Millisecond)

	// Default Get should hit (1h TTL).
	if _, ok := c.Get("key"); !ok {
		t.Error("expected cache hit with default TTL")
	}

	// GetWithTTL with a shorter TTL should miss.
	if _, ok := c.GetWithTTL("key", 1*time.Millisecond); ok {
		t.Error("expected cache miss with shorter TTL")
	}
}

func TestSet_Overwrites(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	c := cache.New(dir, 5*time.Minute)

	if err := c.Set("key", "old"); err != nil {
		t.Fatalf("Set old: %v", err)
	}
	if err := c.Set("key", "new"); err != nil {
		t.Fatalf("Set new: %v", err)
	}

	got, ok := c.Get("key")
	if !ok {
		t.Fatal("expected cache hit")
	}
	if got != "new" {
		t.Errorf("got %q, want %q", got, "new")
	}
}
