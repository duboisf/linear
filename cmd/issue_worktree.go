package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newIssueWorktreeCmd(opts Options) *cobra.Command {
	var user string

	cmd := &cobra.Command{
		Use:     "worktree [IDENTIFIER]",
		Aliases: []string{"wt"},
		Short:   "Create a git worktree for an issue",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := resolveClient(cmd, opts)
			if err != nil {
				return err
			}

			var identifier string
			if len(args) > 0 {
				identifier = args[0]
			} else {
				var issues []issueForCompletion
				if user != "" {
					issues, err = fetchUserIssues(cmd.Context(), client, user)
				} else {
					issues, err = fetchMyIssues(cmd.Context(), client)
				}
				if err != nil {
					return fmt.Errorf("listing issues: %w", err)
				}
				identifier, err = fzfPickIssue(issues)
				if err != nil {
					return err
				}
				if identifier == "" {
					return nil // user cancelled
				}
			}

			return runWorktreeCreate(cmd.Context(), client, identifier, opts.GitWorktreeCreator, opts.Stdout)
		},
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) > 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			if user != "" {
				return completeUserIssues(cmd, opts, user)
			}
			return completeMyIssues(cmd, opts)
		},
	}

	cmd.Flags().StringVarP(&user, "user", "u", "", "User whose issues to browse")
	cmd.RegisterFlagCompletionFunc("user", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return completeUserNames(cmd, opts)
	})

	return cmd
}
