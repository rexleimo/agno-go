# Release Notes — v1.2.9 (2025-11-14)

## ✨ Highlights
- **EvoLink provider for media agents** – Added first-class EvoLink support under `pkg/hno/providers/evolink` and `pkg/hno/models/evolink/*` for text, image, and video generations, with async task polling and typed option structs matching EvoLink constraints.
- **EvoLink Media Agents docs** – New example pages (`website/examples/evolink-media-agents.md`, `website/zh/examples/evolink-media-agents.md`) showing how to wire EvoLink models into agents and workflows, including environment variables, model tables, and end-to-end pipelines.
- **Knowledge upload chunking** – `POST /api/v1/knowledge/content` now accepts `chunk_size` and `chunk_overlap` across JSON, `text/plain` (query params), and multipart uploads, and records these values in chunk metadata together with `chunker_type`.
- **AgentOS HTTP tips** – Documentation now explains how to customize health endpoints, rely on `/openapi.yaml` and `/docs`, and when to call `server.Resync()` after router changes.

## 🔧 Improvements
- Knowledge handlers propagate chunking options to the underlying readers/chunkers and persist `chunk_size`, `chunk_overlap`, and `chunker_type` per stored chunk to mirror Python AgentOS responses.
- AgentOS API docs (`website/api/agentos.md` and localized variants) are updated to be the canonical reference for Knowledge chunking and HTTP surface configuration, reducing reliance on source-level inspection.

## 🧪 Tests
- EvoLink providers include new tests for image and video configurations (aspect ratios, durations, references, webhooks) to catch misconfigurations early.
- Knowledge API tests cover custom chunk sizes/overlaps, metadata propagation, and regression against existing search/config endpoints.

## ✅ Compatibility
- Additive release; no breaking changes to public Go or HTTP APIs.
- Knowledge chunking parameters are optional and default to prior behavior when omitted.

## 📚 Links
- Website Release Notes: /release-notes#version-129-2025-11-14
- Changelog: `CHANGELOG.md`

