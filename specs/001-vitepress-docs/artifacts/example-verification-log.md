# Example Verification Log (T045)

**Feature**: `/Users/molei/codes/agno-Go/specs/001-vitepress-docs/spec.md`  
**Branch**: `001-vitepress-docs`

This file tracks execution of the examples marked as “core features” or “advanced
cases” in the documentation. It is intended to support task **T045** and SC-003:
ensuring that key examples are re-verified before each release.

The scenarios are defined primarily in:

- `docs/guide/core-features-and-api.md`  
- `docs/guide/advanced/multi-provider-routing.md`  
- `docs/guide/advanced/knowledge-base-assistant.md`  
- `docs/guide/advanced/memory-chat.md`  
- and their `zh/`, `ja/`, `ko/` equivalents.

> Note: This file currently provides the structure and guidance for recording results.
> Some end-to-end flows may still be blocked by missing runtime features or provider
> credentials; in those cases, record the blocking reason explicitly.

---

## Summary of latest verification round

- **Date**: _(YYYY-MM-DD)_  
- **Triggered by**: _(e.g. pre-release v0.1.0, nightly, manual)_  
- **Environment**:
  - OS / architecture: _(e.g. macOS 15, darwin/arm64)_  
  - Go version: _(e.g. 1.25.1)_  
  - Provider configuration: _(which providers were actually configured / reachable)_  
- **Overall status**:
  - Core features examples: _(e.g. “3/3 completed successfully”)_  
  - Advanced scenarios: _(e.g. “2/3 completed; 1 blocked by missing provider key”)_  

---

## Per-example verification table

Use one row per documented example (or scenario) per verification round.

| Scenario ID | Doc path                                                                 | Locale | Last run date | Status   | Notes / Blocking issues                                                                                 |
|-------------|-------------------------------------------------------------------------|--------|---------------|----------|---------------------------------------------------------------------------------------------------------|
| CF-001      | `docs/guide/core-features-and-api.md`                                  | en     |               |          |                                                                                                         |
| CF-001-zh   | `docs/zh/guide/core-features-and-api.md`                               | zh     |               |          |                                                                                                         |
| CF-001-ja   | `docs/ja/guide/core-features-and-api.md`                               | ja     |               |          |                                                                                                         |
| CF-001-ko   | `docs/ko/guide/core-features-and-api.md`                               | ko     |               |          |                                                                                                         |
| ADV-001     | `docs/guide/advanced/multi-provider-routing.md`                        | en     | 2025-11-22    | PARTIAL  | Flow verified end-to-end using stub provider (no real provider keys configured; responses are echo text) |
| ADV-001-zh  | `docs/zh/guide/advanced/multi-provider-routing.md`                     | zh     |               |          |                                                                                                         |
| ADV-001-ja  | `docs/ja/guide/advanced/multi-provider-routing.md`                     | ja     |               |          |                                                                                                         |
| ADV-001-ko  | `docs/ko/guide/advanced/multi-provider-routing.md`                     | ko     |               |          |                                                                                                         |
| ADV-002     | `docs/guide/advanced/knowledge-base-assistant.md`                      | en     |               |          |                                                                                                         |
| ADV-002-zh  | `docs/zh/guide/advanced/knowledge-base-assistant.md`                   | zh     |               |          |                                                                                                         |
| ADV-002-ja  | `docs/ja/guide/advanced/knowledge-base-assistant.md`                   | ja     |               |          |                                                                                                         |
| ADV-002-ko  | `docs/ko/guide/advanced/knowledge-base-assistant.md`                   | ko     |               |          |                                                                                                         |
| ADV-003     | `docs/guide/advanced/memory-chat.md`                                   | en     |               |          |                                                                                                         |
| ADV-003-zh  | `docs/zh/guide/advanced/memory-chat.md`                                | zh     |               |          |                                                                                                         |
| ADV-003-ja  | `docs/ja/guide/advanced/memory-chat.md`                                | ja     |               |          |                                                                                                         |
| ADV-003-ko  | `docs/ko/guide/advanced/memory-chat.md`                                | ko     |               |          |                                                                                                         |

Add additional rows if new “core” or “advanced” examples are added to the docs.

For each entry, use the **Status** column with one of:

- `OK` – example executed end-to-end and matched expectations.  
- `PARTIAL` – example executed but required workaround(s) or deviated in minor ways.  
- `BLOCKED` – could not be executed (e.g. missing runtime feature, provider key, or
  incompatible API change).  

---

## Relation to other artifacts

- For detailed step-by-step notes on advanced scenarios (including blocked cases),
  continue to use `advanced-scenarios-notes.md`.  
- For Quickstart usability and timing data, use `quickstart-notes.md`.  
- This log is meant as a compact, release-oriented overview of the verification status
  of all key examples referenced in SC-003.
