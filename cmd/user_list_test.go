package cmd_test

import (
	"strings"
	"testing"

	"github.com/duboisf/linear/cmd"
)

const listUsersResponse = `{
	"data": {
		"users": {
			"nodes": [
				{
					"id": "u1",
					"name": "Jane Doe",
					"displayName": "Jane Doe",
					"email": "jane@example.com",
					"active": true,
					"admin": true,
					"isMe": false
				},
				{
					"id": "u2",
					"name": "John Smith",
					"displayName": "John Smith",
					"email": "john@example.com",
					"active": true,
					"admin": false,
					"isMe": false
				}
			],
			"pageInfo": {"hasNextPage": false, "endCursor": null}
		}
	}
}`

const emptyUsersResponse = `{
	"data": {
		"users": {
			"nodes": [],
			"pageInfo": {"hasNextPage": false, "endCursor": null}
		}
	}
}`

func TestUserList_Default(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, map[string]string{
		"ListUsers": listUsersResponse,
	})

	opts, stdout, _ := testOptionsWithBuffers(t, server)
	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"user", "list"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("user list returned error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "Jane Doe") {
		t.Error("expected output to contain Jane Doe")
	}
	if !strings.Contains(output, "John Smith") {
		t.Error("expected output to contain John Smith")
	}
	if !strings.Contains(output, "jane@example.com") {
		t.Error("expected output to contain jane@example.com")
	}
	if !strings.Contains(output, "Admin") {
		t.Error("expected output to contain Admin")
	}
	if !strings.Contains(output, "Member") {
		t.Error("expected output to contain Member")
	}
}

func TestUserList_EmptyResult(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, map[string]string{
		"ListUsers": emptyUsersResponse,
	})

	opts, stdout, _ := testOptionsWithBuffers(t, server)
	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"user", "list"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("user list with empty result returned error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "NAME") {
		t.Error("expected header in output even with no users")
	}
}

func TestUserList_APIError(t *testing.T) {
	t.Parallel()

	server := newErrorGraphQLServer(t)
	opts, _, _ := testOptionsWithBuffers(t, server)
	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"user", "list"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error from API failure")
	}
	if !strings.Contains(err.Error(), "listing users") {
		t.Errorf("error %q should contain 'listing users'", err.Error())
	}
}

func TestUserList_ResolveClientError(t *testing.T) {
	t.Parallel()

	opts, _, _ := testOptionsKeyringError(t)
	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"user", "list"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when resolveClient fails")
	}
	if !strings.Contains(err.Error(), "resolving API key") {
		t.Errorf("error %q should contain 'resolving API key'", err.Error())
	}
}

func TestUserList_LimitZero(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, nil)
	opts, _, _ := testOptionsWithBuffers(t, server)
	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"user", "list", "--limit", "0"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for --limit 0")
	}
	if !strings.Contains(err.Error(), "--limit must be greater than 0") {
		t.Errorf("error %q should contain '--limit must be greater than 0'", err.Error())
	}
}

func TestUserList_LimitNegative(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, nil)
	opts, _, _ := testOptionsWithBuffers(t, server)
	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"user", "list", "--limit", "-5"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for negative --limit")
	}
	if !strings.Contains(err.Error(), "--limit must be greater than 0") {
		t.Errorf("error %q should contain '--limit must be greater than 0'", err.Error())
	}
}

func TestUserList_Alias(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, map[string]string{
		"ListUsers": listUsersResponse,
	})

	opts, stdout, _ := testOptionsWithBuffers(t, server)
	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"user", "ls"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("user ls alias returned error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "Jane Doe") {
		t.Error("expected output from ls alias")
	}
}

func TestUserList_NilUsers(t *testing.T) {
	t.Parallel()

	nullUsersResponse := `{
		"data": {
			"users": null
		}
	}`

	server := newMockGraphQLServer(t, map[string]string{
		"ListUsers": nullUsersResponse,
	})

	opts, _, _ := testOptionsWithBuffers(t, server)
	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"user", "list"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for nil users")
	}
	if !strings.Contains(err.Error(), "no users data") {
		t.Errorf("error %q should contain 'no users data'", err.Error())
	}
}
