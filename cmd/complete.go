package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/duboisf/linear/internal/api"
)

// completeUsers returns shell completions for user selection: @my first, then
// team member first names from the API.
func completeUsers(cmd *cobra.Command, opts Options) ([]string, cobra.ShellCompDirective) {
	client, err := resolveClient(cmd, opts)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	resp, err := api.UsersForCompletion(cmd.Context(), client, 100)
	if err != nil || resp.Users == nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	comps := make([]string, 0, len(resp.Users.Nodes)+1)
	comps = append(comps, "@my\tYour own issues")
	for _, u := range resp.Users.Nodes {
		comps = append(comps, userCompletionEntry(u.DisplayName, u.Name))
	}
	return comps, cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveKeepOrder
}

// userCompletionEntry formats a user as a shell completion entry:
// "lowercase_first_name\tFull Name".
func userCompletionEntry(displayName, fullName string) string {
	parts := strings.Fields(displayName)
	if len(parts) == 0 {
		return fmt.Sprintf("%s\t%s", strings.ToLower(displayName), fullName)
	}
	return fmt.Sprintf("%s\t%s", strings.ToLower(parts[0]), fullName)
}
