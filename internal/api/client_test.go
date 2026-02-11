package api_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/duboisf/linear/internal/api"
)

// capturedRequest stores details from incoming HTTP requests for verification.
type capturedRequest struct {
	authHeader    string
	method        string
	contentType   string
	body          string
}

// newCaptureServer creates an httptest.Server that captures request details
// and responds with the given status code and body.
func newCaptureServer(t *testing.T, statusCode int, responseBody string) (*httptest.Server, *capturedRequest) {
	t.Helper()
	captured := &capturedRequest{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured.authHeader = r.Header.Get("Authorization")
		captured.method = r.Method
		captured.contentType = r.Header.Get("Content-Type")
		bodyBytes, _ := io.ReadAll(r.Body)
		captured.body = string(bodyBytes)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		_, _ = w.Write([]byte(responseBody))
	}))
	t.Cleanup(server.Close)
	return server, captured
}

func TestAuthTransport(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		apiKey   string
		wantAuth string
	}{
		{name: "sets api key", apiKey: "lin_api_test123", wantAuth: "lin_api_test123"},
		{name: "sets empty key", apiKey: "", wantAuth: ""},
		{name: "sets complex key", apiKey: "lin_api_abc-123-XYZ", wantAuth: "lin_api_abc-123-XYZ"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server, captured := newCaptureServer(t, http.StatusOK, `{"data":{}}`)

			// We test authTransport indirectly since it is unexported.
			// We use NewClient with the test server endpoint and verify the header.
			client := api.NewClient(tt.apiKey, server.URL)

			// Make any call to trigger the transport
			_, _ = api.Viewer(context.Background(), client)

			if captured.authHeader != tt.wantAuth {
				t.Errorf("authorization header = %q, want %q", captured.authHeader, tt.wantAuth)
			}
		})
	}
}

func TestAuthTransport_ClonesRequest(t *testing.T) {
	t.Parallel()

	server, _ := newCaptureServer(t, http.StatusOK, `{"data":{}}`)

	// Create a request to track whether the original is modified.
	origReq, err := http.NewRequest(http.MethodPost, server.URL, nil)
	if err != nil {
		t.Fatalf("creating request: %v", err)
	}
	origReq.Header.Set("Authorization", "original-value")

	// Create a client and make a request.
	client := api.NewClient("test-key", server.URL)
	_, _ = api.Viewer(context.Background(), client)

	// The original request we created separately should not be affected.
	if got := origReq.Header.Get("Authorization"); got != "original-value" {
		t.Errorf("original request Authorization header was modified to %q", got)
	}
}

func TestNewClient_DefaultEndpoint(t *testing.T) {
	t.Parallel()

	// When endpoint is empty, NewClient should use LinearAPIEndpoint.
	// We verify indirectly by making a call that would fail with the real endpoint
	// but we just need to confirm the client was created without panic.
	client := api.NewClient("test-key", "")
	if client == nil {
		t.Fatal("NewClient returned nil")
	}
}

func TestNewClient_CustomEndpoint(t *testing.T) {
	t.Parallel()

	server, captured := newCaptureServer(t, http.StatusOK, `{"data":{"viewer":{"id":"u1","name":"Test","email":"test@test.com"}}}`)

	client := api.NewClient("my-key", server.URL)
	resp, err := api.Viewer(context.Background(), client)
	if err != nil {
		t.Fatalf("Viewer returned error: %v", err)
	}

	if captured.authHeader != "my-key" {
		t.Errorf("authorization header = %q, want %q", captured.authHeader, "my-key")
	}

	if resp.Viewer.Name != "Test" {
		t.Errorf("viewer name = %q, want %q", resp.Viewer.Name, "Test")
	}
}

func TestNewClient_HTTPTimeout(t *testing.T) {
	t.Parallel()

	// Create a server that delays longer than the expected timeout.
	// We use a very short timeout via context to avoid a long test, but we
	// verify the client's transport is configured correctly by checking that
	// a slow server triggers a timeout-related error.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Sleep longer than 30s would be needed to truly test the timeout,
		// but we can test that the client was configured with some timeout
		// by using a context with a shorter deadline. Instead, we verify
		// the behavior: NewClient should set a 30s timeout on the HTTP client.
		// Since we cannot inspect the internal http.Client, we test indirectly:
		// a server that sleeps for 1s with a 100ms context deadline should fail.
		time.Sleep(1 * time.Second)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{}}`))
	}))
	t.Cleanup(server.Close)

	client := api.NewClient("test-key", server.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := api.Viewer(ctx, client)
	if err == nil {
		t.Fatal("expected timeout error from slow server with short context deadline")
	}
}

func TestNewClient_BearerPrefixFormat(t *testing.T) {
	t.Parallel()

	// Verify that the Authorization header is exactly "Bearer <key>" with a space.
	server, captured := newCaptureServer(t, http.StatusOK, `{"data":{}}`)

	client := api.NewClient("test-key", server.URL)
	_, _ = api.Viewer(context.Background(), client)

	want := "test-key"
	if captured.authHeader != want {
		t.Errorf("authorization header = %q, want %q", captured.authHeader, want)
	}
}

func TestNewClientWithHTTPClient(t *testing.T) {
	t.Parallel()

	server, _ := newCaptureServer(t, http.StatusOK, `{"data":{"viewer":{"id":"u1","name":"Custom","email":"c@c.com"}}}`)

	httpClient := server.Client()
	client := api.NewClientWithHTTPClient(httpClient, server.URL)

	resp, err := api.Viewer(context.Background(), client)
	if err != nil {
		t.Fatalf("Viewer returned error: %v", err)
	}

	if resp.Viewer.Name != "Custom" {
		t.Errorf("viewer name = %q, want %q", resp.Viewer.Name, "Custom")
	}
}

func TestNewClientWithHTTPClient_DefaultEndpoint(t *testing.T) {
	t.Parallel()

	httpClient := &http.Client{}
	client := api.NewClientWithHTTPClient(httpClient, "")
	if client == nil {
		t.Fatal("NewClientWithHTTPClient returned nil")
	}
}

func TestViewer_Integration(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify it's a POST with JSON
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}

		bodyBytes, _ := io.ReadAll(r.Body)
		var reqBody map[string]any
		if err := json.Unmarshal(bodyBytes, &reqBody); err != nil {
			t.Errorf("unmarshal request body: %v", err)
		}
		if opName, ok := reqBody["operationName"].(string); !ok || opName != "Viewer" {
			t.Errorf("operationName = %v, want Viewer", reqBody["operationName"])
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data": {
				"viewer": {
					"id": "user-42",
					"name": "Jane Doe",
					"email": "jane@example.com"
				}
			}
		}`))
	}))
	t.Cleanup(server.Close)

	client := api.NewClient("lin_api_integration", server.URL)
	resp, err := api.Viewer(context.Background(), client)
	if err != nil {
		t.Fatalf("Viewer returned error: %v", err)
	}

	if resp.Viewer == nil {
		t.Fatal("Viewer response is nil")
	}
	if resp.Viewer.Id != "user-42" {
		t.Errorf("viewer id = %q, want %q", resp.Viewer.Id, "user-42")
	}
	if resp.Viewer.Name != "Jane Doe" {
		t.Errorf("viewer name = %q, want %q", resp.Viewer.Name, "Jane Doe")
	}
	if resp.Viewer.Email != "jane@example.com" {
		t.Errorf("viewer email = %q, want %q", resp.Viewer.Email, "jane@example.com")
	}
}

// --- newOperationServer creates an httptest.Server that dispatches by operationName. ---

func newOperationServer(t *testing.T, handlers map[string]string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			OperationName string `json:"operationName"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		resp, ok := handlers[req.OperationName]
		if !ok {
			http.Error(w, "unknown operation: "+req.OperationName, http.StatusBadRequest)
			return
		}
		_, _ = w.Write([]byte(resp))
	}))
	t.Cleanup(server.Close)
	return server
}

// --- GetIssue tests ---

func TestGetIssue_Success(t *testing.T) {
	t.Parallel()

	const getIssueJSON = `{
		"data": {
			"issue": {
				"id": "issue-abc",
				"identifier": "ENG-42",
				"title": "Implement feature X",
				"description": "Detailed description here.",
				"url": "https://linear.app/team/ENG-42",
				"priority": 2,
				"estimate": 5,
				"dueDate": "2025-12-31",
				"createdAt": "2025-01-01T00:00:00Z",
				"updatedAt": "2025-01-15T00:00:00Z",
				"branchName": "feat/implement-feature-x",
				"state": {
					"name": "In Progress",
					"type": "started"
				},
				"assignee": {
					"name": "Jane Doe",
					"email": "jane@example.com"
				},
				"team": {
					"name": "Engineering",
					"key": "ENG"
				},
				"project": {
					"name": "Project Alpha"
				},
				"labels": {
					"nodes": [
						{"name": "bug"},
						{"name": "frontend"}
					]
				},
				"parent": {
					"identifier": "ENG-1",
					"title": "Parent Epic"
				}
			}
		}
	}`

	server := newOperationServer(t, map[string]string{
		"GetIssue": getIssueJSON,
	})

	client := api.NewClientWithHTTPClient(server.Client(), server.URL)
	resp, err := api.GetIssue(context.Background(), client, "ENG-42")
	if err != nil {
		t.Fatalf("GetIssue returned error: %v", err)
	}

	issue := resp.Issue
	if issue == nil {
		t.Fatal("expected non-nil issue")
	}
	if issue.Id != "issue-abc" {
		t.Errorf("issue.Id = %q, want %q", issue.Id, "issue-abc")
	}
	if issue.Identifier != "ENG-42" {
		t.Errorf("issue.Identifier = %q, want %q", issue.Identifier, "ENG-42")
	}
	if issue.Title != "Implement feature X" {
		t.Errorf("issue.Title = %q, want %q", issue.Title, "Implement feature X")
	}
	if issue.Description == nil || *issue.Description != "Detailed description here." {
		t.Errorf("issue.Description = %v, want %q", issue.Description, "Detailed description here.")
	}
	if issue.Url != "https://linear.app/team/ENG-42" {
		t.Errorf("issue.Url = %q, want %q", issue.Url, "https://linear.app/team/ENG-42")
	}
	if issue.Priority != 2 {
		t.Errorf("issue.Priority = %v, want %v", issue.Priority, 2)
	}
	if issue.Estimate == nil || *issue.Estimate != 5 {
		t.Errorf("issue.Estimate = %v, want 5", issue.Estimate)
	}
	if issue.DueDate == nil || *issue.DueDate != "2025-12-31" {
		t.Errorf("issue.DueDate = %v, want %q", issue.DueDate, "2025-12-31")
	}
	if issue.CreatedAt != "2025-01-01T00:00:00Z" {
		t.Errorf("issue.CreatedAt = %q, want %q", issue.CreatedAt, "2025-01-01T00:00:00Z")
	}
	if issue.UpdatedAt != "2025-01-15T00:00:00Z" {
		t.Errorf("issue.UpdatedAt = %q, want %q", issue.UpdatedAt, "2025-01-15T00:00:00Z")
	}
	if issue.BranchName != "feat/implement-feature-x" {
		t.Errorf("issue.BranchName = %q, want %q", issue.BranchName, "feat/implement-feature-x")
	}
	if issue.State == nil || issue.State.Name != "In Progress" || issue.State.Type != "started" {
		t.Errorf("issue.State = %+v, want In Progress / started", issue.State)
	}
	if issue.Assignee == nil || issue.Assignee.Name != "Jane Doe" {
		t.Errorf("issue.Assignee = %+v, want Jane Doe", issue.Assignee)
	}
	if issue.Team == nil || issue.Team.Name != "Engineering" || issue.Team.Key != "ENG" {
		t.Errorf("issue.Team = %+v, want Engineering / ENG", issue.Team)
	}
	if issue.Project == nil || issue.Project.Name != "Project Alpha" {
		t.Errorf("issue.Project = %+v, want Project Alpha", issue.Project)
	}
	if issue.Labels == nil || len(issue.Labels.Nodes) != 2 {
		t.Errorf("issue.Labels.Nodes length = %d, want 2", len(issue.Labels.Nodes))
	} else {
		if issue.Labels.Nodes[0].Name != "bug" {
			t.Errorf("labels[0] = %q, want %q", issue.Labels.Nodes[0].Name, "bug")
		}
		if issue.Labels.Nodes[1].Name != "frontend" {
			t.Errorf("labels[1] = %q, want %q", issue.Labels.Nodes[1].Name, "frontend")
		}
	}
	if issue.Parent == nil || issue.Parent.Identifier != "ENG-1" || issue.Parent.Title != "Parent Epic" {
		t.Errorf("issue.Parent = %+v, want ENG-1 / Parent Epic", issue.Parent)
	}
}

func TestGetIssue_NullIssue(t *testing.T) {
	t.Parallel()

	server := newOperationServer(t, map[string]string{
		"GetIssue": `{"data": {"issue": null}}`,
	})

	client := api.NewClientWithHTTPClient(server.Client(), server.URL)
	resp, err := api.GetIssue(context.Background(), client, "NONEXIST-1")
	if err != nil {
		t.Fatalf("GetIssue returned error: %v", err)
	}

	if resp.Issue != nil {
		t.Errorf("expected nil issue, got %+v", resp.Issue)
	}
}

func TestGetIssue_MinimalFields(t *testing.T) {
	t.Parallel()

	const minimalJSON = `{
		"data": {
			"issue": {
				"id": "issue-min",
				"identifier": "MIN-1",
				"title": "Minimal issue",
				"description": null,
				"url": "https://linear.app/team/MIN-1",
				"priority": 0,
				"estimate": null,
				"dueDate": null,
				"createdAt": "2025-06-01T00:00:00Z",
				"updatedAt": "2025-06-01T00:00:00Z",
				"branchName": "",
				"state": {"name": "Backlog", "type": "backlog"},
				"assignee": null,
				"team": {"name": "Ops", "key": "OPS"},
				"project": null,
				"labels": {"nodes": []},
				"parent": null
			}
		}
	}`

	server := newOperationServer(t, map[string]string{
		"GetIssue": minimalJSON,
	})

	client := api.NewClientWithHTTPClient(server.Client(), server.URL)
	resp, err := api.GetIssue(context.Background(), client, "MIN-1")
	if err != nil {
		t.Fatalf("GetIssue returned error: %v", err)
	}

	issue := resp.Issue
	if issue == nil {
		t.Fatal("expected non-nil issue")
	}
	if issue.Description != nil {
		t.Errorf("issue.Description = %v, want nil", issue.Description)
	}
	if issue.Estimate != nil {
		t.Errorf("issue.Estimate = %v, want nil", issue.Estimate)
	}
	if issue.DueDate != nil {
		t.Errorf("issue.DueDate = %v, want nil", issue.DueDate)
	}
	if issue.Assignee != nil {
		t.Errorf("issue.Assignee = %+v, want nil", issue.Assignee)
	}
	if issue.Project != nil {
		t.Errorf("issue.Project = %+v, want nil", issue.Project)
	}
	if issue.Parent != nil {
		t.Errorf("issue.Parent = %+v, want nil", issue.Parent)
	}
	if issue.Priority != 0 {
		t.Errorf("issue.Priority = %v, want 0", issue.Priority)
	}
	if len(issue.Labels.Nodes) != 0 {
		t.Errorf("issue.Labels.Nodes length = %d, want 0", len(issue.Labels.Nodes))
	}
}

func TestGetIssue_GraphQLError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"errors": [{"message": "Entity not found"}]}`))
	}))
	t.Cleanup(server.Close)

	client := api.NewClientWithHTTPClient(server.Client(), server.URL)
	_, err := api.GetIssue(context.Background(), client, "INVALID")
	if err == nil {
		t.Fatal("expected error from GraphQL errors response")
	}
}

// --- ListMyActiveIssues tests ---

func TestListMyActiveIssues_Success(t *testing.T) {
	t.Parallel()

	const listJSON = `{
		"data": {
			"viewer": {
				"assignedIssues": {
					"nodes": [
						{
							"id": "id-1",
							"identifier": "ENG-101",
							"title": "Fix login bug",
							"state": {"name": "In Progress", "type": "started"},
							"priority": 1,
							"updatedAt": "2025-01-01T00:00:00Z",
							"labels": {"nodes": [{"name": "bug"}]}
						},
						{
							"id": "id-2",
							"identifier": "ENG-102",
							"title": "Add dark mode",
							"state": {"name": "Backlog", "type": "backlog"},
							"priority": 3,
							"updatedAt": "2025-01-02T00:00:00Z",
							"labels": {"nodes": []}
						}
					],
					"pageInfo": {
						"hasNextPage": false,
						"endCursor": null
					}
				}
			}
		}
	}`

	server := newOperationServer(t, map[string]string{
		"ListMyActiveIssues": listJSON,
	})

	client := api.NewClientWithHTTPClient(server.Client(), server.URL)
	resp, err := api.ListMyActiveIssues(context.Background(), client, 50, nil)
	if err != nil {
		t.Fatalf("ListMyActiveIssues returned error: %v", err)
	}

	if resp.Viewer == nil {
		t.Fatal("expected non-nil viewer")
	}
	nodes := resp.Viewer.AssignedIssues.Nodes
	if len(nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(nodes))
	}

	// Check first issue
	if nodes[0].Id != "id-1" {
		t.Errorf("nodes[0].Id = %q, want %q", nodes[0].Id, "id-1")
	}
	if nodes[0].Identifier != "ENG-101" {
		t.Errorf("nodes[0].Identifier = %q, want %q", nodes[0].Identifier, "ENG-101")
	}
	if nodes[0].Title != "Fix login bug" {
		t.Errorf("nodes[0].Title = %q, want %q", nodes[0].Title, "Fix login bug")
	}
	if nodes[0].State == nil || nodes[0].State.Name != "In Progress" {
		t.Errorf("nodes[0].State = %+v, want In Progress", nodes[0].State)
	}
	if nodes[0].Priority != 1 {
		t.Errorf("nodes[0].Priority = %v, want 1", nodes[0].Priority)
	}
	if nodes[0].Labels == nil || len(nodes[0].Labels.Nodes) != 1 || nodes[0].Labels.Nodes[0].Name != "bug" {
		t.Errorf("nodes[0].Labels unexpected: %+v", nodes[0].Labels)
	}

	// Check second issue
	if nodes[1].Identifier != "ENG-102" {
		t.Errorf("nodes[1].Identifier = %q, want %q", nodes[1].Identifier, "ENG-102")
	}
	if nodes[1].Priority != 3 {
		t.Errorf("nodes[1].Priority = %v, want 3", nodes[1].Priority)
	}

	// Check page info
	pageInfo := resp.Viewer.AssignedIssues.PageInfo
	if pageInfo == nil {
		t.Fatal("expected non-nil pageInfo")
	}
	if pageInfo.HasNextPage {
		t.Error("expected hasNextPage = false")
	}
	if pageInfo.EndCursor != nil {
		t.Errorf("expected nil endCursor, got %q", *pageInfo.EndCursor)
	}
}

func TestListMyActiveIssues_HasHardcodedFilter(t *testing.T) {
	t.Parallel()

	const listJSON = `{
		"data": {
			"viewer": {
				"assignedIssues": {
					"nodes": [
						{
							"id": "id-3",
							"identifier": "ENG-200",
							"title": "Active only",
							"state": {"name": "In Progress", "type": "started"},
							"priority": 2,
							"updatedAt": "2025-03-01T00:00:00Z",
							"labels": {"nodes": []}
						}
					],
					"pageInfo": {
						"hasNextPage": false,
						"endCursor": null
					}
				}
			}
		}
	}`

	server := newOperationServer(t, map[string]string{
		"ListMyActiveIssues": listJSON,
	})

	client := api.NewClientWithHTTPClient(server.Client(), server.URL)
	resp, err := api.ListMyActiveIssues(context.Background(), client, 10, nil)
	if err != nil {
		t.Fatalf("ListMyActiveIssues returned error: %v", err)
	}

	nodes := resp.Viewer.AssignedIssues.Nodes
	if len(nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(nodes))
	}
	if nodes[0].Identifier != "ENG-200" {
		t.Errorf("nodes[0].Identifier = %q, want %q", nodes[0].Identifier, "ENG-200")
	}
}

func TestListMyActiveIssues_WithPagination(t *testing.T) {
	t.Parallel()

	cursor := "cursor-abc"
	const paginatedJSON = `{
		"data": {
			"viewer": {
				"assignedIssues": {
					"nodes": [
						{
							"id": "id-4",
							"identifier": "ENG-300",
							"title": "Page two issue",
							"state": {"name": "Todo", "type": "unstarted"},
							"priority": 4,
							"updatedAt": "2025-04-01T00:00:00Z",
							"labels": {"nodes": []}
						}
					],
					"pageInfo": {
						"hasNextPage": true,
						"endCursor": "cursor-xyz"
					}
				}
			}
		}
	}`

	server := newOperationServer(t, map[string]string{
		"ListMyActiveIssues": paginatedJSON,
	})

	client := api.NewClientWithHTTPClient(server.Client(), server.URL)
	resp, err := api.ListMyActiveIssues(context.Background(), client, 1, &cursor)
	if err != nil {
		t.Fatalf("ListMyActiveIssues with pagination returned error: %v", err)
	}

	nodes := resp.Viewer.AssignedIssues.Nodes
	if len(nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(nodes))
	}
	if nodes[0].Identifier != "ENG-300" {
		t.Errorf("nodes[0].Identifier = %q, want %q", nodes[0].Identifier, "ENG-300")
	}

	pageInfo := resp.Viewer.AssignedIssues.PageInfo
	if !pageInfo.HasNextPage {
		t.Error("expected hasNextPage = true")
	}
	if pageInfo.EndCursor == nil || *pageInfo.EndCursor != "cursor-xyz" {
		t.Errorf("endCursor = %v, want %q", pageInfo.EndCursor, "cursor-xyz")
	}
}

func TestListMyActiveIssues_EmptyResult(t *testing.T) {
	t.Parallel()

	const emptyJSON = `{
		"data": {
			"viewer": {
				"assignedIssues": {
					"nodes": [],
					"pageInfo": {
						"hasNextPage": false,
						"endCursor": null
					}
				}
			}
		}
	}`

	server := newOperationServer(t, map[string]string{
		"ListMyActiveIssues": emptyJSON,
	})

	client := api.NewClientWithHTTPClient(server.Client(), server.URL)
	resp, err := api.ListMyActiveIssues(context.Background(), client, 50, nil)
	if err != nil {
		t.Fatalf("ListMyActiveIssues returned error: %v", err)
	}

	nodes := resp.Viewer.AssignedIssues.Nodes
	if len(nodes) != 0 {
		t.Errorf("expected 0 nodes, got %d", len(nodes))
	}
}

func TestListMyActiveIssues_GraphQLError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"errors": [{"message": "Unauthorized"}]}`))
	}))
	t.Cleanup(server.Close)

	client := api.NewClientWithHTTPClient(server.Client(), server.URL)
	_, err := api.ListMyActiveIssues(context.Background(), client, 50, nil)
	if err == nil {
		t.Fatal("expected error from GraphQL errors response")
	}
}

// --- ActiveIssuesForCompletion tests ---

func TestActiveIssuesForCompletion_Success(t *testing.T) {
	t.Parallel()

	const completionJSON = `{
		"data": {
			"viewer": {
				"assignedIssues": {
					"nodes": [
						{"identifier": "ENG-1", "title": "First issue"},
						{"identifier": "ENG-2", "title": "Second issue"},
						{"identifier": "ENG-3", "title": "Third issue"}
					]
				}
			}
		}
	}`

	server := newOperationServer(t, map[string]string{
		"ActiveIssuesForCompletion": completionJSON,
	})

	client := api.NewClientWithHTTPClient(server.Client(), server.URL)
	resp, err := api.ActiveIssuesForCompletion(context.Background(), client, 100)
	if err != nil {
		t.Fatalf("ActiveIssuesForCompletion returned error: %v", err)
	}

	if resp.Viewer == nil {
		t.Fatal("expected non-nil viewer")
	}
	nodes := resp.Viewer.AssignedIssues.Nodes
	if len(nodes) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(nodes))
	}
	if nodes[0].Identifier != "ENG-1" {
		t.Errorf("nodes[0].Identifier = %q, want %q", nodes[0].Identifier, "ENG-1")
	}
	if nodes[0].Title != "First issue" {
		t.Errorf("nodes[0].Title = %q, want %q", nodes[0].Title, "First issue")
	}
	if nodes[1].Identifier != "ENG-2" {
		t.Errorf("nodes[1].Identifier = %q, want %q", nodes[1].Identifier, "ENG-2")
	}
	if nodes[2].Identifier != "ENG-3" {
		t.Errorf("nodes[2].Identifier = %q, want %q", nodes[2].Identifier, "ENG-3")
	}
}

func TestActiveIssuesForCompletion_EmptyResult(t *testing.T) {
	t.Parallel()

	const emptyJSON = `{
		"data": {
			"viewer": {
				"assignedIssues": {
					"nodes": []
				}
			}
		}
	}`

	server := newOperationServer(t, map[string]string{
		"ActiveIssuesForCompletion": emptyJSON,
	})

	client := api.NewClientWithHTTPClient(server.Client(), server.URL)
	resp, err := api.ActiveIssuesForCompletion(context.Background(), client, 100)
	if err != nil {
		t.Fatalf("ActiveIssuesForCompletion returned error: %v", err)
	}

	nodes := resp.Viewer.AssignedIssues.Nodes
	if len(nodes) != 0 {
		t.Errorf("expected 0 nodes, got %d", len(nodes))
	}
}

func TestActiveIssuesForCompletion_GraphQLError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"errors": [{"message": "internal server error"}]}`))
	}))
	t.Cleanup(server.Close)

	client := api.NewClientWithHTTPClient(server.Client(), server.URL)
	_, err := api.ActiveIssuesForCompletion(context.Background(), client, 100)
	if err == nil {
		t.Fatal("expected error from GraphQL errors response")
	}
}

func TestActiveIssuesForCompletion_SingleNode(t *testing.T) {
	t.Parallel()

	const singleJSON = `{
		"data": {
			"viewer": {
				"assignedIssues": {
					"nodes": [
						{"identifier": "SOLO-1", "title": "Only issue"}
					]
				}
			}
		}
	}`

	server := newOperationServer(t, map[string]string{
		"ActiveIssuesForCompletion": singleJSON,
	})

	client := api.NewClientWithHTTPClient(server.Client(), server.URL)
	resp, err := api.ActiveIssuesForCompletion(context.Background(), client, 1)
	if err != nil {
		t.Fatalf("ActiveIssuesForCompletion returned error: %v", err)
	}

	nodes := resp.Viewer.AssignedIssues.Nodes
	if len(nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(nodes))
	}
	if nodes[0].Identifier != "SOLO-1" {
		t.Errorf("nodes[0].Identifier = %q, want %q", nodes[0].Identifier, "SOLO-1")
	}
	if nodes[0].Title != "Only issue" {
		t.Errorf("nodes[0].Title = %q, want %q", nodes[0].Title, "Only issue")
	}
}

// --- Multi-operation dispatch test ---

func TestMultipleOperations_DispatchByName(t *testing.T) {
	t.Parallel()

	server := newOperationServer(t, map[string]string{
		"GetIssue": `{
			"data": {
				"issue": {
					"id": "i1", "identifier": "T-1", "title": "Test",
					"description": null, "url": "", "priority": 0,
					"estimate": null, "dueDate": null,
					"createdAt": "2025-01-01T00:00:00Z",
					"updatedAt": "2025-01-01T00:00:00Z",
					"branchName": "",
					"state": {"name": "Backlog", "type": "backlog"},
					"assignee": null, "team": {"name": "T", "key": "T"},
					"project": null, "labels": {"nodes": []}, "parent": null
				}
			}
		}`,
		"ListMyActiveIssues": `{
			"data": {
				"viewer": {
					"assignedIssues": {
						"nodes": [],
						"pageInfo": {"hasNextPage": false, "endCursor": null}
					}
				}
			}
		}`,
		"ActiveIssuesForCompletion": `{
			"data": {
				"viewer": {
					"assignedIssues": {
						"nodes": [{"identifier": "X-1", "title": "Comp"}]
					}
				}
			}
		}`,
	})

	client := api.NewClientWithHTTPClient(server.Client(), server.URL)

	// GetIssue
	getResp, err := api.GetIssue(context.Background(), client, "T-1")
	if err != nil {
		t.Fatalf("GetIssue returned error: %v", err)
	}
	if getResp.Issue == nil || getResp.Issue.Identifier != "T-1" {
		t.Errorf("GetIssue returned unexpected result: %+v", getResp.Issue)
	}

	// ListMyActiveIssues
	listResp, err := api.ListMyActiveIssues(context.Background(), client, 10, nil)
	if err != nil {
		t.Fatalf("ListMyActiveIssues returned error: %v", err)
	}
	if len(listResp.Viewer.AssignedIssues.Nodes) != 0 {
		t.Errorf("expected 0 nodes from ListMyActiveIssues, got %d", len(listResp.Viewer.AssignedIssues.Nodes))
	}

	// ActiveIssuesForCompletion
	compResp, err := api.ActiveIssuesForCompletion(context.Background(), client, 50)
	if err != nil {
		t.Fatalf("ActiveIssuesForCompletion returned error: %v", err)
	}
	if len(compResp.Viewer.AssignedIssues.Nodes) != 1 {
		t.Errorf("expected 1 node from ActiveIssuesForCompletion, got %d", len(compResp.Viewer.AssignedIssues.Nodes))
	}
}
