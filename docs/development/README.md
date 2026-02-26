# Development

Rules and conventions for developing the Linear CLI.

## Key Rules

- Every command must have shell completions and tests.
- Tests run with `-race` flag: `go test -race ./...`
- Use external test package (`package cmd_test`) with mocks from `helpers_test.go`.
- Never edit `generated.go` — run `make generate` to regenerate from GraphQL queries.

## Contents

- [Shell Completions](shell-completions.md) — completion requirements for commands and flags.
- [Testing](testing.md) — test structure, mocking, and conventions.
- [Code Generation](code-generation.md) — genqlient workflow and generated code.
- [Adding Commands](adding-commands.md) — step-by-step checklist for new commands.
