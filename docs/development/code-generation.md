# Code Generation

The project uses [genqlient](https://github.com/Khan/genqlient) to generate typed Go functions from GraphQL queries against the Linear API schema.

## Key Files

| File | Purpose |
|------|---------|
| `internal/api/genqlient.yaml` | genqlient configuration (schema path, output, type bindings) |
| `internal/api/genqlient.graphql` | GraphQL queries and mutations (source of truth) |
| `internal/api/generated.go` | Generated Go code (~8000 lines). **DO NOT edit manually.** |
| `internal/api/client.go` | Contains the `//go:generate` directive |
| `schema.graphql` | Local copy of Linear's GraphQL schema (downloaded by `make schema`) |

## Workflow

### Full regeneration (recommended)

```bash
make generate
```

This runs two steps:
1. **`make schema`** -- downloads the latest schema from Linear's GitHub repo:
   `https://raw.githubusercontent.com/linear/linear/master/packages/sdk/src/schema.graphql`
2. **`go generate ./...`** -- runs genqlient using the directive in `client.go`:
   `//go:generate go tool genqlient genqlient.yaml`

### Adding a new query

1. Write the query in `internal/api/genqlient.graphql` following existing patterns.
2. Run `make generate` to regenerate `generated.go`.
3. Import and call the new typed function from your command code.

## Configuration Details

The `genqlient.yaml` config sets:
- `use_struct_references: true` -- generated structs are passed by pointer.
- `optional: pointer` -- nullable GraphQL fields become Go pointers.
- `bindings` -- maps Linear scalar types (DateTime, UUID, JSON, etc.) to Go types.

## Tool Dependency

genqlient is declared as a tool dependency in `go.mod`:

```
tool github.com/Khan/genqlient
```

No external install is needed; `go tool genqlient` resolves it automatically.

## Test Coverage

Generated code is excluded from coverage reports. The `make cover` target filters it out:

```makefile
grep -v '/generated\.go:' coverage.raw.out > coverage.out
```
