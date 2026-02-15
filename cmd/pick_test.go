package cmd

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/Khan/genqlient/graphql"

	"github.com/duboisf/linear/internal/api"
	"github.com/duboisf/linear/internal/cache"
)

var _ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

// stripANSI removes ANSI escape sequences from s.
func stripANSI(s string) string {
	return _ansiRe.ReplaceAllString(s, "")
}

func TestFormatFzfLines_Empty(t *testing.T) {
	t.Parallel()

	header, lines := formatFzfLines(nil)
	if header != "" {
		t.Errorf("expected empty header for nil input, got %q", header)
	}
	if lines != nil {
		t.Errorf("expected nil lines for nil input, got %v", lines)
	}
}

func TestFormatFzfLines_Format(t *testing.T) {
	t.Parallel()

	issues := []issueForCompletion{
		{Identifier: "ENG-1", Title: "First issue", StateName: "In Progress", StateType: "started", Priority: 2},
		{Identifier: "ENG-2", Title: "Second issue", StateName: "Todo", StateType: "unstarted", Priority: 3},
	}

	header, lines := formatFzfLines(issues)

	// Header should contain column titles
	for _, col := range []string{"IDENTIFIER", "STATUS", "PRIORITY", "TITLE"} {
		if !strings.Contains(header, col) {
			t.Errorf("header should contain %q, got %q", col, header)
		}
	}

	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}

	// Lines should contain identifiers
	if !strings.Contains(lines[0], "ENG-1") {
		t.Errorf("line 0 should contain ENG-1, got %q", lines[0])
	}
	if !strings.Contains(lines[1], "ENG-2") {
		t.Errorf("line 1 should contain ENG-2, got %q", lines[1])
	}

	// Lines should contain titles
	if !strings.Contains(lines[0], "First issue") {
		t.Errorf("line 0 should contain title, got %q", lines[0])
	}

	// Lines should contain priority labels
	if !strings.Contains(lines[0], "High") {
		t.Errorf("line 0 should contain priority 'High', got %q", lines[0])
	}
	if !strings.Contains(lines[1], "Normal") {
		t.Errorf("line 1 should contain priority 'Normal', got %q", lines[1])
	}
}

func TestFormatFzfLines_Alignment(t *testing.T) {
	t.Parallel()

	issues := []issueForCompletion{
		{Identifier: "ENG-1", Title: "Short", StateName: "In Progress", StateType: "started", Priority: 2},
		{Identifier: "ENG-200", Title: "Longer title", StateName: "Todo", StateType: "unstarted", Priority: 4},
	}

	_, lines := formatFzfLines(issues)

	plain0 := stripANSI(lines[0])
	plain1 := stripANSI(lines[1])

	// Titles should be at the same column position
	titleIdx0 := strings.Index(plain0, "Short")
	titleIdx1 := strings.Index(plain1, "Longer title")
	if titleIdx0 != titleIdx1 {
		t.Errorf("titles should be aligned: 'Short' at %d, 'Longer title' at %d\nline0: %q\nline1: %q",
			titleIdx0, titleIdx1, plain0, plain1)
	}
}

func TestFormatFzfLines_NilState(t *testing.T) {
	t.Parallel()

	issues := []issueForCompletion{
		{Identifier: "ENG-1", Title: "No state issue", Priority: 0},
	}

	header, lines := formatFzfLines(issues)
	if header == "" {
		t.Error("expected non-empty header")
	}
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}
	if !strings.Contains(lines[0], "ENG-1") {
		t.Errorf("line should contain ENG-1, got %q", lines[0])
	}
	if !strings.Contains(lines[0], "None") {
		t.Errorf("line should contain priority 'None', got %q", lines[0])
	}
}

// graphqlRequest is used to parse the incoming GraphQL request body.
type graphqlRequest struct {
	OperationName string `json:"operationName"`
}

func TestFetchUserIssues(t *testing.T) {
	t.Parallel()

	response := `{"data":{"issues":{"nodes":[{"identifier":"ENG-1","title":"Test issue","state":{"name":"Todo","type":"unstarted"},"priority":2}]}}}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(response))
	}))
	defer server.Close()

	client := graphql.NewClient(server.URL, server.Client())
	issues, err := fetchUserIssues(context.Background(), client, "marc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(issues))
	}
	if issues[0].Identifier != "ENG-1" {
		t.Errorf("expected identifier ENG-1, got %q", issues[0].Identifier)
	}
	if issues[0].Title != "Test issue" {
		t.Errorf("expected title 'Test issue', got %q", issues[0].Title)
	}
	if issues[0].StateName != "Todo" {
		t.Errorf("expected state name 'Todo', got %q", issues[0].StateName)
	}
	if issues[0].StateType != "unstarted" {
		t.Errorf("expected state type 'unstarted', got %q", issues[0].StateType)
	}
	if issues[0].Priority != 2 {
		t.Errorf("expected priority 2, got %f", issues[0].Priority)
	}
}

func TestFetchUserIssues_NilIssues(t *testing.T) {
	t.Parallel()

	response := `{"data":{"issues":null}}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(response))
	}))
	defer server.Close()

	client := graphql.NewClient(server.URL, server.Client())
	issues, err := fetchUserIssues(context.Background(), client, "marc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if issues != nil {
		t.Errorf("expected nil issues, got %v", issues)
	}
}

func TestIssuesToCompletions(t *testing.T) {
	t.Parallel()

	nodes := []*issueNode{
		{
			Identifier: "ENG-1",
			Title:      "First",
			Priority:   2,
			State: &api.ListMyIssuesViewerUserAssignedIssuesIssueConnectionNodesIssueStateWorkflowState{
				Name: "In Progress",
				Type: "started",
			},
		},
		{
			Identifier: "ENG-2",
			Title:      "Second",
			Priority:   0,
		},
	}

	result := issuesToCompletions(nodes)

	if len(result) != 2 {
		t.Fatalf("expected 2 completions, got %d", len(result))
	}

	if result[0].Identifier != "ENG-1" {
		t.Errorf("expected ENG-1, got %q", result[0].Identifier)
	}
	if result[0].StateName != "In Progress" {
		t.Errorf("expected state 'In Progress', got %q", result[0].StateName)
	}
	if result[0].StateType != "started" {
		t.Errorf("expected state type 'started', got %q", result[0].StateType)
	}
	if result[0].Priority != 2 {
		t.Errorf("expected priority 2, got %f", result[0].Priority)
	}

	// Nil state should produce empty state fields.
	if result[1].StateName != "" {
		t.Errorf("expected empty state name, got %q", result[1].StateName)
	}
	if result[1].StateType != "" {
		t.Errorf("expected empty state type, got %q", result[1].StateType)
	}
}

func TestPrefetchIssueDetails_CacheHit(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	c := cache.New(dir, 5*time.Minute)

	// Pre-populate cache.
	_ = c.Set("issues/ENG-1", "cached content")

	// Server should NOT be called for cached issues.
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"data":{"issue":null}}`))
	}))
	defer server.Close()

	client := graphql.NewClient(server.URL, server.Client())
	prefetchIssueDetails(context.Background(), client, c, []string{"ENG-1"})

	if called {
		t.Error("expected no API call for cached issue")
	}

	// Verify cache still has the original content.
	got, ok := c.Get("issues/ENG-1")
	if !ok {
		t.Fatal("expected cache hit")
	}
	if got != "cached content" {
		t.Errorf("expected cached content, got %q", got)
	}
}

func TestPrefetchIssueDetails_CacheMiss(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	c := cache.New(dir, 5*time.Minute)

	getIssueResponse := `{
		"data": {
			"issue": {
				"id": "id-1",
				"identifier": "ENG-1",
				"title": "Test issue",
				"state": {"name": "In Progress", "type": "started"},
				"priority": 2,
				"url": "https://linear.app/team/ENG-1",
				"branchName": "eng-1-test-issue"
			}
		}
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(getIssueResponse))
	}))
	defer server.Close()

	client := graphql.NewClient(server.URL, server.Client())
	prefetchIssueDetails(context.Background(), client, c, []string{"ENG-1"})

	// Verify cache was populated.
	got, ok := c.Get("issues/ENG-1")
	if !ok {
		t.Fatal("expected cache to be populated after prefetch")
	}
	plain := stripANSI(got)
	if !strings.Contains(plain, "ENG-1") {
		t.Errorf("cached content should contain identifier, got %q", plain)
	}
	if !strings.Contains(plain, "Test issue") {
		t.Errorf("cached content should contain title, got %q", plain)
	}
}

func TestPrefetchIssueDetails_MultipleMixed(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	c := cache.New(dir, 5*time.Minute)

	// Pre-populate one issue.
	_ = c.Set("issues/ENG-1", "already cached")

	getIssueResponse := `{
		"data": {
			"issue": {
				"id": "id-2",
				"identifier": "ENG-2",
				"title": "New issue",
				"state": {"name": "Todo", "type": "unstarted"},
				"priority": 3,
				"url": "https://linear.app/team/ENG-2",
				"branchName": "eng-2-new-issue"
			}
		}
	}`

	var calledOps []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req graphqlRequest
		json.NewDecoder(r.Body).Decode(&req)
		calledOps = append(calledOps, req.OperationName)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(getIssueResponse))
	}))
	defer server.Close()

	client := graphql.NewClient(server.URL, server.Client())
	prefetchIssueDetails(context.Background(), client, c, []string{"ENG-1", "ENG-2"})

	// Only ENG-2 should have triggered an API call.
	if len(calledOps) != 1 {
		t.Errorf("expected 1 API call, got %d", len(calledOps))
	}

	// ENG-1 should still have original content.
	got, ok := c.Get("issues/ENG-1")
	if !ok || got != "already cached" {
		t.Errorf("ENG-1 cache should be unchanged, got %q", got)
	}

	// ENG-2 should now be cached.
	got2, ok := c.Get("issues/ENG-2")
	if !ok {
		t.Fatal("expected ENG-2 to be cached")
	}
	if !strings.Contains(stripANSI(got2), "ENG-2") {
		t.Errorf("ENG-2 cache should contain identifier, got %q", got2)
	}
}

func TestSortCompletionIssues(t *testing.T) {
	t.Parallel()

	issues := []issueForCompletion{
		{Identifier: "ENG-1", StateType: "unstarted", Priority: 3}, // Todo, Normal
		{Identifier: "ENG-2", StateType: "started", Priority: 4},   // In Progress, Low
		{Identifier: "ENG-3", StateType: "backlog", Priority: 1},   // Backlog, Urgent
		{Identifier: "ENG-4", StateType: "started", Priority: 1},   // In Progress, Urgent
		{Identifier: "ENG-5", StateType: "unstarted", Priority: 2}, // Todo, High
		{Identifier: "ENG-6", StateType: "", Priority: 0},          // Unknown state, None priority
	}

	sortCompletionIssues(issues)

	want := []string{"ENG-4", "ENG-2", "ENG-5", "ENG-1", "ENG-3", "ENG-6"}
	for i, id := range want {
		if issues[i].Identifier != id {
			got := make([]string, len(issues))
			for j := range issues {
				got[j] = issues[j].Identifier
			}
			t.Fatalf("sort order mismatch at index %d:\nwant: %v\ngot:  %v", i, want, got)
		}
	}
}
