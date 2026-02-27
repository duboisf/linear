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
	if len(cfg.Interactive.Commands) != 0 {
		t.Errorf("got %d commands, want 0", len(cfg.Interactive.Commands))
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
	if len(cfg.Interactive.Commands) != 0 {
		t.Errorf("got %d commands, want 0", len(cfg.Interactive.Commands))
	}
}

func TestLoad_CustomCommands(t *testing.T) {
	dir := t.TempDir()
	configDir := filepath.Join(dir, "linear")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		t.Fatal(err)
	}
	content := `interactive:
  commands:
    - name: "Claude"
      command: "claude \"Work on {{.Identifier}}: {{.Title}}\""
    - name: "Open in browser"
      command: "xdg-open {{.URL}}"
`
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(func() (string, error) { return dir, nil })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Interactive.Commands) != 2 {
		t.Fatalf("got %d commands, want 2", len(cfg.Interactive.Commands))
	}
	if cfg.Interactive.Commands[0].Name != "Claude" {
		t.Errorf("commands[0].Name = %q, want %q", cfg.Interactive.Commands[0].Name, "Claude")
	}
	if cfg.Interactive.Commands[1].Command != "xdg-open {{.URL}}" {
		t.Errorf("commands[1].Command = %q, want %q", cfg.Interactive.Commands[1].Command, "xdg-open {{.URL}}")
	}
}

func TestLoad_EmptyCommands(t *testing.T) {
	dir := t.TempDir()
	configDir := filepath.Join(dir, "linear")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		t.Fatal(err)
	}
	content := "interactive:\n  commands: []\n"
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(func() (string, error) { return dir, nil })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Interactive.Commands) != 0 {
		t.Fatalf("got %d commands, want 0", len(cfg.Interactive.Commands))
	}
}
