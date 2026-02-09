package cmd

import "github.com/spf13/cobra"

// newIssueCmd creates the parent "issue" command that groups issue subcommands.
func newIssueCmd(opts Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "issue",
		Aliases: []string{"i"},
		Short:   "Manage Linear issues",
	}
	cmd.AddCommand(
		newIssueListCmd(opts),
		newIssueGetCmd(opts),
	)
	return cmd
}
