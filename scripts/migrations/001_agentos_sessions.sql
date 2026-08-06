-- Schema for pkg/hno/session/db/postgres.
-- The table name is intentionally separate from the legacy `sessions` table
-- created by scripts/init-db.sql.
CREATE TABLE IF NOT EXISTS agno_agentos_sessions (
    session_id TEXT PRIMARY KEY,
    agent_id TEXT,
    team_id TEXT,
    workflow_id TEXT,
    user_id TEXT,
    name TEXT,
    metadata JSONB,
    state JSONB,
    agent_data JSONB,
    runs JSONB,
    summary JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_agno_agentos_sessions_agent_id
    ON agno_agentos_sessions(agent_id);
CREATE INDEX IF NOT EXISTS idx_agno_agentos_sessions_team_id
    ON agno_agentos_sessions(team_id);
CREATE INDEX IF NOT EXISTS idx_agno_agentos_sessions_workflow_id
    ON agno_agentos_sessions(workflow_id);
CREATE INDEX IF NOT EXISTS idx_agno_agentos_sessions_user_id
    ON agno_agentos_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_agno_agentos_sessions_updated_at
    ON agno_agentos_sessions(updated_at DESC);
