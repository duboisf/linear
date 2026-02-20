package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/Khan/genqlient/graphql"
	"github.com/spf13/cobra"

	"github.com/duboisf/linear/internal/api"
	"github.com/duboisf/linear/internal/cache"
	"github.com/duboisf/linear/internal/format"
)

// stateTypeOrder maps Linear workflow state types to a sort rank.
// Lower rank = shown first.
var stateTypeOrder = map[string]int{
	"started":   1,
	"unstarted": 2,
	"triage":    3,
	"backlog":   4,
	"completed": 5,
	"canceled":  6,
}

func issueStateRank(issue *issueNode) int {
	if issue.State == nil {
		return 99
	}
	if r, ok := stateTypeOrder[issue.State.Type]; ok {
		return r
	}
	return 99
}

// issuePriorityRank returns a sort rank for priority (lower = more important).
// Priority 0 (None) sorts last.
func issuePriorityRank(p float64) float64 {
	if p == 0 {
		return 99
	}
	return p
}

// newIssueListCmd creates the "issue list" subcommand that lists issues
// assigned to the authenticated user.
func newIssueListCmd(opts Options) *cobra.Command {
	var (
		cycle        string
		interactive  bool
		labelFilter  string
		limit        int
		sortBy       string
		statusFilter string
		user         string
	)

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List issues assigned to you",
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if limit <= 0 {
				return fmt.Errorf("--limit must be greater than 0, got %d", limit)
			}

			client, err := resolveClient(cmd, opts)
			if err != nil {
				return err
			}

			timeNow := opts.TimeNow
			if timeNow == nil {
				timeNow = time.Now
			}
			filter, ci, err := buildIssueFilter(statusFilter, labelFilter, user, cycle, cmd.Context(), client, opts.Cache, timeNow)
			if err != nil {
				return err
			}
			var cycleHeader string
			if ci != nil {
				cycleHeader = ci.formatHeader(format.ColorEnabled(cmd.OutOrStdout()))
			}

			if interactive {
				fetchIssues := func(ctx context.Context) ([]issueForCompletion, error) {
					nodes, err := fetchIssueNodes(ctx, client, user, limit, filter)
					if err != nil {
						return nil, err
					}
					sortIssues(nodes, sortBy)
					return issuesToCompletions(nodes), nil
				}
				selected, err := fzfBrowseIssues(cmd.Context(), client, fetchIssues, opts.Cache, cycleHeader)
				if err != nil {
					return err
				}
				if selected != "" {
					fmt.Fprintln(opts.Stdout, selected)
				}
				return nil
			}

			nodes, err := fetchIssueNodes(cmd.Context(), client, user, limit, filter)
			if err != nil {
				return err
			}

			sortIssues(nodes, sortBy)

			if cycleHeader != "" {
				fmt.Fprintln(opts.Stdout, cycleHeader)
			}

			out := format.FormatIssueList(nodes, format.ColorEnabled(cmd.OutOrStdout()))
			fmt.Fprint(opts.Stdout, out)

			return nil
		},
	}

	cmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "Browse issues interactively with fzf preview")
	_ = cmd.RegisterFlagCompletionFunc("interactive", cobra.NoFileCompletions)
	cmd.Flags().StringVarP(&labelFilter, "label", "l", "", "Filter by label (comma=OR, plus=AND, e.g. bug,devex or bug+frontend)")
	_ = cmd.RegisterFlagCompletionFunc("label", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return completeLabelNames(cmd, opts, toComplete)
	})
	cmd.Flags().StringVarP(&cycle, "cycle", "c", "", "Filter by cycle: all, current, next, previous, or a cycle number (default: current)")
	_ = cmd.RegisterFlagCompletionFunc("cycle", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return completeCycleValues(cmd, opts)
	})
	cmd.Flags().IntVarP(&limit, "limit", "n", 50, "Maximum number of issues to return")
	cmd.Flags().StringVarP(&sortBy, "sort", "s", "status", "Sort by column: status, priority, identifier, title")
	_ = cmd.RegisterFlagCompletionFunc("sort", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"status", "priority", "identifier", "title"}, cobra.ShellCompDirectiveNoFileComp
	})
	cmd.Flags().StringVarP(&statusFilter, "status", "S", "", "Filter by status type: all, or comma-separated list (prefix with ! to exclude, e.g. !completed)")
	_ = cmd.RegisterFlagCompletionFunc("status", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		allStatuses := []string{"started", "todo", "unstarted", "triage", "backlog", "completed", "canceled"}

		parts := strings.Split(toComplete, ",")
		partial := parts[len(parts)-1]
		prefix := strings.Join(parts[:len(parts)-1], ",")

		used := make(map[string]bool, len(parts))
		for _, p := range parts[:len(parts)-1] {
			_, bare, _ := parseNegation(strings.TrimSpace(p))
			used[bare] = true
		}

		negPrefix, _, negated := parseNegation(partial)

		var completions []string
		// Offer "all" only as the sole value.
		if prefix == "" && !negated {
			completions = append(completions, "all")
		}
		for _, s := range allStatuses {
			if used[s] {
				continue
			}
			val := s
			if negated {
				val = negPrefix + s
			}
			if prefix != "" {
				val = prefix + "," + val
			}
			completions = append(completions, val)
		}
		return completions, cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveNoSpace
	})
	cmd.Flags().StringVarP(&user, "user", "u", "", "User whose issues to list")
	_ = cmd.RegisterFlagCompletionFunc("user", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return completeUserNames(cmd, opts)
	})

	return cmd
}

// excludedStateTypes are always filtered out unless --status all is used.
var excludedStateTypes = []string{"completed", "canceled"}

// cutNegationPrefix strips a "!" or "\!" prefix from s.
// The "\!" variant is needed because zsh's BANG_HIST option escapes "!" even
// inside single quotes.
func cutNegationPrefix(s string) (after string, ok bool) {
	if after, ok = strings.CutPrefix(s, `\!`); ok {
		return after, true
	}
	return strings.CutPrefix(s, "!")
}

// statusAliases maps convenience names to Linear workflow state types.
var statusAliases = map[string]string{
	"todo": "unstarted",
}

// resolveStatusAlias returns the canonical state type for a status value,
// expanding known aliases (e.g. "todo" → "unstarted").
func resolveStatusAlias(s string) string {
	if canon, ok := statusAliases[s]; ok {
		return canon
	}
	return s
}

// parseNegation splits a partial completion token into its negation prefix,
// the bare value, and whether negation was present.
// Examples: "!back" → ("!", "back", true),  "\!back" → ("\!", "back", true),
// "started" → ("", "started", false).
func parseNegation(s string) (prefix, bare string, negated bool) {
	if after, ok := strings.CutPrefix(s, `\!`); ok {
		return `\!`, after, true
	}
	if after, ok := strings.CutPrefix(s, "!"); ok {
		return "!", after, true
	}
	return "", s, false
}

// buildIssueFilter constructs an IssueFilter from the flag values.
// When cycle is set, the resolved cycleInfo is returned for header rendering.
func buildIssueFilter(statusFilter, labelFilter, user, cycle string, ctx context.Context, client graphql.Client, c *cache.Cache, timeNow func() time.Time) (*api.IssueFilter, *cycleInfo, error) {
	var filter *api.IssueFilter

	// State filter based on --status flag.
	statusLower := strings.ToLower(strings.TrimSpace(statusFilter))
	switch {
	case statusLower == "all":
		// No state filter — include everything.
		filter = nil
	case statusLower != "":
		// User takes full control: parse positive and !negated values.
		var inTypes, ninTypes []string
		for s := range strings.SplitSeq(statusLower, ",") {
			s = strings.TrimSpace(s)
			if s == "" {
				continue
			}
			if after, ok := cutNegationPrefix(s); ok {
				ninTypes = append(ninTypes, resolveStatusAlias(after))
			} else {
				inTypes = append(inTypes, resolveStatusAlias(s))
			}
		}
		comp := &api.StringComparator{}
		if len(inTypes) > 0 {
			comp.In = inTypes
		}
		if len(ninTypes) > 0 {
			comp.Nin = ninTypes
		}
		filter = &api.IssueFilter{
			State: &api.WorkflowStateFilter{Type: comp},
		}
	default:
		// Default: exclude completed and canceled.
		filter = &api.IssueFilter{
			State: &api.WorkflowStateFilter{
				Type: &api.StringComparator{Nin: excludedStateTypes},
			},
		}
	}

	// Assignee filter for --user flag (only used with ListIssues query).
	// "all" means no assignee filter — show issues from all users.
	if user != "" && !strings.EqualFold(user, "all") {
		if filter == nil {
			filter = &api.IssueFilter{}
		}
		filter.Assignee = &api.NullableUserFilter{
			DisplayName: &api.StringComparator{EqIgnoreCase: &user},
		}
	}

	// Label filter for --label flag.
	// Comma = OR (bug,devex → bug OR devex), plus = AND (bug+devex → bug AND devex).
	labelLower := strings.ToLower(strings.TrimSpace(labelFilter))
	if labelLower != "" {
		if filter == nil {
			filter = &api.IssueFilter{}
		}
		filter.Labels = buildLabelFilter(labelLower)
	}

	// Cycle filter.
	var resolvedCycle *cycleInfo
	cycleLower := strings.ToLower(strings.TrimSpace(cycle))
	switch {
	case cycleLower == "all":
		// No cycle filter — include issues from all cycles.
	case cycleLower != "":
		// Explicit cycle value: resolve via API.
		if filter == nil {
			filter = &api.IssueFilter{}
		}
		ci, err := resolveCycle(ctx, client, c, timeNow, cycleLower)
		if err != nil {
			return nil, nil, err
		}
		resolvedCycle = &ci
		filter.Cycle = &api.NullableCycleFilter{
			Number: &api.NumberComparator{Eq: &ci.Number},
		}
	default:
		// No --cycle flag: default to current cycle.
		// Resolve via API to show the cycle header; fall back to IsActive
		// filter if the resolution fails (e.g. no active cycle).
		if filter == nil {
			filter = &api.IssueFilter{}
		}
		ci, err := resolveCycle(ctx, client, c, timeNow, "current")
		if err == nil {
			resolvedCycle = &ci
			filter.Cycle = &api.NullableCycleFilter{
				Number: &api.NumberComparator{Eq: &ci.Number},
			}
		} else {
			trueVal := true
			filter.Cycle = &api.NullableCycleFilter{
				IsActive: &api.BooleanComparator{Eq: &trueVal},
			}
		}
	}

	return filter, resolvedCycle, nil
}

// fetchIssueNodes fetches issue nodes using the appropriate query based on the
// user flag. When user is non-empty, ListIssues is used; otherwise ListMyIssues.
func fetchIssueNodes(ctx context.Context, client graphql.Client, user string, limit int, filter *api.IssueFilter) ([]*issueNode, error) {
	if user != "" {
		resp, err := api.ListIssues(ctx, client, limit, nil, filter)
		if err != nil {
			return nil, fmt.Errorf("listing issues: %w", err)
		}
		if resp.Issues == nil {
			return nil, fmt.Errorf("no issues data returned from API")
		}
		var nodes []*issueNode
		for _, n := range resp.Issues.Nodes {
			nodes = append(nodes, convertListIssuesNode(n))
		}
		return nodes, nil
	}
	resp, err := api.ListMyIssues(ctx, client, limit, nil, filter)
	if err != nil {
		return nil, fmt.Errorf("listing issues: %w", err)
	}
	if resp.Viewer == nil {
		return nil, fmt.Errorf("no viewer data returned from API")
	}
	if resp.Viewer.AssignedIssues == nil {
		return nil, fmt.Errorf("no assigned issues data returned from API")
	}
	return resp.Viewer.AssignedIssues.Nodes, nil
}

// convertListIssuesNode converts a ListIssues node to the canonical issueNode type.
func convertListIssuesNode(n *api.ListIssuesIssuesIssueConnectionNodesIssue) *issueNode {
	var labels *api.ListMyIssuesViewerUserAssignedIssuesIssueConnectionNodesIssueLabelsIssueLabelConnection
	if n.Labels != nil {
		convertedNodes := make([]*api.ListMyIssuesViewerUserAssignedIssuesIssueConnectionNodesIssueLabelsIssueLabelConnectionNodesIssueLabel, len(n.Labels.Nodes))
		for i, l := range n.Labels.Nodes {
			convertedNodes[i] = &api.ListMyIssuesViewerUserAssignedIssuesIssueConnectionNodesIssueLabelsIssueLabelConnectionNodesIssueLabel{
				Name: l.Name,
			}
		}
		labels = &api.ListMyIssuesViewerUserAssignedIssuesIssueConnectionNodesIssueLabelsIssueLabelConnection{
			Nodes: convertedNodes,
		}
	}
	return &issueNode{
		Id:         n.Id,
		Identifier: n.Identifier,
		Title:      n.Title,
		State:      (*api.ListMyIssuesViewerUserAssignedIssuesIssueConnectionNodesIssueStateWorkflowState)(n.State),
		Priority:   n.Priority,
		UpdatedAt:  n.UpdatedAt,
		Labels:     labels,
	}
}

type issueNode = api.ListMyIssuesViewerUserAssignedIssuesIssueConnectionNodesIssue

// cycleInfo holds resolved cycle metadata for display and mutations.
type cycleInfo struct {
	Id       string
	Number   float64
	Name     string
	StartsAt string
	EndsAt   string
}

// formatHeader returns a colorized display string like "Cycle 11 - Sprint 11 (Jan 15 – Jan 28)".
func (c cycleInfo) formatHeader(color bool) string {
	header := format.Colorize(color, format.Bold+format.Cyan, fmt.Sprintf("Cycle %.0f", c.Number))
	if c.Name != "" {
		header += " - " + c.Name
	}
	start := formatCycleDate(c.StartsAt)
	end := formatCycleDate(c.EndsAt)
	if start != "" && end != "" {
		header += " " + format.Colorize(color, format.Gray, fmt.Sprintf("(%s – %s)", start, end))
	}
	return header
}

// formatCycleDate extracts "Jan _2" from an ISO timestamp.
// The _2 format pads single-digit days with a leading space for alignment.
func formatCycleDate(ts string) string {
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return ""
	}
	return t.Format("Jan _2")
}

const (
	_cycleCacheKey = "cycles/list"
	_cycleCacheTTL = 24 * time.Hour
)

// cycleBoundaryCrossed reports whether the cached cycle data is stale because
// a cycle boundary has been crossed. The boolean flags (IsActive, IsNext, etc.)
// are point-in-time snapshots from the API, so they become wrong when the
// active cycle ends or a new one starts.
func cycleBoundaryCrossed(resp *api.ListCyclesResponse, now time.Time) bool {
	if resp.Cycles == nil {
		return true
	}
	for _, c := range resp.Cycles.Nodes {
		if !c.IsActive {
			continue
		}
		endsAt, err := time.Parse(time.RFC3339, c.EndsAt)
		if err != nil {
			return true
		}
		return now.After(endsAt)
	}
	// No active cycle in cached data; treat as stale so we re-fetch.
	return true
}

// listCyclesCached returns cycle data, serving from cache when available.
// The cache is invalidated if a cycle boundary has been crossed since the
// data was fetched, even if the TTL has not expired.
func listCyclesCached(ctx context.Context, client graphql.Client, c *cache.Cache, timeNow func() time.Time) (*api.ListCyclesResponse, error) {
	if c != nil {
		if data, ok := c.GetWithTTL(_cycleCacheKey, _cycleCacheTTL); ok {
			var resp api.ListCyclesResponse
			if err := json.Unmarshal([]byte(data), &resp); err == nil {
				if !cycleBoundaryCrossed(&resp, timeNow()) {
					return &resp, nil
				}
			}
		}
	}

	resp, err := api.ListCycles(ctx, client, 50)
	if err != nil {
		return nil, err
	}

	if c != nil {
		if data, err := json.Marshal(resp); err == nil {
			_ = c.Set(_cycleCacheKey, string(data))
		}
	}

	return resp, nil
}

// resolveCycle converts a cycle flag value to cycle info.
// Named values (current, next, previous) are resolved via the ListCycles API;
// numeric strings also query ListCycles to get the full metadata.
func resolveCycle(ctx context.Context, client graphql.Client, c *cache.Cache, timeNow func() time.Time, value string) (cycleInfo, error) {
	isNumeric := false
	var numericVal float64
	if n, err := strconv.ParseFloat(value, 64); err == nil {
		isNumeric = true
		numericVal = n
	} else {
		switch strings.ToLower(value) {
		case "current", "next", "previous":
		default:
			return cycleInfo{}, fmt.Errorf("invalid --cycle value %q: must be current, next, previous, or a number", value)
		}
	}

	resp, err := listCyclesCached(ctx, client, c, timeNow)
	if err != nil {
		return cycleInfo{}, fmt.Errorf("fetching cycles: %w", err)
	}
	if resp.Cycles == nil {
		return cycleInfo{}, fmt.Errorf("no cycles data returned from API")
	}

	for _, c := range resp.Cycles.Nodes {
		if isNumeric {
			if c.Number == numericVal {
				name := ""
				if c.Name != nil {
					name = *c.Name
				}
				return cycleInfo{Id: c.Id, Number: c.Number, Name: name, StartsAt: c.StartsAt, EndsAt: c.EndsAt}, nil
			}
			continue
		}
		matched := false
		switch strings.ToLower(value) {
		case "current":
			matched = c.IsActive
		case "next":
			matched = c.IsNext
		case "previous":
			matched = c.IsPrevious
		}
		if matched {
			name := ""
			if c.Name != nil {
				name = *c.Name
			}
			return cycleInfo{Id: c.Id, Number: c.Number, Name: name, StartsAt: c.StartsAt, EndsAt: c.EndsAt}, nil
		}
	}

	return cycleInfo{}, fmt.Errorf("no %s cycle found", value)
}

func sortIssues(nodes []*issueNode, sortBy string) {
	slices.SortFunc(nodes, func(a, b *issueNode) int {
		switch strings.ToLower(sortBy) {
		case "priority":
			if d := issuePriorityRank(a.Priority) - issuePriorityRank(b.Priority); d != 0 {
				if d < 0 {
					return -1
				}
				return 1
			}
			return issueStateRank(a) - issueStateRank(b)
		case "identifier":
			return strings.Compare(a.Identifier, b.Identifier)
		case "title":
			return strings.Compare(
				strings.ToLower(a.Title),
				strings.ToLower(b.Title),
			)
		default: // "status"
			if d := issueStateRank(a) - issueStateRank(b); d != 0 {
				return d
			}
			// Within same status, sort by priority (most important first)
			da := issuePriorityRank(a.Priority)
			db := issuePriorityRank(b.Priority)
			if da < db {
				return -1
			}
			if da > db {
				return 1
			}
			return 0
		}
	})
}

// buildLabelFilter parses the label flag value into an IssueLabelCollectionFilter.
// Comma separates OR groups, plus separates AND terms within a group.
//
//	"bug,devex"       → bug OR devex
//	"bug+frontend"    → bug AND frontend
//	"bug+frontend,devex" → (bug AND frontend) OR devex
func buildLabelFilter(value string) *api.IssueLabelCollectionFilter {
	labelFilter := func(name string) *api.IssueLabelFilter {
		return &api.IssueLabelFilter{
			Name: &api.StringComparator{EqIgnoreCase: &name},
		}
	}

	// Parse OR groups (comma-separated).
	var orGroups []*api.IssueLabelCollectionFilter
	for group := range strings.SplitSeq(value, ",") {
		group = strings.TrimSpace(group)
		if group == "" {
			continue
		}

		// Parse AND terms within a group (plus-separated).
		var andTerms []*api.IssueLabelCollectionFilter
		for term := range strings.SplitSeq(group, "+") {
			term = strings.TrimSpace(term)
			if term == "" {
				continue
			}
			andTerms = append(andTerms, &api.IssueLabelCollectionFilter{
				Some: labelFilter(term),
			})
		}

		switch len(andTerms) {
		case 0:
			continue
		case 1:
			orGroups = append(orGroups, andTerms[0])
		default:
			orGroups = append(orGroups, &api.IssueLabelCollectionFilter{
				And: andTerms,
			})
		}
	}

	switch len(orGroups) {
	case 0:
		return nil
	case 1:
		return orGroups[0]
	default:
		return &api.IssueLabelCollectionFilter{
			Or: orGroups,
		}
	}
}
