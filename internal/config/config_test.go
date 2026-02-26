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

func TestLoad_DefaultClaudeModes(t *testing.T) {
	dir := t.TempDir()
	cfg, err := Load(func() (string, error) { return dir, nil })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Interactive.ClaudeModes) != len(DefaultClaudeModes) {
		t.Fatalf("got %d modes, want %d", len(cfg.Interactive.ClaudeModes), len(DefaultClaudeModes))
	}
	for i, m := range cfg.Interactive.ClaudeModes {
		if m != DefaultClaudeModes[i] {
			t.Errorf("mode[%d] got %+v, want %+v", i, m, DefaultClaudeModes[i])
		}
	}
}

func TestLoad_CustomClaudeModes(t *testing.T) {
	dir := t.TempDir()
	configDir := filepath.Join(dir, "linear")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		t.Fatal(err)
	}
	content := `interactive:
  claude_modes:
    - label: "Normal"
    - label: "Yolo"
      args: "--dangerously-skip-permissions"
    - label: "Resume"
      args: "--resume"
`
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(func() (string, error) { return dir, nil })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Interactive.ClaudeModes) != 3 {
		t.Fatalf("got %d modes, want 3", len(cfg.Interactive.ClaudeModes))
	}
	if cfg.Interactive.ClaudeModes[0].Label != "Normal" {
		t.Errorf("mode[0].Label = %q, want %q", cfg.Interactive.ClaudeModes[0].Label, "Normal")
	}
	if cfg.Interactive.ClaudeModes[1].Args != "--dangerously-skip-permissions" {
		t.Errorf("mode[1].Args = %q, want %q", cfg.Interactive.ClaudeModes[1].Args, "--dangerously-skip-permissions")
	}
	if cfg.Interactive.ClaudeModes[2].Args != "--resume" {
		t.Errorf("mode[2].Args = %q, want %q", cfg.Interactive.ClaudeModes[2].Args, "--resume")
	}
}

func TestLoad_EmptyClaudeModesUsesDefault(t *testing.T) {
	dir := t.TempDir()
	configDir := filepath.Join(dir, "linear")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		t.Fatal(err)
	}
	content := "interactive:\n  claude_modes: []\n"
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(func() (string, error) { return dir, nil })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Interactive.ClaudeModes) != len(DefaultClaudeModes) {
		t.Fatalf("got %d modes, want %d", len(cfg.Interactive.ClaudeModes), len(DefaultClaudeModes))
	}
}
