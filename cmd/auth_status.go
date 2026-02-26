package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/duboisf/linear/internal/api"
)

func newAuthStatusCmd(opts Options) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Check authentication status",
		Args:  cobra.NoArgs,
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			apiKey, err := opts.KeyringProvider.GetAPIKey()
			if err != nil {
				return fmt.Errorf("not authenticated. Run 'linear auth setup' to configure your API key")
			}

			client := opts.NewAPIClient(apiKey)
			resp, err := api.Viewer(cmd.Context(), client)
			if err != nil {
				return fmt.Errorf("authentication failed: %w", err)
			}
			if resp.Viewer == nil {
				return fmt.Errorf("authentication failed: invalid token")
			}

			fmt.Fprintf(opts.Stdout, "Authenticated as %s (%s)\n", resp.Viewer.Name, resp.Viewer.Email)
			return nil
		},
	}
}
