package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/duboisf/linear/internal/config"
)

// newConfigEditCmd creates the "config edit" subcommand.
func newConfigEditCmd(opts Options) *cobra.Command {
	return &cobra.Command{
		Use:   "edit",
		Short: "Open the config file in your editor",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			path := config.FilePath()
			if path == "" {
				return fmt.Errorf("could not determine config directory")
			}

			// Ensure parent directory exists.
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				return fmt.Errorf("creating config directory: %w", err)
			}

			// Seed an empty config file if it doesn't exist yet.
			if _, err := os.Stat(path); os.IsNotExist(err) {
				if err := os.WriteFile(path, config.DefaultConfigContent, 0o644); err != nil {
					return fmt.Errorf("creating config file: %w", err)
				}
			}

			fmt.Fprintf(opts.Stderr, "Editing %s\n", path)

			parts := strings.Fields(editorCmd())
			editor := exec.Command(parts[0], append(parts[1:], path)...)
			editor.Stdin = os.Stdin
			editor.Stdout = os.Stdout
			editor.Stderr = os.Stderr

			if err := editor.Run(); err != nil {
				return fmt.Errorf("running editor: %w", err)
			}
			return nil
		},
		ValidArgsFunction: cobra.NoFileCompletions,
	}
}
