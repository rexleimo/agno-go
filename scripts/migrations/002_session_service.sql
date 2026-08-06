-- Schema for cmd/agentos-session and internal/session/store/postgres.
CREATE TABLE IF NOT EXISTS agno_sessions (
    session_id TEXT PRIMARY KEY,
    session_type TEXT NOT NULL,
    agent_id TEXT,
    team_id TEXT,
    workflow_id TEXT,
    user_id TEXT,
    session_data JSONB,
    agent_data JSONB,
    team_data JSONB,
    workflow_data JSONB,
    metadata JSONB,
    runs JSONB,
    summary JSONB,
    created_at BIGINT NOT NULL,
    updated_at BIGINT
);

CREATE INDEX IF NOT EXISTS idx_agno_sessions_type
    ON agno_sessions(session_type);
CREATE INDEX IF NOT EXISTS idx_agno_sessions_created_at
    ON agno_sessions(created_at);
