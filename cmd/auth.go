package cmd

import "github.com/spf13/cobra"

// newAuthCmd creates the parent "auth" command that groups auth subcommands.
func newAuthCmd(opts Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage authentication",
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
	}
	cmd.AddCommand(
		newAuthSetupCmd(opts),
		newAuthStatusCmd(opts),
	)
	return cmd
}
