package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/duboisf/linear/internal/config"
)

// newIssuePickClaudeModeCmd creates the hidden "issue pick-claude-mode"
// subcommand used by the fzf ctrl-w binding. It presents a mode picker via
// nested fzf (if multiple modes are configured), then execs into claude.
func newIssuePickClaudeModeCmd(opts Options) *cobra.Command {
	var prompt string

	cmd := &cobra.Command{
		Use:    "pick-claude-mode",
		Short:  "Pick a claude launch mode (used by fzf binding)",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			modes := config.DefaultClaudeModes
			if opts.Config != nil && len(opts.Config.Interactive.ClaudeModes) > 0 {
				modes = opts.Config.Interactive.ClaudeModes
			}

			selected := modes[0]
			if len(modes) > 1 {
				lines := make([]string, len(modes))
				for i, m := range modes {
					lines[i] = fmt.Sprintf("%d\t%s", i, m.Label)
				}
				picked, err := fzfPickValue("Launch claude", lines, true)
				if err != nil || picked == "" {
					return err // user cancelled
				}
				// Extract index from first tab-delimited field.
				idxStr, _, _ := strings.Cut(picked, "\t")
				var idx int
				if _, err := fmt.Sscanf(idxStr, "%d", &idx); err != nil || idx < 0 || idx >= len(modes) {
					return fmt.Errorf("invalid selection: %q", picked)
				}
				selected = modes[idx]
			}

			return execClaude(selected.Args, prompt)
		},
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
	}

	cmd.Flags().StringVar(&prompt, "prompt", "", "Resolved prompt to pass to claude")

	return cmd
}

// execClaude replaces the current process with claude.
func execClaude(modeArgs, prompt string) error {
	claudePath, err := exec.LookPath("claude")
	if err != nil {
		return fmt.Errorf("claude not found: %w", err)
	}

	argv := []string{"claude"}
	if modeArgs != "" {
		argv = append(argv, strings.Fields(modeArgs)...)
	}
	if prompt != "" {
		argv = append(argv, prompt)
	}

	return syscall.Exec(claudePath, argv, os.Environ())
}
