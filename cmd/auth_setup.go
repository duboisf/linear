package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/duboisf/linear/internal/api"
	"github.com/duboisf/linear/internal/keyring"
)

func newAuthSetupCmd(opts Options) *cobra.Command {
	return &cobra.Command{
		Use:   "setup",
		Short: "Configure API key authentication",
		Args:  cobra.NoArgs,
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// Warn if overriding an existing token.
			if _, err := opts.KeyringProvider.GetAPIKey(); err == nil {
				fmt.Fprintln(opts.Stderr, "Warning: an API key is already configured. Entering a new key will override it.")
			}

			fmt.Fprintln(opts.Stderr, "Create a personal API key at: https://linear.app/settings/api")

			newKey, err := opts.KeyReader.ReadAPIKey(opts.Stderr)
			if err != nil {
				return fmt.Errorf("reading API key: %w", err)
			}

			// Validate the token before storing.
			fmt.Fprintln(opts.Stderr, "Validating token...")
			client := opts.NewAPIClient(newKey)
			resp, err := api.Viewer(cmd.Context(), client)
			if err != nil {
				return fmt.Errorf("token validation failed: %w", err)
			}
			if resp.Viewer == nil {
				return fmt.Errorf("token validation failed: invalid token")
			}

			// Token is valid — store it.
			keyring.StoreKey(newKey, keyring.ResolveOptions{
				NativeStore: opts.NativeStore,
				FileStore:   opts.FileStore,
				Stdin:       opts.Stdin,
				MsgWriter:   opts.Stderr,
			})

			fmt.Fprintf(opts.Stdout, "Authenticated as %s (%s). API key saved.\n",
				resp.Viewer.Name, resp.Viewer.Email)
			return nil
		},
	}
}
