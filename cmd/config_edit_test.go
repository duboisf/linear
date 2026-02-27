package cmd_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/duboisf/linear/cmd"
)

func TestConfigEdit_CreatesFileAndOpensEditor(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	// Use "true" as the editor so it exits immediately.
	t.Setenv("EDITOR", "true")

	var stdout, stderr bytes.Buffer
	root := cmd.NewRootCmd(cmd.Options{
		Stdout: &stdout,
		Stderr: &stderr,
	})
	root.SetArgs([]string{"config", "edit"})

	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	configPath := filepath.Join(tmpDir, "linear", "config.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("config file should exist: %v", err)
	}

	if len(data) == 0 {
		t.Error("config file should contain default content")
	}

	got := string(data)
	if !bytes.Contains(data, []byte("commands")) {
		t.Errorf("expected default config to contain commands, got: %s", got)
	}

	if stderrStr := stderr.String(); stderrStr == "" {
		t.Error("expected stderr to contain the config file path")
	}
}

func TestConfigEdit_NoArgs(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	root := cmd.NewRootCmd(cmd.Options{
		Stdout: &stdout,
		Stderr: &stderr,
	})
	root.SetArgs([]string{"config", "edit", "extra"})

	if err := root.Execute(); err == nil {
		t.Fatal("expected error for extra args")
	}
}
