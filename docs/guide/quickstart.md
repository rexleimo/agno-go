# Quickstart: Basics (Go runtime)

Scope: Go rewrite for docs.agno.com/basics, running directly from the Go module at `./go`. Make targets run from repo root; runtime commands run inside `go/`. Go version: 1.25.1.

## 1) Layout & environment

From repo root:
```bash
cp .env.example .env
```
Enter Go module root:
```bash
cd go
export GOCACHE=$PWD/../.cache/go-build   # optional
```
- `AGNO_API_KEY` required (`X-API-Key` only; FR-004).
- Provider vars (`*_API_KEY`, `*_CHAT_MODEL`, `*_EMBED_MODEL`) optional; missing keys mark providers as “not configured” and demos/tests skip with reasons.
- Config: `../config/default.yaml` (router timeouts/retries/concurrency; memory store `memory|bolt|badger`; provider endpoints from env).

## 2) Run Basics scenarios (CLI + fixtures)

Choose a scenario:
```bash
cd go
go run ./cmd/agno --config ../config/default.yaml \
  --scenario <basic|memory|rag|tool|workflow> \
  --fixtures ../specs/001-agno-agents-refactor/contracts/fixtures
```
- `basic`: single-turn agent, stub replay.
- `memory`: multi-turn with MemoryStore history.
- `rag`: retriever down → hint/error fallback.
- `tool`: tool + HITL + guardrail (stream events).
- `workflow`: branching workflow placeholder with multimodal hooks reserved.

## 3) Tests & demos

- Contract (5 scenarios, target ≥95%):
  ```bash
  cd go
  go test ./tests/contract -run Basics
  ```
- Provider demo (env-gated; skips logged):
  ```bash
  cd go
  go run ./cmd/agno --demo --providers openai,gemini \
    --parallel --providers-log ../specs/001-agno-agents-refactor/artifacts/coverage/providers.log
  ```
- Full regression + docs build (repo root):
  ```bash
  make constitution-check
  ```
  Artifacts: `specs/001-agno-agents-refactor/artifacts/coverage.txt`, `bench.txt`, `coverage/providers.log`.

## 4) Provider client examples (Go)

Use the clients under `go/pkg/providers/<provider>`. Example (OpenAI):
```go
package main

import (
  "context"
  "log"
  "os"
  "time"

  "github.com/rexleimo/agno-go/internal/agent"
  "github.com/rexleimo/agno-go/internal/model"
  "github.com/rexleimo/agno-go/pkg/providers/openai"
)

func main() {
  ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
  defer cancel()
  apiKey := os.Getenv("OPENAI_API_KEY")
  if apiKey == "" {
    log.Fatal("OPENAI_API_KEY not set")
  }
  client := openai.New("", apiKey, nil)
  resp, err := client.Chat(ctx, model.ChatRequest{
    Model: agent.ModelConfig{Provider: agent.ProviderOpenAI, ModelID: "gpt-4o-mini", Stream: false},
    Messages: []agent.Message{{Role: agent.RoleUser, Content: "Introduce Agno-Go briefly."}},
  })
  if err != nil {
    log.Fatalf("chat error: %v", err)
  }
  log.Println("assistant:", resp.Message.Content)
}
```
Apply the same shape for other providers (`pkg/providers/<provider>`), honoring each provider’s required env keys.

## 5) Fallbacks, guardrails, auth

- RAG fallback: `knowledge.Engine` hint mode returns guidance without writing history; error mode returns `ErrUnavailable` without state pollution.
- Guardrails: PII / prompt-injection → `ErrGuardrailViolation`, history untouched (see `go/tests/contract/security_contract_test.go`).
- Auth (FR-004): only `X-API-Key`; Basic/Bearer/OAuth/custom rejected (see `go/tests/contract/auth_middleware_negative_test.go`).

## 6) Provider skip & deviations

- Missing keys/unreachable endpoints are logged as skipped in `specs/001-agno-agents-refactor/artifacts/coverage/providers.log`.
- Known gaps vs Python (fixtures are shape-only for multimodal/stream) in `specs/001-agno-agents-refactor/contracts/deviations.md`.
- Benchmarks: compare to `specs/001-agno-agents-refactor/artifacts/baseline/python-bench.json` when present; current Go samples in `specs/001-agno-agents-refactor/artifacts/bench.txt` (targets: p95 -20%, peak mem -25%).

## 7) Build docs (VitePress)

From repo root:
```bash
cd docs
npm run docs:build    # npm install first if needed
# or:
DOCS_DIR=$(pwd)/docs make docs
```
Output: `docs/.vitepress/dist`; Make logs at `specs/001-agno-agents-refactor/artifacts/logs/docs.log`.
