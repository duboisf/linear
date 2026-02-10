package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/duboisf/linear/internal/api"
	"github.com/duboisf/linear/internal/format"
)

func newGetCmd(opts Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <user> issue <identifier>",
		Short: "Get details for a resource",
		Args:  cobra.ExactArgs(3),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			switch len(args) {
			case 0: // completing user
				client, err := resolveClient(cmd, opts)
				if err != nil {
					return nil, cobra.ShellCompDirectiveNoFileComp
				}
				resp, err := api.UsersForCompletion(cmd.Context(), client, 100)
				if err != nil || resp.Users == nil {
					return nil, cobra.ShellCompDirectiveNoFileComp
				}
				comps := []string{"@my\tYour own issues"}
				for _, u := range resp.Users.Nodes {
					firstName := strings.ToLower(strings.Fields(u.DisplayName)[0])
					comps = append(comps, fmt.Sprintf("%s\t%s", firstName, u.DisplayName))
				}
				return comps, cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveKeepOrder
			case 1: // completing resource
				return []string{"issue\tGet issue details"}, cobra.ShellCompDirectiveNoFileComp
			case 2: // completing identifier
				client, err := resolveClient(cmd, opts)
				if err != nil {
					return nil, cobra.ShellCompDirectiveNoFileComp
				}
				if args[0] == "@my" {
					resp, err := api.ActiveIssuesForCompletion(cmd.Context(), client, 100)
					if err != nil || resp.Viewer == nil || resp.Viewer.AssignedIssues == nil {
						return nil, cobra.ShellCompDirectiveNoFileComp
					}
					var comps []string
					for _, issue := range resp.Viewer.AssignedIssues.Nodes {
						comps = append(comps, fmt.Sprintf("%s\t%s", issue.Identifier, issue.Title))
					}
					return comps, cobra.ShellCompDirectiveNoFileComp
				}
				resp, err := api.UserIssuesForCompletion(cmd.Context(), client, 100, args[0])
				if err != nil || resp.Issues == nil {
					return nil, cobra.ShellCompDirectiveNoFileComp
				}
				var comps []string
				for _, issue := range resp.Issues.Nodes {
					comps = append(comps, fmt.Sprintf("%s\t%s", issue.Identifier, issue.Title))
				}
				return comps, cobra.ShellCompDirectiveNoFileComp
			default:
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			resource := args[1]
			identifier := args[2]

			if resource != "issue" {
				return fmt.Errorf("unsupported resource %q; valid resources: issue", resource)
			}

			client, err := resolveClient(cmd, opts)
			if err != nil {
				return err
			}

			resp, err := api.GetIssue(cmd.Context(), client, identifier)
			if err != nil {
				return fmt.Errorf("getting issue: %w", err)
			}

			if resp.Issue == nil {
				return fmt.Errorf("issue %s not found", identifier)
			}

			out := format.FormatIssueDetail(resp.Issue, format.ColorEnabled(cmd.OutOrStdout()))
			fmt.Fprint(opts.Stdout, out)

			return nil
		},
	}

	return cmd
}
