package cmd

import "github.com/spf13/cobra"

// newUserCmd creates the parent "user" command that groups user subcommands.
func newUserCmd(opts Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "user",
		Aliases: []string{"u"},
		Short:   "Manage Linear users",
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
	}
	cmd.AddCommand(
		newUserListCmd(opts),
		newUserGetCmd(opts),
	)
	return cmd
}
