package config

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_MissingFile(t *testing.T) {
	dir := t.TempDir()
	cfg, err := Load(func() (string, error) { return dir, nil })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Interactive.ClaudePrompt != DefaultClaudePrompt {
		t.Errorf("got %q, want %q", cfg.Interactive.ClaudePrompt, DefaultClaudePrompt)
	}
}

func TestLoad_CustomPrompt(t *testing.T) {
	dir := t.TempDir()
	configDir := filepath.Join(dir, "linear")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		t.Fatal(err)
	}
	content := "interactive:\n  claude_prompt: \"custom prompt {identifier}\"\n"
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(func() (string, error) { return dir, nil })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Interactive.ClaudePrompt != "custom prompt {identifier}" {
		t.Errorf("got %q, want %q", cfg.Interactive.ClaudePrompt, "custom prompt {identifier}")
	}
}

func TestLoad_EmptyPromptUsesDefault(t *testing.T) {
	dir := t.TempDir()
	configDir := filepath.Join(dir, "linear")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		t.Fatal(err)
	}
	content := "interactive:\n  claude_prompt: \"\"\n"
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(func() (string, error) { return dir, nil })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Interactive.ClaudePrompt != DefaultClaudePrompt {
		t.Errorf("got %q, want %q", cfg.Interactive.ClaudePrompt, DefaultClaudePrompt)
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	configDir := filepath.Join(dir, "linear")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(":\n\t:bad"), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := Load(func() (string, error) { return dir, nil })
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestLoad_ConfigDirError(t *testing.T) {
	cfg, err := Load(func() (string, error) { return "", fmt.Errorf("no config dir") })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Interactive.ClaudePrompt != DefaultClaudePrompt {
		t.Errorf("got %q, want %q", cfg.Interactive.ClaudePrompt, DefaultClaudePrompt)
	}
}
