## Contributing & Quality Gates

This page explains how to contribute to Agno-Go and which quality checks are expected
before code or docs changes are merged.

### 1. Where to start

- **Read `AGENTS.md`** at the repository root for an overview of:
  - Project structure (`go/`, `docs/`, `specs/`, `scripts/`).  
  - Runtime constraints (pure Go, no runtime Python or cgo bridges).  
  - How specs and fixtures drive behavior.  
- **Review the relevant spec** under `specs/` before implementing a feature:
  - Core AgentOS contracts: `specs/001-go-agno-rewrite/`.  
  - VitePress docs planning: `specs/001-vitepress-docs/`.  

For non-trivial changes, it is strongly recommended to update the corresponding spec,
plan and tasks before touching runtime code.

### 2. Core `make` targets

All quality checks are wired through the root `Makefile`. The most important targets are:

- `make fmt` – format Go code using `gofumpt`.  
- `make lint` – run `golangci-lint` with the configured linters.  
- `make test` – run Go unit and package tests (`go test ./...`).  
- `make providers-test` – run provider integration tests (env-gated).  
- `make coverage` – generate coverage profiles and a summary report.  
- `make bench` – run benchmarks and summarize them with `benchstat`.  
- `make constitution-check` – run the full gate: fmt, lint, test, provider tests,
  coverage, bench and an audit that ensures there is no cgo or Python subprocess usage.  
- `make docs-build` – install docs dependencies (via `pnpm` or `npm`) and build the
  VitePress documentation site from `docs/`.  
- `make docs-check` – run path safety checks on `docs/` (no maintainer absolute paths)
  and then perform a fresh docs build.

Before opening a pull request, you should at least run:

```bash
make fmt lint test docs-check
```

If you are changing provider behavior, contracts or performance-sensitive paths, also
run:

```bash
make providers-test coverage bench constitution-check
```

### 3. Go code expectations

- **Style and layout**
  - Code is formatted by `gofumpt` (via `make fmt`).  
  - Packages should follow existing patterns under `go/internal` and `go/pkg`.  
- **Testing**
  - Every package should have `_test.go` files.  
  - Changes that affect behavior should be covered by unit tests and, where applicable,
    contract tests under `go/tests/contract` or `go/tests/providers`.  
- **No runtime bridges**
  - The Go runtime must not shell out to Python or rely on cgo bridges.  
  - The `make constitution-check` target includes audits that enforce this rule.  

### 4. Docs and specs expectations

- **Specs as the source of truth**
  - When adding or changing features, update the relevant spec under `specs/` and
    regenerate tasks where necessary.  
  - Use the spec to drive changes in Go code and VitePress docs, not the other way around.  

- **Docs alignment**
  - Keep the VitePress docs under `docs/` aligned with:
    - HTTP contracts (`specs/001-go-agno-rewrite/contracts/`).  
    - Provider fixtures and behavior (`specs/001-go-agno-rewrite/contracts/fixtures/`).  
  - Avoid hard-coding maintainer-specific absolute paths in docs; use relative paths
    (for example `./config/default.yaml`) or placeholders.  

- **Multi-language consistency**
  - For core pages (Overview, Quickstart, Core Features & API, Provider Matrix, Advanced
    Guides, Configuration & Security, Contributing & Quality), ensure that:
    - en/zh/ja/ko all have corresponding pages.  
    - Code examples are behaviorally equivalent across languages (only text is localized).  

### 5. What to include in a PR

When opening a pull request, include:

- A short description of the change and which spec/task it addresses.  
- Notes on which `make` targets you ran locally (paste key command outputs if helpful).  
- For docs-heavy changes:
  - Screenshots or a brief summary of the new/updated pages.  
  - Any known gaps (for example, translations that will follow in a separate PR).  

Following these practices helps keep the Go runtime, contracts and VitePress docs aligned
and ensures that new contributions respect the project’s quality gates.
