package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/rexleimo/agno-go/internal/session/dto"
	"github.com/rexleimo/agno-go/internal/session/store"
)

const sessionTableDDL = `
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
CREATE INDEX IF NOT EXISTS idx_agno_sessions_type ON agno_sessions(session_type);
CREATE INDEX IF NOT EXISTS idx_agno_sessions_created_at ON agno_sessions(created_at);
`

var errDockerUnavailable = errors.New("docker unavailable")

func setupTestStore(t *testing.T) (*Store, func()) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)

	container, err := startPostgresContainer(ctx)
	if err != nil {
		cancel()
		if errors.Is(err, errDockerUnavailable) {
			t.Skipf("skipping Postgres store tests: %v", err)
		}
		t.Fatalf("failed to start postgres container: %v", err)
	}

	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		container.Terminate(ctx) //nolint:errcheck
		cancel()
		t.Fatalf("failed to get postgres connection string: %v", err)
	}

	st, err := New(ctx, Config{DSN: connStr})
	if err != nil {
		container.Terminate(ctx) //nolint:errcheck
		cancel()
		t.Fatalf("failed to create store: %v", err)
	}

	if err := bootstrapSessionTable(ctx, st.pool); err != nil {
		st.Close()
		container.Terminate(ctx) //nolint:errcheck
		cancel()
		t.Fatalf("failed to bootstrap postgres session table: %v", err)
	}

	cleanup := func() {
		st.Close()
		container.Terminate(context.Background()) //nolint:errcheck
		cancel()
	}

	return st, cleanup
}

func bootstrapSessionTable(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, sessionTableDDL)
	return err
}

func startPostgresContainer(ctx context.Context) (container *postgres.PostgresContainer, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%w: %v", errDockerUnavailable, r)
		}
	}()

	container, err = postgres.RunContainer(ctx,
		testcontainers.WithImage("postgres:16-alpine"),
		postgres.WithDatabase("agno"),
		postgres.WithUsername("agno"),
		postgres.WithPassword("agno"),
		// BasicWaitStrategies waits for both PostgreSQL's second ready log and
		// Docker Desktop's host-side port proxy. The latter is required on
		// Windows; a log-only wait can still yield EOF from pgx.
		postgres.BasicWaitStrategies(),
	)
	if err != nil && isDockerUnavailable(err) {
		err = fmt.Errorf("%w: %v", errDockerUnavailable, err)
	}
	return
}

func isDockerUnavailable(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "Cannot connect to the Docker daemon") || strings.Contains(msg, "permission denied while trying to connect to the Docker daemon")
}

func sampleRecord(sessionID string, sessionType dto.SessionType) *dto.SessionRecord {
	agentID := "agent-123"
	userID := "user-abc"
	return &dto.SessionRecord{
		SessionID:   sessionID,
		SessionType: sessionType,
		AgentID:     &agentID,
		UserID:      &userID,
		SessionData: map[string]any{
			"session_name":  "Test Session",
			"session_state": map[string]any{"step": "init"},
		},
		AgentData: map[string]any{"name": "basic-agent"},
		Metadata:  map[string]any{"priority": "normal"},
		Runs: []map[string]any{
			{
				"run_id":     "run-1",
				"status":     "completed",
				"started_at": time.Now().Add(-time.Minute).Unix(),
			},
		},
		CreatedAt: time.Unix(1_700_000_000, 0).UTC(),
		UpdatedAt: time.Unix(1_700_000_000, 0).UTC(),
	}
}

func TestUpsertSessionInsertAndUpdate(t *testing.T) {
	pgStore, cleanup := setupTestStore(t)
	defer cleanup()
	ctx := context.Background()

	record := sampleRecord("session-1", dto.SessionTypeAgent)

	inserted, err := pgStore.UpsertSession(ctx, record, false)
	require.NoError(t, err)
	require.Equal(t, record.SessionID, inserted.SessionID)
	require.Equal(t, record.CreatedAt.UTC(), inserted.CreatedAt)
	require.Equal(t, record.SessionData["session_name"], inserted.SessionName())

	record.SessionData["session_state"].(map[string]any)["step"] = "updated"
	record.UpdatedAt = record.UpdatedAt.Add(time.Minute)

	updated, err := pgStore.UpsertSession(ctx, record, false)
	require.NoError(t, err)
	require.Equal(t, "updated", updated.SessionState()["step"])
	require.True(t, updated.UpdatedAt.After(updated.CreatedAt))

	fetched, err := pgStore.GetSession(ctx, record.SessionID, dto.SessionTypeAgent)
	require.NoError(t, err)
	require.Equal(t, updated.SessionState()["step"], fetched.SessionState()["step"])
}

func TestUpsertPreservesCreatedAt(t *testing.T) {
	pgStore, cleanup := setupTestStore(t)
	defer cleanup()
	ctx := context.Background()

	record := sampleRecord("session-2", dto.SessionTypeAgent)

	inserted, err := pgStore.UpsertSession(ctx, record, false)
	require.NoError(t, err)
	initialCreated := inserted.CreatedAt

	record.SessionData["session_name"] = "Updated Name"
	record.CreatedAt = time.Unix(1_800_000_000, 0)
	record.UpdatedAt = time.Unix(1_800_000_100, 0)

	updated, err := pgStore.UpsertSession(ctx, record, false)
	require.NoError(t, err)
	require.Equal(t, initialCreated, updated.CreatedAt)
	require.Equal(t, "Updated Name", updated.SessionName())
}

func TestListSessionsFiltersAndPaging(t *testing.T) {
	pgStore, cleanup := setupTestStore(t)
	defer cleanup()
	ctx := context.Background()

	agentA := sampleRecord("session-a", dto.SessionTypeAgent)
	agentB := sampleRecord("session-b", dto.SessionTypeAgent)
	team := sampleRecord("session-team", dto.SessionTypeTeam)
	team.AgentID = nil
	teamID := "team-999"
	team.TeamID = &teamID

	records := []*dto.SessionRecord{agentA, agentB, team}
	for _, rec := range records {
		_, err := pgStore.UpsertSession(ctx, rec, false)
		require.NoError(t, err)
	}

	items, total, err := pgStore.ListSessions(ctx, store.ListSessionsOptions{
		SessionType: dto.SessionTypeAgent,
		Limit:       1,
		Page:        2,
		SortBy:      "created_at",
		SortOrder:   "asc",
	})
	require.NoError(t, err)
	require.Equal(t, 2, total)
	require.Len(t, items, 1)

	componentFiltered, _, err := pgStore.ListSessions(ctx, store.ListSessionsOptions{
		SessionType: dto.SessionTypeTeam,
		ComponentID: teamID,
	})
	require.NoError(t, err)
	require.Len(t, componentFiltered, 1)
	require.Equal(t, "session-team", componentFiltered[0].SessionID)
}

func TestRenameAndDeleteSession(t *testing.T) {
	pgStore, cleanup := setupTestStore(t)
	defer cleanup()
	ctx := context.Background()

	record := sampleRecord("session-del", dto.SessionTypeAgent)
	_, err := pgStore.UpsertSession(ctx, record, false)
	require.NoError(t, err)

	renamed, err := pgStore.RenameSession(ctx, record.SessionID, dto.SessionTypeAgent, "Renamed Session")
	require.NoError(t, err)
	require.Equal(t, "Renamed Session", renamed.SessionName())
	require.True(t, renamed.UpdatedAt.After(renamed.CreatedAt))

	require.NoError(t, pgStore.DeleteSession(ctx, record.SessionID, dto.SessionTypeAgent))
	_, err = pgStore.GetSession(ctx, record.SessionID, dto.SessionTypeAgent)
	require.ErrorIs(t, err, store.ErrNotFound)
}

func TestConcurrentUpsertSession(t *testing.T) {
	pgStore, cleanup := setupTestStore(t)
	defer cleanup()
	ctx := context.Background()

	record := sampleRecord("session-concurrent", dto.SessionTypeAgent)

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(iter int) {
			defer wg.Done()
			recordCopy := *record
			state := map[string]any{"iteration": iter}
			recordCopy.SessionData = map[string]any{
				"session_name":  "Concurrent",
				"session_state": state,
			}
			recordCopy.UpdatedAt = recordCopy.CreatedAt.Add(time.Duration(iter+1) * time.Second)
			_, err := pgStore.UpsertSession(ctx, &recordCopy, false)
			require.NoError(t, err)
		}(i)
	}

	wg.Wait()

	fetched, err := pgStore.GetSession(ctx, record.SessionID, dto.SessionTypeAgent)
	require.NoError(t, err)
	require.Equal(t, "Concurrent", fetched.SessionName())
}
