package config

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureExampleFile_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	err := EnsureExampleFile(func() (string, error) { return dir, nil })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(dir, "linear", "config.example.yaml"))
	if err != nil {
		t.Fatalf("reading example file: %v", err)
	}
	if string(got) != string(ExampleContent) {
		t.Errorf("content mismatch:\ngot:  %q\nwant: %q", got, ExampleContent)
	}
}

func TestEnsureExampleFile_OverwritesStaleContent(t *testing.T) {
	dir := t.TempDir()
	linearDir := filepath.Join(dir, "linear")
	if err := os.MkdirAll(linearDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(linearDir, "config.example.yaml"), []byte("old content"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := EnsureExampleFile(func() (string, error) { return dir, nil })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(linearDir, "config.example.yaml"))
	if err != nil {
		t.Fatalf("reading example file: %v", err)
	}
	if string(got) != string(ExampleContent) {
		t.Errorf("stale content was not replaced")
	}
}

func TestEnsureExampleFile_SkipsWhenCurrent(t *testing.T) {
	dir := t.TempDir()
	linearDir := filepath.Join(dir, "linear")
	if err := os.MkdirAll(linearDir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(linearDir, "config.example.yaml")
	if err := os.WriteFile(path, ExampleContent, 0o644); err != nil {
		t.Fatal(err)
	}

	info, _ := os.Stat(path)
	modBefore := info.ModTime()

	err := EnsureExampleFile(func() (string, error) { return dir, nil })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	info, _ = os.Stat(path)
	if info.ModTime() != modBefore {
		t.Error("file was rewritten even though content was current")
	}
}

func TestEnsureExampleFile_ConfigDirError(t *testing.T) {
	err := EnsureExampleFile(func() (string, error) { return "", fmt.Errorf("no dir") })
	if err == nil {
		t.Fatal("expected error when configDir fails")
	}
}
