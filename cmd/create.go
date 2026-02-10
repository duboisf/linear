package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newCreateCmd(opts Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create <user> <resource> <identifier>",
		Short: "Create a resource (worktree)",
		Args:  cobra.ExactArgs(3),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			switch len(args) {
			case 0: // completing user
				return completeUsers(cmd, opts)
			case 1: // completing resource
				return []string{
					"worktree\tCreate a git worktree for an issue",
				}, cobra.ShellCompDirectiveNoFileComp
			case 2: // completing identifier
				if args[0] == "@my" {
					return completeMyIssues(cmd, opts)
				}
				return completeUserIssues(cmd, opts, args[0])
			default:
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			resource := args[1]
			identifier := args[2]

			client, err := resolveClient(cmd, opts)
			if err != nil {
				return err
			}

			switch resource {
			case "worktree":
				return runWorktreeCreate(cmd.Context(), client, identifier, opts.GitWorktreeCreator, opts.Stdout)
			default:
				return fmt.Errorf("unsupported resource %q; valid resources: worktree", resource)
			}
		},
	}

	return cmd
}
