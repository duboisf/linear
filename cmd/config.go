package cmd

import (
	"github.com/spf13/cobra"
)

// newConfigCmd creates the parent "config" command.
func newConfigCmd(opts Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage CLI configuration",
	}
	cmd.AddCommand(newConfigEditCmd(opts))
	return cmd
}
