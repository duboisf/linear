package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// newCacheCmd creates the parent "cache" command.
func newCacheCmd(opts Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cache",
		Short: "Manage the local cache",
	}
	cmd.AddCommand(newCacheClearCmd(opts))
	return cmd
}

// newCacheClearCmd creates the "cache clear" subcommand.
func newCacheClearCmd(opts Options) *cobra.Command {
	return &cobra.Command{
		Use:   "clear",
		Short: "Remove all cached data",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			n, err := opts.Cache.Clear()
			if err != nil {
				return fmt.Errorf("clearing cache: %w", err)
			}
			if n == 0 {
				fmt.Fprintln(opts.Stdout, "Cache is already empty.")
				return nil
			}
			fmt.Fprintf(opts.Stdout, "Cleared %d cached file(s).\n", n)
			return nil
		},
		ValidArgsFunction: cobra.NoFileCompletions,
	}
}
