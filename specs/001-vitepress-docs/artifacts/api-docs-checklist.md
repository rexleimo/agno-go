# API Docs Checklist: Core Features & HTTP Surface

**Feature**: `/Users/molei/codes/agno-Go/specs/001-vitepress-docs/spec.md`  
**Branch**: `001-vitepress-docs`  
**Generated**: Core Features & API / Provider Matrix alignment pass

> Purpose: ensure that the public documentation (Core Features & API + Quickstart) remains consistent with the HTTP surface defined in `contracts/docs-site-openapi.yaml`.

## 1. Endpoint Coverage Matrix

| Endpoint                                      | Method | Described in docs?                  | Docs location                                                                 | Notes |
|-----------------------------------------------|--------|-------------------------------------|-------------------------------------------------------------------------------|-------|
| `/health`                                     | GET    | Yes                                 | `docs/guide/core-features-and-api.md` (+ quickstart health check snippet)    | Shape summarized as “status/version/providers” without enumerating all fields. |
| `/agents`                                     | POST   | Yes                                 | `docs/guide/core-features-and-api.md` (Agent creation section)               | Payload/response described at concept level; exact JSON schema covered by OpenAPI. |
| `/agents/{agentId}`                           | GET    | Yes                                 | `docs/guide/core-features-and-api.md` (Agent retrieval section)              | Error codes (404/400) implied via “invalid ID / not found” wording. |
| `/agents/{agentId}/sessions`                  | POST   | Yes                                 | `docs/guide/core-features-and-api.md` + Quickstart                           | Request fields (`userId`, `metadata`) and response semantics described. |
| `/agents/{agentId}/sessions/{sessionId}/messages` | POST | Yes (non-streaming + streaming)    | `docs/guide/core-features-and-api.md` + Quickstart                           | `stream` query parameter and SSE behavior explained; response fields summarized. |
| `/agents/{agentId}/tools/{toolName}`          | PATCH  | Yes                                 | `docs/guide/core-features-and-api.md` (Tools section)                        | Only high-level payload (`enabled`) and result semantics described. |

No additional public HTTP endpoints are currently documented in `docs-site-openapi.yaml`; the Core Features & API page and Quickstart cover all of them at the level expected for user-facing docs.

## 2. Field & Behavior Consistency Notes

- **Message response**:  
  - OpenAPI describes a structured response containing `messageId`, `content`, `toolCalls`, `usage`, and `state`.  
  - Docs show representative JSON snippets with exactly these fields. Implementation details (for example internal IDs or provider-specific subfields) are intentionally omitted.  

- **Error handling and status codes**:  
  - OpenAPI distinguishes between various error cases (invalid ID, not found, provider unavailable, etc.).  
  - Docs describe errors in conceptual terms（“invalid agentId”、“not found”、“provider unavailable”），and leave exact status codes to the API reference / contracts. This is acceptable for user-facing documentation.  

- **Streaming (`stream=true`)**:  
  - OpenAPI models streaming responses via `207 Multi-Status` and SSE semantics.  
  - Docs describe that `?stream=true` triggers Server-Sent Events with incremental updates, without enumerating the exact event schema. This is sufficient for the advanced guides and Quickstart examples.  

## 3. Deviations & Follow-ups

Current documentation and OpenAPI definitions are conceptually aligned. The following items are noted for future refinement, but are not considered blocking:

- **Detailed error schemas**:  
  - OpenAPI may specify structured error payloads; docs currently describe errors in prose. If future users need programmatic error handling examples, we may add a dedicated “Error handling” subsection.  

- **Provider-specific nuances**:  
  - The main HTTP surface is provider-agnostic; provider differences are documented on the Provider Matrix page rather than per-endpoint. This separation is intentional.  

No blocking discrepancies were found during this pass.

