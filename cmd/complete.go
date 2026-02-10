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
		firstName := strings.ToLower(strings.Fields(u.DisplayName)[0])
		comps = append(comps, fmt.Sprintf("%s\t%s", firstName, u.Name))
	}
	return comps, cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveKeepOrder
}
