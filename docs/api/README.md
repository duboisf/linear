# API

GraphQL client, code generation, and caching patterns.

## Key Rules

- Never edit `internal/api/generated.go` — run `make generate` instead.
- Comparator types use `marshalOmitZero` — never send null fields to the Linear API (they conflict).
- `req.Clone()` is mandatory in `authTransport` (RoundTripper contract forbids mutating the original).
- Cache writes are atomic (temp file + rename) — safe for concurrent reads.
- After editing an issue, call `refreshIssueCache` to update the preview cache.

## Contents

- [GraphQL Client](graphql-client.md) — genqlient setup, auth transport, and comparator gotchas.
- [Caching](caching.md) — file-based cache, TTL, atomic writes, and invalidation strategies.
