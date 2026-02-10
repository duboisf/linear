package cmd_test

import (
	"strings"
	"testing"

	"github.com/duboisf/linear/cmd"
)

const getUserResponse = `{
	"data": {
		"users": {
			"nodes": [
				{
					"id": "user-1",
					"name": "Jane Doe",
					"displayName": "jane",
					"email": "jane@example.com",
					"active": true,
					"admin": false,
					"isMe": false
				}
			]
		}
	}
}`

const getUserNotFoundResponse = `{
	"data": {
		"users": {
			"nodes": []
		}
	}
}`

const getUserNullUsersResponse = `{
	"data": {
		"users": null
	}
}`

func TestUserGet_Success(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, map[string]string{
		"GetUserByDisplayName": getUserResponse,
	})

	opts, stdout, _ := testOptionsWithBuffers(t, server)
	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"user", "get", "jane"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("user get returned error: %v", err)
	}

	output := stdout.String()
	checks := []string{
		"Jane Doe",
		"jane",
		"jane@example.com",
		"Member",
		"Active",
	}
	for _, check := range checks {
		if !strings.Contains(output, check) {
			t.Errorf("output does not contain %q", check)
		}
	}
}

func TestUserGet_NotFound(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, map[string]string{
		"GetUserByDisplayName": getUserNotFoundResponse,
	})

	opts, _, _ := testOptionsWithBuffers(t, server)
	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"user", "get", "nonexistent"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for not found user")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error %q should contain 'not found'", err.Error())
	}
}

func TestUserGet_NullUsers(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, map[string]string{
		"GetUserByDisplayName": getUserNullUsersResponse,
	})

	opts, _, _ := testOptionsWithBuffers(t, server)
	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"user", "get", "jane"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for null users response")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error %q should contain 'not found'", err.Error())
	}
}

func TestUserGet_MissingArgs(t *testing.T) {
	t.Parallel()

	server := newMockGraphQLServer(t, nil)
	opts, _, _ := testOptionsWithBuffers(t, server)
	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"user", "get"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when no args provided")
	}
}

func TestUserGet_Aliases(t *testing.T) {
	t.Parallel()

	aliases := []string{"show", "view"}
	for _, alias := range aliases {
		t.Run(alias, func(t *testing.T) {
			t.Parallel()

			server := newMockGraphQLServer(t, map[string]string{
				"GetUserByDisplayName": getUserResponse,
			})

			opts, stdout, _ := testOptionsWithBuffers(t, server)
			root := cmd.NewRootCmd(opts)
			root.SetArgs([]string{"user", alias, "jane"})

			err := root.Execute()
			if err != nil {
				t.Fatalf("user %s returned error: %v", alias, err)
			}

			output := stdout.String()
			if !strings.Contains(output, "Jane Doe") {
				t.Errorf("alias %q output does not contain 'Jane Doe'", alias)
			}
		})
	}
}

func TestUserGet_ResolveClientError(t *testing.T) {
	t.Parallel()

	opts, _, _ := testOptionsKeyringError(t)
	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"user", "get", "jane"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when resolveClient fails")
	}
	if !strings.Contains(err.Error(), "resolving API key") {
		t.Errorf("error %q should contain 'resolving API key'", err.Error())
	}
}

func TestUserGet_APIError(t *testing.T) {
	t.Parallel()

	server := newErrorGraphQLServer(t)
	opts, _, _ := testOptionsWithBuffers(t, server)
	root := cmd.NewRootCmd(opts)
	root.SetArgs([]string{"user", "get", "jane"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error from API failure")
	}
	if !strings.Contains(err.Error(), "getting user") {
		t.Errorf("error %q should contain 'getting user'", err.Error())
	}
}

func TestUserGet_ValidArgsFunction(t *testing.T) {
	t.Parallel()

	completionResponse := `{
		"data": {
			"users": {
				"nodes": [
					{"id": "u1", "name": "Jane Doe", "displayName": "Jane"},
					{"id": "u2", "name": "John Smith", "displayName": "John"}
				]
			}
		}
	}`

	server := newMockGraphQLServer(t, map[string]string{
		"UsersForCompletion": completionResponse,
	})

	opts := testOptions(t, server)
	root := cmd.NewRootCmd(opts)

	// Find the user get command
	userCmd, _, _ := root.Find([]string{"user"})
	for _, c := range userCmd.Commands() {
		if c.Name() == "get" {
			if c.ValidArgsFunction == nil {
				t.Fatal("get command should have ValidArgsFunction")
			}

			// Call ValidArgsFunction with no args (should return completions)
			completions, directive := c.ValidArgsFunction(c, []string{}, "")
			if directive != 4 { // cobra.ShellCompDirectiveNoFileComp == 4
				t.Errorf("directive = %d, want ShellCompDirectiveNoFileComp (4)", directive)
			}
			if len(completions) != 2 {
				t.Errorf("expected 2 completions, got %d", len(completions))
			}

			// Call with args already present (should return nil)
			completions2, directive2 := c.ValidArgsFunction(c, []string{"jane"}, "")
			if directive2 != 4 {
				t.Errorf("directive = %d, want ShellCompDirectiveNoFileComp (4)", directive2)
			}
			if completions2 != nil {
				t.Errorf("expected nil completions when arg already provided, got %v", completions2)
			}
			return
		}
	}
	t.Fatal("get command not found")
}

func TestUserGet_ValidArgsFunction_ResolveClientError(t *testing.T) {
	t.Parallel()

	opts, _, _ := testOptionsKeyringError(t)
	root := cmd.NewRootCmd(opts)

	userCmd, _, _ := root.Find([]string{"user"})
	for _, c := range userCmd.Commands() {
		if c.Name() == "get" {
			completions, directive := c.ValidArgsFunction(c, []string{}, "")
			if directive != 4 {
				t.Errorf("directive = %d, want ShellCompDirectiveNoFileComp (4)", directive)
			}
			if completions != nil {
				t.Errorf("expected nil completions on resolveClient error, got %v", completions)
			}
			return
		}
	}
	t.Fatal("get command not found")
}

func TestUserGet_ValidArgsFunction_APIError(t *testing.T) {
	t.Parallel()

	server := newErrorGraphQLServer(t)
	opts := testOptions(t, server)
	root := cmd.NewRootCmd(opts)

	userCmd, _, _ := root.Find([]string{"user"})
	for _, c := range userCmd.Commands() {
		if c.Name() == "get" {
			completions, directive := c.ValidArgsFunction(c, []string{}, "")
			if directive != 4 {
				t.Errorf("directive = %d, want ShellCompDirectiveNoFileComp (4)", directive)
			}
			if completions != nil {
				t.Errorf("expected nil completions on API error, got %v", completions)
			}
			return
		}
	}
	t.Fatal("get command not found")
}

func TestUserGet_ValidArgsFunction_NullUsers(t *testing.T) {
	t.Parallel()

	nullUsersResponse := `{
		"data": {
			"users": null
		}
	}`

	server := newMockGraphQLServer(t, map[string]string{
		"UsersForCompletion": nullUsersResponse,
	})

	opts := testOptions(t, server)
	root := cmd.NewRootCmd(opts)

	userCmd, _, _ := root.Find([]string{"user"})
	for _, c := range userCmd.Commands() {
		if c.Name() == "get" {
			completions, directive := c.ValidArgsFunction(c, []string{}, "")
			if directive != 4 {
				t.Errorf("directive = %d, want ShellCompDirectiveNoFileComp (4)", directive)
			}
			if completions != nil {
				t.Errorf("expected nil completions on null users, got %v", completions)
			}
			return
		}
	}
	t.Fatal("get command not found")
}
