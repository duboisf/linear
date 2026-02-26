# GraphQL Client

## Code Generation with genqlient

The project uses [genqlient](https://github.com/Khan/genqlient) to generate typed Go functions from GraphQL operations. The workflow:

1. Define queries/mutations in `internal/api/genqlient.graphql`
2. Run `make generate` (invokes `go generate` on `client.go`)
3. genqlient reads `internal/api/genqlient.yaml` and produces `internal/api/generated.go`

`generated.go` is ~8000+ lines of auto-generated code and is excluded from test coverage. Never edit it manually.

### Adding a New Query

1. Add the query or mutation to `internal/api/genqlient.graphql`
2. Run `make generate`
3. Use the generated function directly: e.g., `api.ListMyIssues(ctx, client, 50, nil, filter)`

### genqlient.yaml Configuration

Key settings in `internal/api/genqlient.yaml`:

- `schema: ../../schema.graphql` -- Linear's schema file at repo root
- `use_struct_references: true` -- generates pointer types for nested structs
- `optional: pointer` -- optional fields become Go pointers (nil = absent)
- `bindings` -- maps Linear scalars (`DateTime`, `UUID`, `JSON`, etc.) to Go types (mostly `string` or `any`)

## Auth Transport

`internal/api/client.go` defines `authTransport`, which wraps `http.RoundTripper` to inject the `Authorization` header on every request.

```go
func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
    req = req.Clone(req.Context()) // required by RoundTripper contract
    req.Header.Set("Authorization", t.apiKey)
    return t.wrapped.RoundTrip(req)
}
```

**Critical**: `req.Clone()` is mandatory. The `RoundTripper` contract forbids modifying the original request. Removing the clone causes data races in concurrent requests.

`NewClient` creates a production client (30s timeout, default transport). `NewClientWithHTTPClient` accepts a custom `http.Client` for testing.

## Comparator MarshalJSON Gotcha

`internal/api/comparator_json.go` defines custom `MarshalJSON` methods for comparator types (`StringComparator`, `NumberComparator`, `BooleanComparator`, etc.) and `IssueUpdateInput`.

**Why this exists**: genqlient generates comparator structs without `omitempty` JSON tags. When you set only one field (e.g., `Nin`), unset fields serialize as `null`. The Linear API interprets `null` fields as active constraints, causing conflicts (e.g., `{eq: null, nin: ["completed"]}` fails).

The `marshalOmitZero` helper uses reflection to emit only non-zero fields:

```go
func marshalOmitZero(v any) ([]byte, error) {
    // Skips nil pointers, empty slices, zero numbers, empty strings
}
```

For `IssueUpdateInput`, the semantics differ: `nil` pointer = omit from payload, non-nil empty string = explicitly unset the field on the server.

## Key Files

| File | Purpose |
|---|---|
| `internal/api/client.go` | Client factory, auth transport |
| `internal/api/genqlient.yaml` | genqlient config (schema path, bindings) |
| `internal/api/genqlient.graphql` | All GraphQL operations |
| `internal/api/generated.go` | Auto-generated (do not edit) |
| `internal/api/comparator_json.go` | Custom JSON marshaling for comparators |
