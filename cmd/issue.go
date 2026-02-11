package cmd

import "github.com/spf13/cobra"

// newIssueCmd creates the parent "issue" command that groups issue subcommands.
func newIssueCmd(opts Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "issue",
		Aliases: []string{"i"},
		Short:   "Manage Linear issues",
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
	}
	cmd.AddCommand(
		newIssueListCmd(opts),
		newIssueGetCmd(opts),
		newIssueWorktreeCmd(opts),
	)
	return cmd
}
