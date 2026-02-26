package cmd

import (
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/duboisf/linear/internal/format"
)

// newIssuePickCycleCmd creates the hidden "issue pick-cycle" subcommand
// used by the fzf ctrl-y binding. It presents a cycle picker via nested fzf
// and writes the selected cycle number to a state file.
func newIssuePickCycleCmd(opts Options) *cobra.Command {
	var stateFile string

	cmd := &cobra.Command{
		Use:    "pick-cycle",
		Short:  "Pick a cycle filter (used by fzf binding)",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if stateFile == "" {
				return fmt.Errorf("--state-file is required")
			}

			client, err := resolveClient(cmd, opts)
			if err != nil {
				return err
			}

			timeNow := opts.TimeNow
			if timeNow == nil {
				timeNow = time.Now
			}

			resp, err := listCyclesCached(cmd.Context(), client, opts.Cache, timeNow)
			if err != nil {
				return fmt.Errorf("listing cycles: %w", err)
			}
			if resp.Cycles == nil || len(resp.Cycles.Nodes) == 0 {
				return fmt.Errorf("no cycles found")
			}

			// Read current cycle value from state file to mark it.
			currentValue := ""
			if data, err := os.ReadFile(stateFile); err == nil {
				currentValue = strings.TrimSpace(string(data))
			}

			// Sort cycles: current first, then next, upcoming, previous.
			type cycleEntry struct {
				rank int
				line string
			}
			var entries []cycleEntry
			for _, c := range resp.Cycles.Nodes {
				if c.IsPast && !c.IsPrevious {
					continue // skip old past cycles
				}

				cycleNum := fmt.Sprintf("%.0f", c.Number)
				marker := "  "
				if currentValue == cycleNum {
					marker = "* "
				}

				label := fmt.Sprintf("#%.0f", c.Number)
				if c.Name != nil && *c.Name != "" {
					label += " " + *c.Name
				}

				var status string
				rank := 3 // default for upcoming
				switch {
				case c.IsActive:
					status = format.Colorize(true, format.Green, "Current")
					rank = 0
				case c.IsNext:
					status = format.Colorize(true, format.Yellow, "Next")
					rank = 1
				case c.IsFuture:
					status = format.Colorize(true, format.Cyan, "Upcoming")
					rank = 2
				case c.IsPrevious:
					status = format.Colorize(true, format.Gray, "Previous")
					rank = 4
				}

				dates := formatCycleDateRange(c.StartsAt, c.EndsAt)
				if dates != "" {
					label += "  " + format.Colorize(true, format.Gray, dates)
				}

				entries = append(entries, cycleEntry{
					rank: rank,
					line: fmt.Sprintf("%s\t%s%s  %s", cycleNum, marker, label, status),
				})
			}
			slices.SortFunc(entries, func(a, b cycleEntry) int {
				return a.rank - b.rank
			})
			var lines []string
			for _, e := range entries {
				lines = append(lines, e.line)
			}

			// Add "All cycles" option.
			allMarker := "  "
			if currentValue == "all" {
				allMarker = "* "
			}
			lines = append(lines, fmt.Sprintf("all\t%s%s", allMarker, format.Colorize(true, format.Gray, "All cycles")))

			selected, err := fzfPickValue("Switch cycle filter", lines, true)
			if err != nil || selected == "" {
				return err // user cancelled — no change
			}

			// Extract the cycle number or "all" (first tab-delimited field).
			value, _, _ := strings.Cut(selected, "\t")

			// Write cycle value to state file.
			if err := os.WriteFile(stateFile, []byte(value), 0o644); err != nil {
				return fmt.Errorf("writing state file: %w", err)
			}

			// Write header to <state-file>.header.
			headerFile := stateFile + ".header"
			header := ""
			if value != "all" {
				// Resolve cycle info for header formatting.
				if n, err := strconv.ParseFloat(value, 64); err == nil {
					for _, c := range resp.Cycles.Nodes {
						if c.Number == n {
							name := ""
							if c.Name != nil {
								name = *c.Name
							}
							ci := cycleInfo{
								Id:       c.Id,
								Number:   c.Number,
								Name:     name,
								StartsAt: c.StartsAt,
								EndsAt:   c.EndsAt,
							}
							header = ci.formatHeader(true)
							break
						}
					}
				}
			}
			if err := os.WriteFile(headerFile, []byte(header), 0o644); err != nil {
				return fmt.Errorf("writing header file: %w", err)
			}

			return nil
		},
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
	}

	cmd.Flags().StringVar(&stateFile, "state-file", "", "Path to cycle state file")

	return cmd
}
