# Redis VectorDB Provider (Optional)

This is a minimal Redis-backed `VectorDB` provider for Agno-Go.

- Optional build: compile with `-tags redis`
- No hard dependency when unused
- Naive similarity search in Go (no RediSearch required)

## Build

```bash
# Run tests (skips unless TEST_REDIS_VECTORDB=1)
go test -tags redis ./pkg/hno/vectordb/redisdb -v

# Use in your app
go build -tags redis ./...
```

## Usage

```go
import (
    "context"
    redb "github.com/rexleimo/agno-go/pkg/hno/vectordb/redisdb"
)

ctx := context.Background()
db, _ := redb.New(redb.Config{Addr: "localhost:6379", CollectionName: "docs"})
_ = db.CreateCollection(ctx, "", nil)
// ... Add / Query / Delete
```

## Migration CLI

`cmd/vectordb_migrate` supports Redis when built with the `redis` tag.

```bash
go run -tags redis ./cmd/vectordb_migrate \
  --action up --provider redis --collection mycol --chroma-url localhost:6379
```

Notes:
- `--chroma-url` is reused as Redis address for convenience
- Distance defaults to cosine; use `--distance` to override

## Testing

```bash
export TEST_REDIS_VECTORDB=1
export REDIS_ADDR=localhost:6379

# Redis-tagged tests only
go test -tags redis ./pkg/hno/vectordb/redisdb -v

# Full suite (Redis tests are tag-gated)
go test ./...
```
