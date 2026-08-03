# Repository Guidelines

## Project Structure & Module Organization
- `pkg/agno` hosts agents, models, toolkits, and shared business logic. HTTP + server runtime lives under `pkg/agentos`.
- CLI and server binaries reside in `cmd/`; each example (for instance `cmd/examples/evolink_media`) includes its own `main.go`.
- Shared services live in `internal/` (e.g., `internal/http`, `internal/session`, `internal/storage`). Tests sit beside the code they cover.
- Docs live in `docs/`, the VitePress site in `website/`, deployment assets under `deploy/`.
- Planning & design docs live in `docs/design/` (see `docs/design/go-agent-framework-design.md` for the v2 framework design). Research reports live in `docs/research/`.

## Build, Test, and Development Commands
- `go build ./...` — compile every package; run this before sending patches.
- `go test ./...` or `make test` — executes the full race-enabled suite and writes `coverage.txt`.
- `make fmt` then `make vet` — format imports and lint for common Go issues.
- `make lint` — runs `golangci-lint` (install via `make install-tools`).
- `make build` / `make clean` — produce binaries into `bin/` or wipe artifacts.
- Docs: `npm install && npm run docs:dev` (or `docs:build`) inside `website/`.

## Coding Style & Naming Conventions
- Follow idiomatic Go: tabs, `UpperCamelCase` exports, `lowerCamelCase` locals, constructors named `NewX`, configs suffixed `Config`, interfaces ending in `er`.
- Document exported symbols with GoDoc comments. Wrap errors using `fmt.Errorf("context: %w", err)` and keep messages lowercase.
- No inline comments unless clarifying novel logic. Use `make fmt` (gofmt + goimports) before pushing.

## Testing Guidelines
- Target ≥70 % coverage in core packages; add regression tests for every behavior change.
- Keep tests table-driven and colocated (`*_test.go`). Use stable doubles for external providers/toolkits.
- Run `make contract-test` when touching Go↔Python parity layers and `make coverage` to review HTML reports if coverage changes materially.

## Commit & Pull Request Guidelines
- Use Conventional Commits (e.g., `feat(agentos): add workflow summaries`, `fix(pkg/tools): correct Tavily timeout`).
- Keep commits scoped and self-testing; rebase or squash WIP series before raising a PR.
- PRs must describe intent, link issues (`Fixes #123`), and paste recent `make test`, `make lint`, or docs build output. Include screenshots or CLI transcripts when altering UX/CLI flows.

## Security & Configuration Tips
- Never commit secrets. Provide `OPENAI_API_KEY`, `ANTHROPIC_API_KEY`, EvoLink keys, DSNs, etc., via your shell or `.envrc`.
- Docker compose reads env vars at runtime—prefer `.env` overrides instead of editing manifests. Redact payloads in shared logs and rotate keys after demos or load tests.
