package config

import (
	"bytes"
	"os"
	"path/filepath"
)

// ExampleContent is the canonical config.example.yaml content.
var ExampleContent = []byte(`# Linear CLI configuration
#
# Commands are available via ctrl-o in interactive mode (linear issue list).
# They receive the selected issue's data as Go template fields.
#
# Set exec: true on a command to exit fzf and replace the process with the
# command (useful for long-running sessions like Claude). Without exec, the
# command runs inside fzf and returns to the issue list when done.
#
# All fields are shell-quoted by default (safe for use in shell commands).
# Use {{.Raw.Field}} to get the unquoted value for display-only contexts.
#
# Available template fields:
#   {{.Identifier}}  - Issue ID (e.g. AIS-123)
#   {{.Title}}       - Issue title
#   {{.Description}} - Markdown description
#   {{.URL}}         - Linear URL
#   {{.BranchName}}  - Suggested git branch
#   {{.State}}       - Workflow state name
#   {{.Priority}}    - Priority label
#   {{.Assignee}}    - Assigned user name
#   {{.Team}}        - Team name
#   {{.TeamKey}}     - Team key prefix
#   {{.Cycle}}       - Cycle name
#   {{.Project}}     - Project name
#   {{.Labels}}      - Label names (slice)
#   {{.DueDate}}     - Due date string
#   {{.Parent}}      - Parent issue identifier

interactive:
  commands:
    # Ask Claude Code to work on the issue (exec: exits fzf first)
    - name: "Claude"
      exec: true
      command: "claude {{.Title}}"

    # Open the issue in your browser
    - name: "Open in browser"
      command: "xdg-open {{.Raw.URL}}"

    # Check out the suggested branch (creates it if it doesn't exist)
    - name: "Git checkout"
      exec: true
      command: "git checkout {{.Raw.BranchName}} 2>/dev/null || git checkout -b {{.Raw.BranchName}}"

    # Create a worktree for isolated work on the issue
    - name: "Git worktree"
      exec: true
      command: "git worktree add ../{{.Raw.BranchName}} -b {{.Raw.BranchName}} 2>/dev/null || git worktree add ../{{.Raw.BranchName}} {{.Raw.BranchName}}"

    # Copy the issue identifier to clipboard (Linux)
    - name: "Copy ID"
      command: "printf '%s' {{.Identifier}} | xclip -selection clipboard && echo Copied {{.Identifier}}"

    # Create a PR linking back to the Linear issue
    - name: "Create PR"
      exec: true
      command: "gh pr create --title {{.Title}} --body {{.URL}}"

    # Start a commit message pre-filled with the issue ID
    - name: "Commit"
      exec: true
      command: "git commit -m {{.Identifier}}:"

    # Show a quick summary in the terminal (pipes into less for paging)
    - name: "Summary"
      command: |
        { printf '%s [%s] %s\n%s\n' {{.Identifier}} {{.State}} {{.Priority}} {{.Title}}
        {{if .Raw.Assignee}}printf 'Assignee: %s\n' {{.Assignee}}
        {{end}}{{if .Raw.DueDate}}printf 'Due: %s\n' {{.DueDate}}
        {{end}}printf '%s\n' {{.URL}}; } | less

    # Dump every template field to verify what data is available
    - name: "Test all template fields"
      command: |
        { printf 'Identifier:  %s\n' {{.Identifier}}
        printf 'Title:       %s\n' {{.Title}}
        printf 'Description: %s\n' {{.Description}}
        printf 'URL:         %s\n' {{.URL}}
        printf 'BranchName:  %s\n' {{.BranchName}}
        printf 'State:       %s\n' {{.State}}
        printf 'Priority:    %s\n' {{.Priority}}
        printf 'Assignee:    %s\n' {{.Assignee}}
        printf 'Team:        %s\n' {{.Team}}
        printf 'TeamKey:     %s\n' {{.TeamKey}}
        printf 'Cycle:       %s\n' {{.Cycle}}
        printf 'Project:     %s\n' {{.Project}}
        printf 'Labels:      %s\n' {{.Labels}}
        printf 'DueDate:     %s\n' {{.DueDate}}
        printf 'Parent:      %s\n' {{.Parent}}; } | less
`)

// DefaultConfigContent is written to config.yaml when it doesn't exist yet.
// Everything is commented out so users start with a blank config.
var DefaultConfigContent = []byte(`# Linear CLI configuration
# Copy settings from config.example.yaml and uncomment to customize.
`)

// EnsureExampleFile writes config.example.yaml to the config directory
// if it is missing or differs from the canonical content.
// configDir overrides directory resolution; nil uses os.UserConfigDir.
func EnsureExampleFile(configDir func() (string, error)) error {
	if configDir == nil {
		configDir = os.UserConfigDir
	}
	dir, err := configDir()
	if err != nil {
		return err
	}

	linearDir := filepath.Join(dir, "linear")
	if err := os.MkdirAll(linearDir, 0o755); err != nil {
		return err
	}

	path := filepath.Join(linearDir, "config.example.yaml")
	existing, err := os.ReadFile(path)
	if err == nil && bytes.Equal(existing, ExampleContent) {
		return nil
	}

	return os.WriteFile(path, ExampleContent, 0o644)
}
