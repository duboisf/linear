package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/duboisf/linear/internal/api"
	"github.com/duboisf/linear/internal/format"
)

// validOutputFormats lists the accepted values for --output.
var validOutputFormats = []string{"plain", "markdown", "json", "yaml"}

// newIssueGetCmd creates the "issue get" subcommand that displays detailed
// information for a specific issue.
func newIssueGetCmd(opts Options) *cobra.Command {
	var (
		outputFormat string
		user         string
	)

	cmd := &cobra.Command{
		Use:     "get [IDENTIFIER]",
		Aliases: []string{"show", "view"},
		Short:   "Get details for a specific issue",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := resolveClient(cmd, opts)
			if err != nil {
				return err
			}

			var identifier string
			if len(args) > 0 {
				identifier = args[0]
			} else {
				var issues []issueForCompletion
				if user != "" {
					issues, err = fetchUserIssues(cmd.Context(), client, user)
				} else {
					issues, err = fetchMyIssues(cmd.Context(), client)
				}
				if err != nil {
					return fmt.Errorf("listing issues: %w", err)
				}
				identifier, err = fzfPickIssue(issues)
				if err != nil {
					return err
				}
				if identifier == "" {
					return nil // user cancelled
				}
			}

			resp, err := api.GetIssue(cmd.Context(), client, identifier)
			if err != nil {
				return fmt.Errorf("getting issue: %w", err)
			}

			if resp.Issue == nil {
				return fmt.Errorf("issue %s not found", identifier)
			}

			var out string
			switch outputFormat {
			case "json":
				out, err = format.FormatIssueDetailJSON(resp.Issue)
				if err != nil {
					return err
				}
			case "yaml":
				out = format.FormatIssueDetailYAML(resp.Issue)
			case "markdown", "md":
				out = format.FormatIssueDetailMarkdown(resp.Issue)
			default:
				out = format.FormatIssueDetail(resp.Issue, format.ColorEnabled(cmd.OutOrStdout()))
			}
			fmt.Fprint(opts.Stdout, out)

			return nil
		},
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) > 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			if user != "" {
				return completeUserIssues(cmd, opts, user)
			}
			return completeMyIssues(cmd, opts)
		},
	}

	cmd.Flags().StringVarP(&outputFormat, "output", "o", "plain", "Output format: plain, markdown, json, yaml")
	cmd.RegisterFlagCompletionFunc("output", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return validOutputFormats, cobra.ShellCompDirectiveNoFileComp
	})

	cmd.Flags().StringVarP(&user, "user", "u", "", "User whose issues to browse")
	cmd.RegisterFlagCompletionFunc("user", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return completeUserNames(cmd, opts)
	})

	return cmd
}
