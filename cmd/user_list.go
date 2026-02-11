package cmd

import (
	"fmt"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/duboisf/linear/internal/api"
	"github.com/duboisf/linear/internal/format"
)

// isIntegrationUser returns true if the user is a Linear integration/bot
// (identified by email ending in linear.app).
func isIntegrationUser(email string) bool {
	return strings.HasSuffix(email, "linear.app")
}

// newUserListCmd creates the "user list" subcommand that lists users
// in the organization.
func newUserListCmd(opts Options) *cobra.Command {
	var (
		limit       int
		includeBots bool
	)

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List users in the organization",
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if limit <= 0 {
				return fmt.Errorf("--limit must be greater than 0, got %d", limit)
			}

			client, err := resolveClient(cmd, opts)
			if err != nil {
				return err
			}

			resp, err := api.ListUsers(cmd.Context(), client, limit, nil)
			if err != nil {
				return fmt.Errorf("listing users: %w", err)
			}

			if resp.Users == nil {
				return fmt.Errorf("no users data returned from API")
			}

			users := resp.Users.Nodes
			if !includeBots {
				filtered := make([]*api.ListUsersUsersUserConnectionNodesUser, 0, len(users))
				for _, u := range users {
					if !isIntegrationUser(u.Email) {
						filtered = append(filtered, u)
					}
				}
				users = filtered
			}

			slices.SortFunc(users, func(a, b *api.ListUsersUsersUserConnectionNodesUser) int {
				return strings.Compare(
					strings.ToLower(a.DisplayName),
					strings.ToLower(b.DisplayName),
				)
			})

			out := format.FormatUserList(users, format.ColorEnabled(cmd.OutOrStdout()))
			fmt.Fprint(opts.Stdout, out)

			return nil
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "n", 50, "Maximum number of users to return")
	cmd.Flags().BoolVar(&includeBots, "include-bots", false, "Include integration/bot users")

	return cmd
}
