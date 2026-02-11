package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/duboisf/linear/internal/api"
	"github.com/duboisf/linear/internal/format"
)

// newUserGetCmd creates the "user get" subcommand that displays detailed
// information for a specific user.
func newUserGetCmd(opts Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "get <username>",
		Aliases: []string{"show", "view"},
		Short:   "Get details for a user",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			username := args[0]

			client, err := resolveClient(cmd, opts)
			if err != nil {
				return err
			}

			resp, err := api.GetUserByDisplayName(cmd.Context(), client, username)
			if err != nil {
				return fmt.Errorf("getting user: %w", err)
			}

			if resp.Users == nil || len(resp.Users.Nodes) == 0 {
				return fmt.Errorf("user %q not found", username)
			}

			out := format.FormatUserDetail(resp.Users.Nodes[0], format.ColorEnabled(cmd.OutOrStdout()))
			fmt.Fprint(opts.Stdout, out)

			return nil
		},
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) > 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}

			client, err := resolveClient(cmd, opts)
			if err != nil {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}

			resp, err := usersForCompletionCached(cmd.Context(), client, opts.Cache)
			if err != nil || resp.Users == nil {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}

			var completions []string
			for _, u := range resp.Users.Nodes {
				completions = append(completions, userCompletionEntry(u.DisplayName, u.Name))
			}

			return completions, cobra.ShellCompDirectiveNoFileComp
		},
	}

	return cmd
}
