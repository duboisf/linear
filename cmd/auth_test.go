package cmd_test

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/Khan/genqlient/graphql"

	"github.com/duboisf/linear/cmd"
)

const viewerResponse = `{
	"data": {
		"viewer": {
			"id": "user-1",
			"name": "Fred Dubois",
			"email": "fred@example.com"
		}
	}
}`

const nullViewerResponse = `{
	"data": {
		"viewer": null
	}
}`

// --- auth status tests ---

func TestAuthStatus_Success(t *testing.T) {
	t.Parallel()
	server := newMockGraphQLServer(t, map[string]string{
		"Viewer": viewerResponse,
	})
	opts, stdout, _ := testOptionsWithBuffers(t, server)
	root := cmd.NewRootCmd(opts)
	_, _, err := executeCommand(root, "auth", "status")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := stdout.String()
	if !strings.Contains(got, "Authenticated as Fred Dubois (fred@example.com)") {
		t.Errorf("expected authenticated output, got: %s", got)
	}
}

func TestAuthStatus_NoKey(t *testing.T) {
	t.Parallel()
	opts, _, _ := testOptionsKeyringError(t)
	root := cmd.NewRootCmd(opts)
	_, _, err := executeCommand(root, "auth", "status")
	if err == nil {
		t.Fatal("expected error when no key configured")
	}
	if !strings.Contains(err.Error(), "not authenticated") {
		t.Errorf("expected 'not authenticated' error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "auth setup") {
		t.Errorf("expected error to mention 'auth setup', got: %v", err)
	}
}

func TestAuthStatus_InvalidToken(t *testing.T) {
	t.Parallel()
	server := newErrorGraphQLServer(t)
	opts, _, _ := testOptionsWithBuffers(t, server)
	root := cmd.NewRootCmd(opts)
	_, _, err := executeCommand(root, "auth", "status")
	if err == nil {
		t.Fatal("expected error for invalid token")
	}
	if !strings.Contains(err.Error(), "authentication failed") {
		t.Errorf("expected 'authentication failed' error, got: %v", err)
	}
}

func TestAuthStatus_NullViewer(t *testing.T) {
	t.Parallel()
	server := newMockGraphQLServer(t, map[string]string{
		"Viewer": nullViewerResponse,
	})
	opts, _, _ := testOptionsWithBuffers(t, server)
	root := cmd.NewRootCmd(opts)
	_, _, err := executeCommand(root, "auth", "status")
	if err == nil {
		t.Fatal("expected error for null viewer")
	}
	if !strings.Contains(err.Error(), "authentication failed") {
		t.Errorf("expected 'authentication failed' error, got: %v", err)
	}
}

// --- auth setup tests ---

func TestAuthSetup_Success_NewKey(t *testing.T) {
	t.Parallel()
	server := newMockGraphQLServer(t, map[string]string{
		"Viewer": viewerResponse,
	})
	store := &recordingProvider{getErr: fmt.Errorf("no key")}
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	opts := cmd.Options{
		NewAPIClient: func(apiKey string) graphql.Client {
			return graphql.NewClient(server.URL, server.Client())
		},
		KeyringProvider: &errorProvider{err: fmt.Errorf("no key")},
		KeyReader:       &mockKeyReader{key: "lin_api_test123"},
		NativeStore:     store,
		FileStore:       store,
		Stdin:           strings.NewReader(""),
		Stdout:          stdout,
		Stderr:          stderr,
	}
	root := cmd.NewRootCmd(opts)
	_, _, err := executeCommand(root, "auth", "setup")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := stdout.String()
	if !strings.Contains(got, "Authenticated as Fred Dubois (fred@example.com)") {
		t.Errorf("expected success output, got: %s", got)
	}
	if !strings.Contains(got, "API key saved") {
		t.Errorf("expected 'API key saved' in output, got: %s", got)
	}
	// Should not warn about existing key.
	if strings.Contains(stderr.String(), "already configured") {
		t.Error("should not warn about existing key when none exists")
	}
	// Verify storage was called.
	if !store.storeCalled {
		t.Error("expected StoreAPIKey to be called")
	}
}

func TestAuthSetup_Success_OverrideKey(t *testing.T) {
	t.Parallel()
	server := newMockGraphQLServer(t, map[string]string{
		"Viewer": viewerResponse,
	})
	store := &recordingProvider{getKey: "old-key"}
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	opts := cmd.Options{
		NewAPIClient: func(apiKey string) graphql.Client {
			return graphql.NewClient(server.URL, server.Client())
		},
		KeyringProvider: &staticProvider{key: "old-key"},
		KeyReader:       &mockKeyReader{key: "lin_api_new_key"},
		NativeStore:     store,
		FileStore:       store,
		Stdin:           strings.NewReader(""),
		Stdout:          stdout,
		Stderr:          stderr,
	}
	root := cmd.NewRootCmd(opts)
	_, _, err := executeCommand(root, "auth", "setup")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should warn about overriding.
	if !strings.Contains(stderr.String(), "already configured") {
		t.Error("expected warning about existing key")
	}
	got := stdout.String()
	if !strings.Contains(got, "Authenticated as Fred Dubois") {
		t.Errorf("expected success output, got: %s", got)
	}
}

func TestAuthSetup_InvalidToken_DoesNotStore(t *testing.T) {
	t.Parallel()
	server := newErrorGraphQLServer(t)
	store := &recordingProvider{getErr: fmt.Errorf("no key")}
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	opts := cmd.Options{
		NewAPIClient: func(apiKey string) graphql.Client {
			return graphql.NewClient(server.URL, server.Client())
		},
		KeyringProvider: &errorProvider{err: fmt.Errorf("no key")},
		KeyReader:       &mockKeyReader{key: "bad-token"},
		NativeStore:     store,
		FileStore:       store,
		Stdin:           strings.NewReader(""),
		Stdout:          stdout,
		Stderr:          stderr,
	}
	root := cmd.NewRootCmd(opts)
	_, _, err := executeCommand(root, "auth", "setup")
	if err == nil {
		t.Fatal("expected error for invalid token")
	}
	if !strings.Contains(err.Error(), "token validation failed") {
		t.Errorf("expected 'token validation failed' error, got: %v", err)
	}
	if store.storeCalled {
		t.Error("StoreAPIKey should NOT be called when token validation fails")
	}
}

func TestAuthSetup_NullViewer_DoesNotStore(t *testing.T) {
	t.Parallel()
	server := newMockGraphQLServer(t, map[string]string{
		"Viewer": nullViewerResponse,
	})
	store := &recordingProvider{getErr: fmt.Errorf("no key")}
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	opts := cmd.Options{
		NewAPIClient: func(apiKey string) graphql.Client {
			return graphql.NewClient(server.URL, server.Client())
		},
		KeyringProvider: &errorProvider{err: fmt.Errorf("no key")},
		KeyReader:       &mockKeyReader{key: "some-token"},
		NativeStore:     store,
		FileStore:       store,
		Stdin:           strings.NewReader(""),
		Stdout:          stdout,
		Stderr:          stderr,
	}
	root := cmd.NewRootCmd(opts)
	_, _, err := executeCommand(root, "auth", "setup")
	if err == nil {
		t.Fatal("expected error for null viewer")
	}
	if !strings.Contains(err.Error(), "token validation failed") {
		t.Errorf("expected 'token validation failed' error, got: %v", err)
	}
	if store.storeCalled {
		t.Error("StoreAPIKey should NOT be called when token validation fails")
	}
}

func TestAuthSetup_PromptFails(t *testing.T) {
	t.Parallel()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	opts := cmd.Options{
		NewAPIClient: func(apiKey string) graphql.Client {
			return nil
		},
		KeyringProvider: &errorProvider{err: fmt.Errorf("no key")},
		KeyReader:       &mockKeyReader{err: fmt.Errorf("terminal not available")},
		Stdin:           strings.NewReader(""),
		Stdout:          stdout,
		Stderr:          stderr,
	}
	root := cmd.NewRootCmd(opts)
	_, _, err := executeCommand(root, "auth", "setup")
	if err == nil {
		t.Fatal("expected error when prompt fails")
	}
	if !strings.Contains(err.Error(), "reading API key") {
		t.Errorf("expected 'reading API key' error, got: %v", err)
	}
}
