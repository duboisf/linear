package cmd

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"

	"github.com/Khan/genqlient/graphql"
	"github.com/charmbracelet/glamour"
	"github.com/muesli/termenv"

	"github.com/duboisf/linear/internal/api"
	"github.com/duboisf/linear/internal/cache"
	"github.com/duboisf/linear/internal/format"
)

// fetchMyIssues returns the current user's active issues.
func fetchMyIssues(ctx context.Context, client graphql.Client) ([]issueForCompletion, error) {
	resp, err := api.ActiveIssuesForCompletion(ctx, client, 100)
	if err != nil {
		return nil, err
	}
	if resp.Viewer == nil || resp.Viewer.AssignedIssues == nil {
		return nil, nil
	}

	issues := make([]issueForCompletion, len(resp.Viewer.AssignedIssues.Nodes))
	for i, n := range resp.Viewer.AssignedIssues.Nodes {
		issues[i] = issueForCompletion{
			Identifier: n.Identifier,
			Title:      n.Title,
			Priority:   n.Priority,
		}
		if n.State != nil {
			issues[i].StateName = n.State.Name
			issues[i].StateType = n.State.Type
		}
	}
	return issues, nil
}

// fetchUserIssues returns a specific user's active issues.
func fetchUserIssues(ctx context.Context, client graphql.Client, userName string) ([]issueForCompletion, error) {
	resp, err := api.UserIssuesForCompletion(ctx, client, 100, userName)
	if err != nil {
		return nil, err
	}
	if resp.Issues == nil {
		return nil, nil
	}

	issues := make([]issueForCompletion, len(resp.Issues.Nodes))
	for i, n := range resp.Issues.Nodes {
		issues[i] = issueForCompletion{
			Identifier: n.Identifier,
			Title:      n.Title,
			Priority:   n.Priority,
		}
		if n.State != nil {
			issues[i].StateName = n.State.Name
			issues[i].StateType = n.State.Type
		}
	}
	return issues, nil
}

// fetchIssuesForUser fetches issues for the given user arg ("@my" or a user name).
func fetchIssuesForUser(ctx context.Context, client graphql.Client, userArg string) ([]issueForCompletion, error) {
	if userArg == "@my" {
		return fetchMyIssues(ctx, client)
	}
	return fetchUserIssues(ctx, client, userArg)
}

// formatFzfLines formats issues into aligned, ANSI-colored lines for fzf,
// including a header line. Returns the header and the data lines separately.
func formatFzfLines(issues []issueForCompletion) (header string, lines []string) {
	if len(issues) == 0 {
		return "", nil
	}

	// Include header label widths in column calculations so header and data
	// columns align. The shell completion version skips this because zsh pads
	// the VALUE column independently, but fzf doesn't.
	maxID := len("IDENTIFIER")
	maxState := len("STATUS")
	maxPri := len("PRIORITY")
	for _, issue := range issues {
		if len(issue.Identifier) > maxID {
			maxID = len(issue.Identifier)
		}
		if len(issue.StateName) > maxState {
			maxState = len(issue.StateName)
		}
		if l := len(format.PriorityLabel(issue.Priority)); l > maxPri {
			maxPri = l
		}
	}

	gap := "  "

	header = fmt.Sprintf("%-*s%s%-*s%s%-*s%s%s",
		maxID, "IDENTIFIER", gap,
		maxState, "STATUS", gap,
		maxPri, "PRIORITY", gap,
		"TITLE")

	lines = make([]string, len(issues))
	for i, issue := range issues {
		stateCol := format.PadColor(true, format.StateColor(issue.StateType), issue.StateName, maxState)
		priCol := format.PadColor(true, format.PriorityColor(issue.Priority), format.PriorityLabel(issue.Priority), maxPri)

		lines[i] = fmt.Sprintf("%-*s%s%s%s%s%s%s",
			maxID, issue.Identifier, gap,
			stateCol, gap,
			priCol, gap,
			issue.Title)
	}
	return header, lines
}

// fzfPickIssue presents issues in fzf with aligned columns and returns the
// selected identifier. Returns empty string if the user cancelled (ESC/Ctrl-C).
func fzfPickIssue(issues []issueForCompletion) (string, error) {
	if len(issues) == 0 {
		return "", fmt.Errorf("no issues to select from")
	}

	header, lines := formatFzfLines(issues)

	// Pass header as the first input line with --header-lines=1 so fzf
	// applies the same left margin as data lines (pointer-width aware).
	input := header + "\n" + strings.Join(lines, "\n") + "\n"

	cmd := exec.Command("fzf", "--ansi", "--header-lines=1", "--no-sort", "--layout=reverse")
	cmd.Stdin = strings.NewReader(input)

	var out bytes.Buffer
	cmd.Stdout = &out
	// Let fzf use /dev/tty for interactive input.
	cmd.Stderr = nil

	err := cmd.Run()
	if err != nil {
		// fzf exits 130 on ESC/Ctrl-C.
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 130 {
			return "", nil
		}
		// fzf exits 1 when no match.
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return "", nil
		}
		return "", fmt.Errorf("running fzf: %w", err)
	}

	selected := strings.TrimSpace(out.String())
	if selected == "" {
		return "", nil
	}

	// First field is the identifier.
	fields := strings.Fields(selected)
	if len(fields) == 0 {
		return "", nil
	}
	return fields[0], nil
}

// issuesToCompletions converts issueNode slice to issueForCompletion slice.
func issuesToCompletions(nodes []*issueNode) []issueForCompletion {
	result := make([]issueForCompletion, len(nodes))
	for i, n := range nodes {
		result[i] = issueForCompletion{
			Identifier: n.Identifier,
			Title:      n.Title,
			Priority:   n.Priority,
		}
		if n.State != nil {
			result[i].StateName = n.State.Name
			result[i].StateType = n.State.Type
		}
	}
	return result
}

// _glamourStyle caches the detected terminal background style so that
// termenv.HasDarkBackground (which sends OSC 11 queries) is called at most
// once per process. Concurrent goroutines reuse the cached result.
var (
	_glamourStyle     string
	_glamourStyleOnce sync.Once
)

// glamourStyle returns "dark" or "light" based on the terminal background.
// The detection runs once; subsequent calls return the cached value.
func glamourStyle() string {
	_glamourStyleOnce.Do(func() {
		_glamourStyle = "dark"
		if !termenv.HasDarkBackground() {
			_glamourStyle = "light"
		}
	})
	return _glamourStyle
}

// renderMarkdown renders markdown to ANSI-colored text using glamour.
// It forces TrueColor output regardless of TTY detection so the result
// can be written to a file and later displayed via cat.
func renderMarkdown(markdown string) (string, error) {
	r, err := glamour.NewTermRenderer(
		glamour.WithColorProfile(termenv.TrueColor),
		glamour.WithStandardStyle(glamourStyle()),
		glamour.WithWordWrap(80),
	)
	if err != nil {
		return "", err
	}
	return r.Render(markdown)
}

// formatIssueCache renders an issue as ANSI-colored text for the fzf preview
// cache. It renders markdown through glamour, falling back to the built-in
// ANSI formatter on error.
func formatIssueCache(issue *api.GetIssueIssue) string {
	md := format.FormatIssueDetailMarkdown(issue)
	rendered, err := renderMarkdown(md)
	if err == nil {
		return rendered
	}
	return format.FormatIssueDetail(issue, true)
}

// prefetchIssueDetails fetches full issue details in parallel and writes
// formatted output to the cache. Already-cached issues are skipped.
func prefetchIssueDetails(ctx context.Context, client graphql.Client, c *cache.Cache, identifiers []string) {
	var wg sync.WaitGroup
	sem := make(chan struct{}, 5)
	for _, id := range identifiers {
		if _, ok := c.Get("issues/" + id); ok {
			continue
		}
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			resp, err := api.GetIssue(ctx, client, id)
			if err != nil || resp.Issue == nil {
				return
			}
			content := formatIssueCache(resp.Issue)
			_ = c.Set("issues/"+id, content)
		}(id)
	}
	wg.Wait()
}

// fzfBrowseIssues launches fzf with a preview pane showing full issue details.
// It pre-fetches details for all issues in parallel, renders them as ANSI-colored
// text (via glamour), and writes the result to cache files. fzf's --preview
// uses cat to display the pre-rendered content.
// Returns the selected identifier, or empty string if cancelled.
func fzfBrowseIssues(ctx context.Context, client graphql.Client, issues []issueForCompletion, c *cache.Cache) (string, error) {
	if len(issues) == 0 {
		return "", fmt.Errorf("no issues to browse")
	}

	// Eagerly detect terminal background style before launching goroutines.
	// HasDarkBackground sends an OSC 11 query to the terminal; doing it once
	// here (synchronously, before fzf) avoids concurrent queries whose
	// responses would leak into the terminal as garbage characters.
	_ = glamourStyle()

	// Collect identifiers and pre-fetch details.
	identifiers := make([]string, len(issues))
	for i, iss := range issues {
		identifiers[i] = iss.Identifier
	}
	prefetchIssueDetails(ctx, client, c, identifiers)

	header, lines := formatFzfLines(issues)
	input := header + "\n" + strings.Join(lines, "\n") + "\n"

	// Cache already contains pre-rendered ANSI (either from glow or the
	// built-in formatter), so plain cat is sufficient.
	cacheFile := fmt.Sprintf("%s/issues/{1}", c.Dir)
	previewCmd := fmt.Sprintf("cat '%s'", cacheFile)

	cmd := exec.Command("fzf",
		"--ansi",
		"--header-lines=1",
		"--no-sort",
		"--layout=reverse",
		"--preview", previewCmd,
		"--preview-window", "right:60%:wrap",
	)
	cmd.Stdin = strings.NewReader(input)

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = nil

	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 130 {
			return "", nil
		}
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return "", nil
		}
		return "", fmt.Errorf("running fzf: %w", err)
	}

	selected := strings.TrimSpace(out.String())
	if selected == "" {
		return "", nil
	}

	fields := strings.Fields(selected)
	if len(fields) == 0 {
		return "", nil
	}
	return fields[0], nil
}
