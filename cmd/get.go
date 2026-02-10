package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/duboisf/linear/internal/api"
	"github.com/duboisf/linear/internal/format"
)

func newGetCmd(opts Options) *cobra.Command {
	var outputFormat string

	cmd := &cobra.Command{
		Use:   "get <user> <resource> [identifier]",
		Short: "Get a resource (issue)",
		Args:  cobra.RangeArgs(2, 3),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			switch len(args) {
			case 0: // completing user
				return completeUsers(cmd, opts)
			case 1: // completing resource
				return []string{
					"issue\tGet issue details",
				}, cobra.ShellCompDirectiveNoFileComp
			case 2: // completing identifier
				if args[0] == "@my" {
					return completeMyIssues(cmd, opts)
				}
				return completeUserIssues(cmd, opts, args[0])
			default:
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			resource := args[1]

			client, err := resolveClient(cmd, opts)
			if err != nil {
				return err
			}

			var identifier string
			if len(args) == 3 {
				identifier = args[2]
			} else {
				issues, err := fetchIssuesForUser(cmd.Context(), client, args[0])
				if err != nil {
					return err
				}
				identifier, err = fzfPickIssue(issues)
				if err != nil {
					return err
				}
				if identifier == "" {
					return nil // user cancelled
				}
			}

			switch resource {
			case "issue":
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

			default:
				return fmt.Errorf("unsupported resource %q; valid resources: issue", resource)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&outputFormat, "output", "o", "plain", "Output format: plain, markdown, json, yaml")
	cmd.RegisterFlagCompletionFunc("output", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return validOutputFormats, cobra.ShellCompDirectiveNoFileComp
	})

	return cmd
}
