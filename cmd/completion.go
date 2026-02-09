package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// newCompletionCmd creates the "completion" command that generates shell
// completion scripts for bash and zsh.
func newCompletionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:       "completion [bash|zsh]",
		Short:     "Generate shell completion script",
		Args:      cobra.ExactArgs(1),
		ValidArgs: []string{"bash", "zsh"},
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			switch args[0] {
			case "bash":
				return cmd.Root().GenBashCompletionV2(out, true)
			case "zsh":
				return cmd.Root().GenZshCompletion(out)
			default:
				return fmt.Errorf("unsupported shell: %s", args[0])
			}
		},
	}

	return cmd
}
