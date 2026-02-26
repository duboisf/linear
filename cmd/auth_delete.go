package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newAuthDeleteCmd(opts Options) *cobra.Command {
	return &cobra.Command{
		Use:   "delete",
		Short: "Remove stored API key",
		Args:  cobra.NoArgs,
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// Check if a key exists first.
			if _, err := opts.KeyringProvider.GetAPIKey(); err != nil {
				return fmt.Errorf("no API key configured")
			}

			var deleted bool

			// Delete from native store.
			if opts.NativeStore != nil {
				if err := opts.NativeStore.DeleteAPIKey(); err == nil {
					deleted = true
				}
			}

			// Delete from file store.
			if opts.FileStore != nil {
				if err := opts.FileStore.DeleteAPIKey(); err == nil {
					deleted = true
				}
			}

			if !deleted {
				return fmt.Errorf("failed to delete API key")
			}

			fmt.Fprintln(opts.Stdout, "API key deleted.")
			return nil
		},
	}
}
