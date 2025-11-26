# Data Model - 001-agno-agents-refactor

## Entities

- **Agent**  
  - Fields: id, name, instructions, model_ref, toolset_refs, memory_policy, hooks, guardrails, multimodal_capabilities, debug_level.  
  - Relationships: may belong to a **Team**; uses **Tools/MCP Servers**; persists **Session/ChatHistory**; reads/writes **MemoryStore**; calls **Knowledge Retriever**.

- **Team**  
  - Fields: id, name, members (agent_ids), routing/coordination strategy, shared_context_policy, shared_toolset_refs.  
  - Relationships: orchestrates multiple **Agents**; shares **Sessions/State/Memory** per policy.

- **Session / State**  
  - Fields: session_id, user_id, agent_id/team_id, state_blob, last_n_messages, created_at, updated_at, ttl.  
  - Relationships: owns **ChatHistory**, links to **Memory** items, influences tool selection and context.

- **ChatHistory**  
  - Fields: session_id, messages[], tool_calls[], streaming_events[], retention_policy.  
  - Relationships: bound to **Session**; summarized into **Memory**.

- **MemoryStore / Memory Item**  
  - Fields: memory_id, owner (agent/team/user), type (short/long/episodic), embedding_ref, payload, metadata (timestamp, score), vector_db_ref (optional).  
  - Relationships: indexed via **Retriever**; attached to **Session** and **Knowledge Base** as needed.

- **Knowledge Base / Retriever**  
  - Fields: kb_id, content_source (ingestion spec), chunking_strategy, embedder_ref, vector_db_ref, filters.  
  - Relationships: used by **Agents/Teams** for RAG; consumes **Embedders/Models**; outputs to **Tools** when agentic filtering is enabled.

- **Tool / MCP Server**  
  - Fields: tool_id, name, signature/schema, transport (local/stdio/sse/http), auth_params, rate_limits, selection_strategy.  
  - Relationships: callable by **Agents/Teams**; may depend on **Dependencies**; participates in **HITL** and guardrails.

- **Model Provider / Embedder**  
  - Fields: provider, model_name, mode (chat/embedding/reasoning/stream), limits (tokens/rate), endpoint, auth_vars, fallback_priority.  
  - Relationships: invoked by **Agent/Team** runtime; referenced by **Knowledge Base** and **Reasoning** paths.

- **Workflow / Step / Hook**  
  - Fields: workflow_id, steps[], conditions, hooks (pre/post), error_handlers, async_flags.  
  - Relationships: executed by **Agent/Team**; uses **Session/State**; may call **Tools** and **Models**.

- **Telemetry / Metrics**  
  - Fields: run_id, session_id, timestamps (start/end), latencies (model/tool), token_usage, errors, guardrail_events.  
  - Relationships: emitted by runtime for observability and benchmarks.

## Constraints & Rules

- Sessions must be uniquely identified (session_id) and isolate user context; sharing requires explicit policy on teams.  
- Memory items require ownership metadata to avoid cross-user leakage; retention and TTL must be enforced per policy.  
- Tool calls must record schema version and auth method; selection must respect per-run tool-call limits.  
- Provider configs must specify capability (chat/stream/embedding) and fail with typed errors; missing env results in skip with logged reason.  
- Knowledge ingestion must record chunking and embedder versions for reproducibility; retrieval must support filters declared in Basics docs.  
- Telemetry must capture latencies and resource usage sufficient for p95/peak measurements.
