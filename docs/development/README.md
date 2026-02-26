# Development

Rules and conventions for developing the Linear CLI.

## Quick Rules

- Every command must have shell completions and tests.
- Tests run with `-race` flag: `go test -race ./...`

## Contents

- [Shell Completions](shell-completions.md) — completion requirements for commands and flags.
- [Testing](testing.md) — test structure, mocking, and conventions.
