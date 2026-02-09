package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/duboisf/linear/internal/api"
	"github.com/duboisf/linear/internal/format"
)

// newIssueGetCmd creates the "issue get" subcommand that displays detailed
// information for a specific issue.
func newIssueGetCmd(opts Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "get <IDENTIFIER>",
		Aliases: []string{"show", "view"},
		Short:   "Get details for a specific issue",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := resolveClient(cmd, opts)
			if err != nil {
				return err
			}

			resp, err := api.GetIssue(cmd.Context(), client, args[0])
			if err != nil {
				return fmt.Errorf("getting issue: %w", err)
			}

			if resp.Issue == nil {
				return fmt.Errorf("issue %s not found", args[0])
			}

			out := format.FormatIssueDetail(resp.Issue, format.ColorEnabled(cmd.OutOrStdout()))
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

			resp, err := api.ActiveIssuesForCompletion(cmd.Context(), client, 100)
			if err != nil {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}

			if resp.Viewer == nil || resp.Viewer.AssignedIssues == nil {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}

			var completions []string
			for _, issue := range resp.Viewer.AssignedIssues.Nodes {
				completions = append(completions, fmt.Sprintf("%s\t%s", issue.Identifier, issue.Title))
			}

			return completions, cobra.ShellCompDirectiveNoFileComp
		},
	}

	return cmd
}
