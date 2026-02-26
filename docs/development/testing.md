# Testing

- Every command must have corresponding tests in `cmd/<command>_test.go`.
- Tests use an external test package (`package cmd_test`).
- Use the mock GraphQL server from `helpers_test.go` (`newMockGraphQLServer`).
- Use dependency injection via `Options` struct for testability (e.g., `GitWorktreeCreator`).
- All tests must run with `-race` flag.
