package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/duboisf/linear/internal/config"
	"github.com/duboisf/linear/internal/prompt"
)

// newIssueRunCommandCmd creates the hidden "issue run-command" subcommand
// used by the fzf ctrl-o binding. It runs a user-configured custom command
// with issue data available via Go template fields.
func newIssueRunCommandCmd(opts Options) *cobra.Command {
	var issueDataFile string
	var identifier string
	var execFile string

	cmd := &cobra.Command{
		Use:    "run-command",
		Short:  "Run a custom command (used by fzf binding)",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			var commands []config.Command
			if opts.Config != nil {
				commands = opts.Config.Interactive.Commands
			}
			if len(commands) == 0 {
				// Write to /dev/tty so the message is visible inside fzf's execute().
				tty, err := os.OpenFile("/dev/tty", os.O_WRONLY, 0)
				if err != nil {
					tty = os.Stderr
				}
				fmt.Fprintln(tty, "No commands configured.")
				fmt.Fprintln(tty, "")
				fmt.Fprintln(tty, "Add commands to your config file to use ctrl-o.")
				fmt.Fprintln(tty, "Run 'linear config edit' to open your config file.")
				fmt.Fprintln(tty, "")
				fmt.Fprintln(tty, "Example:")
				fmt.Fprintln(tty, "  interactive:")
				fmt.Fprintln(tty, "    commands:")
				fmt.Fprintln(tty, `      - name: "Claude"`)
				fmt.Fprintln(tty, `        command: "claude \"Work on {{.Identifier}}: {{.Title}}\""`)
				fmt.Fprintln(tty, "")
				fmt.Fprint(tty, "Press enter to continue...")
				// Wait for keypress so the user can read the message.
				ttyIn, err := os.Open("/dev/tty")
				if err == nil {
					buf := make([]byte, 1)
					_, _ = ttyIn.Read(buf)
					ttyIn.Close()
				}
				if tty != os.Stderr {
					tty.Close()
				}
				return nil
			}

			// Read issue data from cache file, polling briefly for prefetch.
			var issueData prompt.IssueData
			if issueDataFile != "" {
				for range 20 {
					data, err := os.ReadFile(issueDataFile)
					if err == nil && len(data) > 0 {
						if err := json.Unmarshal(data, &issueData); err == nil {
							break
						}
					}
					time.Sleep(100 * time.Millisecond)
				}
			}

			// Fallback: populate identifier from flag if issue data wasn't loaded.
			if issueData.Identifier == "" && identifier != "" {
				issueData.Identifier = identifier
			}

			selected := commands[0]
			if len(commands) > 1 {
				lines := make([]string, len(commands))
				for i, c := range commands {
					lines[i] = fmt.Sprintf("%d\t%s", i, c.Name)
				}
				picked, err := fzfPickValue("Run command", lines, true)
				if err != nil || picked == "" {
					return err // user cancelled
				}
				idxStr, _, _ := strings.Cut(picked, "\t")
				var idx int
				if _, err := fmt.Sscanf(idxStr, "%d", &idx); err != nil || idx < 0 || idx >= len(commands) {
					return fmt.Errorf("invalid selection: %q", picked)
				}
				selected = commands[idx]
			}

			rendered, err := prompt.Render(selected.Command, issueData)
			if err != nil {
				return fmt.Errorf("rendering command template: %w", err)
			}

			// For exec commands, atomically write the rendered command to a
			// file so the caller can exec it after fzf exits.
			if selected.Exec && execFile != "" {
				tmp, err := os.CreateTemp(filepath.Dir(execFile), ".exec-*")
				if err != nil {
					return fmt.Errorf("creating temp exec file: %w", err)
				}
				if _, err := tmp.WriteString(rendered); err != nil {
					tmp.Close()
					os.Remove(tmp.Name())
					return fmt.Errorf("writing exec file: %w", err)
				}
				tmp.Close()
				return os.Rename(tmp.Name(), execFile)
			}

			// Remove stale exec file from a previous invocation.
			if execFile != "" {
				os.Remove(execFile)
			}

			shPath := "/bin/sh"
			return syscall.Exec(shPath, []string{"sh", "-c", rendered}, os.Environ())
		},
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
	}

	cmd.Flags().StringVar(&issueDataFile, "issue-data-file", "", "Path to cached issue data JSON file")
	cmd.Flags().StringVar(&identifier, "identifier", "", "Fallback issue identifier")
	cmd.Flags().StringVar(&execFile, "exec-file", "", "Path to write rendered command for deferred exec")

	return cmd
}
