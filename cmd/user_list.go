package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/duboisf/linear/internal/api"
	"github.com/duboisf/linear/internal/format"
)

// newUserListCmd creates the "user list" subcommand that lists users
// in the organization.
func newUserListCmd(opts Options) *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List users in the organization",
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

			out := format.FormatUserList(resp.Users.Nodes, format.ColorEnabled(cmd.OutOrStdout()))
			fmt.Fprint(opts.Stdout, out)

			return nil
		},
	}

	cmd.Flags().IntVarP(&limit, "limit", "n", 50, "Maximum number of users to return")

	return cmd
}
