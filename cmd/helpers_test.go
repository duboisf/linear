package cmd_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Khan/genqlient/graphql"
	"github.com/spf13/cobra"

	"github.com/duboisf/linear/cmd"
)

// --- Mock keyring provider ---

type staticProvider struct {
	key string
}

func (p *staticProvider) GetAPIKey() (string, error) {
	return p.key, nil
}

func (p *staticProvider) StoreAPIKey(_ string) error {
	return nil
}

// errorProvider always returns an error from GetAPIKey and StoreAPIKey.
type errorProvider struct {
	err error
}

func (p *errorProvider) GetAPIKey() (string, error) {
	return "", p.err
}

func (p *errorProvider) StoreAPIKey(_ string) error {
	return p.err
}

// --- Mock prompter ---

type noopPrompter struct{}

func (p *noopPrompter) PromptForAPIKey(_ io.Reader, _ io.Writer) (string, error) {
	return "test-api-key", nil
}

// errorPrompter always returns an error when prompting.
type errorPrompter struct {
	err error
}

func (p *errorPrompter) PromptForAPIKey(_ io.Reader, _ io.Writer) (string, error) {
	return "", p.err
}

// --- GraphQL mock server ---

// graphqlRequest represents the JSON body of a GraphQL request.
type graphqlRequest struct {
	OperationName string          `json:"operationName"`
	Query         string          `json:"query"`
	Variables     json.RawMessage `json:"variables"`
}

// newMockGraphQLServer creates an httptest.Server that routes based on
// operationName and returns canned responses. The handlers map keys are
// operation names (e.g., "ListMyActiveIssues", "GetIssue").
func newMockGraphQLServer(t *testing.T, handlers map[string]string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "reading body", http.StatusInternalServerError)
			return
		}

		var req graphqlRequest
		if err := json.Unmarshal(bodyBytes, &req); err != nil {
			http.Error(w, "parsing json", http.StatusBadRequest)
			return
		}

		response, ok := handlers[req.OperationName]
		if !ok {
			http.Error(w, "unknown operation: "+req.OperationName, http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(response))
	}))
	t.Cleanup(server.Close)
	return server
}

// newErrorGraphQLServer creates an httptest.Server that always returns an error.
func newErrorGraphQLServer(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"errors":[{"message":"test error"}]}`))
	}))
	t.Cleanup(server.Close)
	return server
}

// testOptions creates cmd.Options wired to the given httptest.Server.
func testOptions(t *testing.T, server *httptest.Server) cmd.Options {
	t.Helper()
	var stdout, stderr bytes.Buffer
	return cmd.Options{
		NewAPIClient: func(apiKey string) graphql.Client {
			return graphql.NewClient(server.URL, server.Client())
		},
		KeyringProvider: &staticProvider{key: "test-api-key"},
		Prompter:        &noopPrompter{},
		NativeStore:     &staticProvider{key: "test-api-key"},
		Stdout:          &stdout,
		Stderr:          &stderr,
	}
}

// testOptionsWithBuffers creates cmd.Options wired to the given httptest.Server,
// returning the stdout and stderr buffers for inspection.
func testOptionsWithBuffers(t *testing.T, server *httptest.Server) (cmd.Options, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	opts := cmd.Options{
		NewAPIClient: func(apiKey string) graphql.Client {
			return graphql.NewClient(server.URL, server.Client())
		},
		KeyringProvider: &staticProvider{key: "test-api-key"},
		Prompter:        &noopPrompter{},
		NativeStore:     &staticProvider{key: "test-api-key"},
		Stdout:          stdout,
		Stderr:          stderr,
	}
	return opts, stdout, stderr
}

// testOptionsKeyringError creates cmd.Options where keyring resolution always
// fails, useful for testing resolveClient error paths.
func testOptionsKeyringError(t *testing.T) (cmd.Options, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	keyErr := fmt.Errorf("no key available")
	opts := cmd.Options{
		NewAPIClient: func(apiKey string) graphql.Client {
			return nil // should never be reached
		},
		KeyringProvider: &errorProvider{err: keyErr},
		Prompter:        &errorPrompter{err: keyErr},
		NativeStore:     &errorProvider{err: keyErr},
		Stdout:          stdout,
		Stderr:          stderr,
	}
	return opts, stdout, stderr
}

// --- Mock GitWorktreeCreator ---

type fetchCall struct {
	remote, branch string
}

type createCall struct {
	path, branch, startPoint string
}

type mockGitWorktreeCreator struct {
	repoRoot    string
	repoRootErr error
	fetchErr    error
	createErr   error
	fetchCalls  []fetchCall
	createCalls []createCall
}

func (m *mockGitWorktreeCreator) RepoRootDir() (string, error) {
	return m.repoRoot, m.repoRootErr
}

func (m *mockGitWorktreeCreator) FetchBranch(remote, branch string) error {
	m.fetchCalls = append(m.fetchCalls, fetchCall{remote, branch})
	return m.fetchErr
}

func (m *mockGitWorktreeCreator) CreateWorktree(path, branch, startPoint string) error {
	m.createCalls = append(m.createCalls, createCall{path, branch, startPoint})
	return m.createErr
}

// --- Shared test fixtures ---

const usersForCompletionResponse = `{
	"data": {
		"users": {
			"nodes": [
				{"id": "u1", "name": "Marc Dupont", "displayName": "Marc Dupont"},
				{"id": "u2", "name": "Jane Smith", "displayName": "Jane Smith"}
			]
		}
	}
}`

// executeCommand executes the given cobra command with args and captures output.
func executeCommand(root *cobra.Command, args ...string) (stdout, stderr string, err error) {
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	root.SetOut(outBuf)
	root.SetErr(errBuf)
	root.SetArgs(args)
	err = root.Execute()
	return outBuf.String(), errBuf.String(), err
}
