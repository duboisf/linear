package cmd_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/duboisf/linear/cmd"
	"github.com/duboisf/linear/internal/cache"
)

func TestCacheClear_Empty(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	c := cache.New(dir, 5*time.Minute)
	var out bytes.Buffer
	root := cmd.NewRootCmd(cmd.Options{
		Cache:  c,
		Stdout: &out,
		Stderr: &bytes.Buffer{},
	})
	root.SetArgs([]string{"cache", "clear"})

	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := out.String(); got != "Cache is already empty.\n" {
		t.Errorf("unexpected output: %q", got)
	}
}

func TestCacheClear_WithEntries(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	c := cache.New(dir, 5*time.Minute)
	_ = c.Set("issues/ENG-1", "data1")
	_ = c.Set("issues/ENG-2", "data2")

	var out bytes.Buffer
	root := cmd.NewRootCmd(cmd.Options{
		Cache:  c,
		Stdout: &out,
		Stderr: &bytes.Buffer{},
	})
	root.SetArgs([]string{"cache", "clear"})

	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := out.String(); got != "Cleared 2 cached file(s).\n" {
		t.Errorf("unexpected output: %q", got)
	}

	// Verify cache is actually empty.
	if _, ok := c.Get("issues/ENG-1"); ok {
		t.Error("ENG-1 should no longer be cached")
	}
	if _, ok := c.Get("issues/ENG-2"); ok {
		t.Error("ENG-2 should no longer be cached")
	}
}
