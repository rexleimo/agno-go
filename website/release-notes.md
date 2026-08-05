---
title: Release Notes
description: Version history and release notes for HNO
outline: deep
---

# Release Notes

## Version 2.0.0 (2026-08)

### 🎯 The Rewrite

HNO 2.0 is a ground-up rewrite of the framework: Go-native, type-safe, and
architected around a shared orchestration kernel. It is not an incremental
release — every layer was redesigned.

### ✨ Highlights

- **Shared orchestration kernel** (`run.Loop`): agents, teams and workflows
  run on one loop engine. New orchestrators (supervisors, routers, consensus
  drivers) compose in tens of lines.
- **One abstraction, 17 providers**: ten OpenAI-compatible providers share a
  single `OpenAICompat` adapter (thin shells, ~40 lines each); Anthropic and
  Gemini implement native protocols against the same `Model` interface.
- **Type-safe workflows**: `GenericStep[In, Out]` with compile-time input and
  output types (aligned with Genkit Flow[In,Out]).
- **Team as composition**: four collaboration strategies (sequential,
  parallel, leader-follower, consensus) are pluggable `Scheduler`
  implementations on the shared kernel.
- **Function call dispatch optimized**: O(1) merged tool index, concurrent
  tool execution with deterministic ordering, required-parameter validation,
  `execute_tool` OTel spans.
- **Agent Skills**: progressive disclosure (catalog injection → `use_skill`
  tool → on-demand references), aligned with the open Agent Skills standard.
- **First-class MCP**: MCP server tools bridge into standard toolkits with
  server-name prefixes and schema mapping.
- **Observability built in**: OTel GenAI semantic spans (no-op by default),
  cost estimation, retries, rate limiting, circuit breakers.
- **Operations dashboard**: `/ops/ui` shows runtime status, skill catalog and
  eval runs with live refresh.
- **Windows parity**: chroma-go v2 migration removes cgo; the entire test
  suite (79 packages) passes on Windows with zero failures.

### 🔧 Platform Fixes

- chromadb upgraded to v0.4.1 (pure Go, no cgo) with a full v2 API rewrite.
- Path validator handles Windows drive paths and forward-slash relative
  paths correctly; injection protection preserved for bare commands.
- Workflow timestamps are monotonic (CompletedAt strictly after StartedAt);
  memory storage evicts by insertion order (deterministic FIFO).
- External-network tests replaced with local HTTP servers (no flaky CI).

### 💥 Breaking Changes

- Package path renamed: `pkg/agno` → `pkg/hno`; `AgnoError` → `HnoError`.
- `Memory.Add` uses variadic user IDs; `Memory.Size` added.
- Team and Workflow execution internals moved to the shared kernel — custom
  orchestration should implement the `run.Unit` interface.
