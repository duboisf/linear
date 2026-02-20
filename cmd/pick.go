package cmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"slices"
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

// fetchAllIssues returns active issues from all users.
func fetchAllIssues(ctx context.Context, client graphql.Client) ([]issueForCompletion, error) {
	resp, err := api.AllActiveIssuesForCompletion(ctx, client, 100)
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

// sortCompletionIssues sorts issues by state type (In Progress first, then
// Todo, triage, backlog) with priority as tiebreaker within each state group.
func sortCompletionIssues(issues []issueForCompletion) {
	slices.SortFunc(issues, func(a, b issueForCompletion) int {
		ra, rb := completionStateRank(a.StateType), completionStateRank(b.StateType)
		if d := ra - rb; d != 0 {
			return d
		}
		da, db := issuePriorityRank(a.Priority), issuePriorityRank(b.Priority)
		if da < db {
			return -1
		}
		if da > db {
			return 1
		}
		return 0
	})
}

// completionStateRank returns a sort rank for a state type string.
func completionStateRank(stateType string) int {
	if r, ok := stateTypeOrder[stateType]; ok {
		return r
	}
	return 99
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

// fzfExitOK returns true if the error is an fzf exit that should be treated
// as a non-error (ESC/Ctrl-C = 130, no match = 1).
func fzfExitOK(err error) bool {
	if exitErr, ok := err.(*exec.ExitError); ok {
		code := exitErr.ExitCode()
		return code == 130 || code == 1
	}
	return false
}

// fzfSelectedID extracts the first whitespace-delimited field (the identifier)
// from fzf's output. Returns empty string if output is blank.
func fzfSelectedID(raw string) string {
	selected := strings.TrimSpace(raw)
	if selected == "" {
		return ""
	}
	fields := strings.Fields(selected)
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}

// fzfPickIssue presents issues in fzf with aligned columns and returns the
// selected identifier. Returns empty string if the user cancelled (ESC/Ctrl-C).
func fzfPickIssue(issues []issueForCompletion) (string, error) {
	if len(issues) == 0 {
		return "", fmt.Errorf("no issues to select from")
	}

	sortCompletionIssues(issues)
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

	if err := cmd.Run(); err != nil {
		if fzfExitOK(err) {
			return "", nil
		}
		return "", fmt.Errorf("running fzf: %w", err)
	}

	return fzfSelectedID(out.String()), nil
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

const _glamourStyleEnv = "LINEAR_GLAMOUR_STYLE"

// glamourStyle returns "dark" or "light" based on the terminal background.
// It first checks the LINEAR_GLAMOUR_STYLE environment variable (set by the
// parent process when spawning subprocesses like fzf bindings), falling back
// to terminal background detection. The result is cached per process.
func glamourStyle() string {
	_glamourStyleOnce.Do(func() {
		if v := os.Getenv(_glamourStyleEnv); v == "dark" || v == "light" {
			_glamourStyle = v
			return
		}
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

// refreshIssueCache re-fetches a single issue and updates its cached preview.
func refreshIssueCache(ctx context.Context, client graphql.Client, c *cache.Cache, identifier string) {
	resp, err := api.GetIssue(ctx, client, identifier)
	if err != nil || resp.Issue == nil {
		return
	}
	_ = c.Set("issues/"+identifier, formatIssueCache(resp.Issue))
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
		wg.Go(func() {
			sem <- struct{}{}
			defer func() { <-sem }()
			resp, err := api.GetIssue(ctx, client, id)
			if err != nil || resp.Issue == nil {
				return
			}
			content := formatIssueCache(resp.Issue)
			_ = c.Set("issues/"+id, content)
		})
	}
	wg.Wait()
}

// fzfBrowseIssues launches fzf immediately and fetches issues in the
// background. fzf displays its built-in loading indicator while waiting for
// data to arrive on stdin. Issue detail prefetching also runs concurrently so
// previews populate as the user browses.
// Returns the selected identifier, or empty string if cancelled.
func fzfBrowseIssues(ctx context.Context, client graphql.Client, fetchIssues func(context.Context) ([]issueForCompletion, error), c *cache.Cache, cycleHeader string) (string, error) {
	// Eagerly detect terminal background style before launching goroutines.
	// HasDarkBackground sends an OSC 11 query to the terminal; doing it once
	// here (synchronously, before fzf) avoids concurrent queries whose
	// responses would leak into the terminal as garbage characters.
	_ = glamourStyle()

	// Use a pipe so fzf starts immediately while issues are fetched.
	pr, pw := io.Pipe()
	fetchErrCh := make(chan error, 1)

	fetchCtx, fetchCancel := context.WithCancel(ctx)

	go func() {
		defer pw.Close()

		issues, err := fetchIssues(fetchCtx)
		if err != nil {
			fetchErrCh <- err
			return
		}
		if len(issues) == 0 {
			fetchErrCh <- fmt.Errorf("no issues to browse")
			return
		}

		sortCompletionIssues(issues)

		// Start detail prefetch in background (don't block fzf input).
		identifiers := make([]string, len(issues))
		for i, iss := range issues {
			identifiers[i] = iss.Identifier
		}
		go prefetchIssueDetails(ctx, client, c, identifiers)

		header, lines := formatFzfLines(issues)
		input := header + "\n" + strings.Join(lines, "\n") + "\n"
		_, _ = io.WriteString(pw, input)
		fetchErrCh <- nil
	}()

	// Cache already contains pre-rendered ANSI (either from glamour or the
	// built-in formatter), so plain cat is sufficient.
	self, _ := os.Executable()
	cacheFile := fmt.Sprintf("%s/issues/{1}", c.Dir)
	previewCmd := fmt.Sprintf("cat '%s'", cacheFile)

	// Build ctrl-y binding to set the selected issue's cycle to the current cycle.
	// 1. transform-header: runs the edit and shows the result in the fzf header.
	//    The edit command also invalidates the cached preview for the issue.
	// 2. refresh-preview: forces fzf to re-run the preview command (live fetch).
	helpLine := `ctrl-y: set current cycle  ctrl-d/u: scroll preview  shift-↑/↓: line by line\nenter: select  esc: cancel`
	setCycleBinding := fmt.Sprintf(
		`transform-header(%s issue edit {1} --cycle current 2>&1; echo ""; echo "%s")+refresh-preview`,
		self, helpLine,
	)

	fzfHeader := "ctrl-y: set current cycle  ctrl-d/u: scroll preview  shift-↑/↓: line by line\nenter: select  esc: cancel"
	if cycleHeader != "" {
		fzfHeader = cycleHeader + "\n" + fzfHeader
	}

	cmd := exec.Command("fzf",
		"--ansi",
		"--header-lines=1",
		"--header", fzfHeader,
		"--header-first",
		"--no-sort",
		"--layout=reverse",
		"--preview", previewCmd,
		"--preview-window", "right:60%:wrap",
		"--bind", "ctrl-d:preview-half-page-down,ctrl-u:preview-half-page-up",
		"--bind", "shift-down:preview-down,shift-up:preview-up",
		"--bind", "ctrl-y:"+setCycleBinding,
	)
	cmd.Stdin = pr
	cmd.Env = append(os.Environ(), _glamourStyleEnv+"="+glamourStyle())

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = nil

	runErr := cmd.Run()

	// Cancel the fetch if still running (e.g. user pressed ESC early).
	fetchCancel()

	// Wait for the fetch goroutine to report its result. By this point the
	// goroutine has either completed or will finish quickly via cancellation.
	fetchErr := <-fetchErrCh

	if runErr != nil {
		// Surface fetch errors (e.g. API failure, empty list) over fzf errors.
		if fetchErr != nil && !errors.Is(fetchErr, context.Canceled) {
			return "", fetchErr
		}
		if fzfExitOK(runErr) {
			return "", nil
		}
		return "", fmt.Errorf("running fzf: %w", runErr)
	}

	return fzfSelectedID(out.String()), nil
}
