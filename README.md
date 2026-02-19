# Linear CLI

A command-line interface for the [Linear](https://linear.app) issue tracker, built with Go.

## Features

- **Interactive browsing** — fuzzy-find issues with live preview powered by [fzf](https://github.com/junegunn/fzf) and [glamour](https://github.com/charmbracelet/glamour)
- **Smart shell completions** — dynamic completions for issue identifiers, users, labels, cycles, and statuses
- **Git worktree integration** — create a worktree from any issue with `issue worktree`
- **Multiple output formats** — plain, markdown, JSON, YAML
- **Advanced filtering** — filter issues by status, label (with AND/OR semantics), cycle, and assignee

## Installation

Requires Go 1.25+.

```bash
# Clone and build
git clone https://github.com/duboisf/linear.git
cd linear
make build

# Or install directly to $GOPATH/bin
make install
```

## Authentication

On first run, the CLI prompts for your [Linear API key](https://linear.app/settings/api) and stores it securely.

**Resolution order** (first match wins):

1. `LINEAR_API_KEY` environment variable
2. OS keyring — `secret-tool` on Linux (libsecret), Keychain on macOS
3. File at `$XDG_CONFIG_HOME/linear/credentials` (`~/.config/linear/credentials`)

If the native keyring tool is not installed, the CLI offers to fall back to file storage (`0600` permissions).

## Usage

### Listing issues

```bash
# List your issues in the current cycle (default)
linear issue list

# Interactive mode with fzf preview
linear issue list --interactive

# Filter by status
linear issue list --status started,todo

# Exclude statuses with ! prefix
linear issue list --status '!completed'

# Show all issues regardless of status
linear issue list --status all

# Filter by label (comma = OR, plus = AND)
linear issue list --label 'bug,devex'         # bug OR devex
linear issue list --label 'bug+frontend'      # bug AND frontend

# Filter by cycle
linear issue list --cycle next
linear issue list --cycle previous
linear issue list --cycle all

# List another user's issues
linear issue list --user alice

# Sort and limit
linear issue list --sort priority --limit 10
```

### Viewing an issue

```bash
# View an issue (opens fzf picker if no identifier given)
linear issue get AIS-42

# Different output formats
linear issue get AIS-42 --output json
linear issue get AIS-42 --output yaml
linear issue get AIS-42 --output markdown
```

### Git worktree integration

Creates a git worktree using the issue's branch name:

```bash
# Create worktree for an issue (opens fzf picker if no identifier given)
linear issue worktree AIS-42
```

The worktree is placed at `<parent-of-repo>/<lowercase-issue-id>/<repo-name>`. For example, if your repo is at `~/git/myrepo`, the worktree for `AIS-42` goes to `~/git/ais-42/myrepo`.

### Users

```bash
# List users
linear user list

# Include bot/integration users
linear user list --include-bots

# View a specific user
linear user get alice
```

## Shell Completions

```bash
# zsh — add to fpath
linear completion zsh > "${fpath[1]}/_linear"

# zsh — source inline
source <(linear completion zsh)

# bash
linear completion bash > /etc/bash_completion.d/linear
# or
source <(linear completion bash)
```

Completions dynamically fetch and cache issue identifiers, user names, labels, cycle numbers, and statuses from the Linear API.

## Caching

Responses are cached to `$XDG_CACHE_HOME/linear/` (typically `~/.cache/linear/`) with a default TTL of 5 minutes. User, label, and cycle data is cached for 24 hours.

```bash
# Clear all cached data
linear cache clear

# Or use --refresh / -r before any command
linear --refresh issue list
```

## Development

```bash
# Build
make build

# Run tests (with race detector)
make test

# Test coverage
make cover

# Regenerate GraphQL client (downloads schema + runs genqlient)
make generate

# Download Linear GraphQL schema only
make schema

# Tidy dependencies
make deps
```
