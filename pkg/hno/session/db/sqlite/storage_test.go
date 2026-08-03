package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/rexleimo/agno-go/pkg/hno/session"
)

func TestNewStorage_CreatesSchema(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()

	store, err := NewStorage(db, Config{})
	if err != nil {
		t.Fatalf("NewStorage() error = %v", err)
	}
	defer store.Close()

	if _, err := db.Exec(`SELECT session_id FROM sessions LIMIT 1`); err != nil {
		t.Fatalf("expected sessions table, got error: %v", err)
	}
}

func TestStorage_CreateAndGet(t *testing.T) {
	db := openTestDB(t)
	store := mustNewStorage(t, db)
	defer store.Close()

	ctx := context.Background()
	sess := session.NewSession("sqlite-1", "agent-1")
	sess.Metadata["tier"] = "gold"

	if err := store.Create(ctx, sess); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	retrieved, err := store.Get(ctx, "sqlite-1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if retrieved.Metadata["tier"] != "gold" {
		t.Fatalf("metadata mismatch: %#v", retrieved.Metadata)
	}

	if retrieved.SessionID != "sqlite-1" {
		t.Fatalf("unexpected session id: %s", retrieved.SessionID)
	}
}

func TestStorage_Update(t *testing.T) {
	db := openTestDB(t)
	store := mustNewStorage(t, db)
	defer store.Close()

	ctx := context.Background()
	sess := session.NewSession("sqlite-2", "agent-1")
	if err := store.Create(ctx, sess); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	sess.Name = "Renamed"
	sess.UserID = "user-42"

	if err := store.Update(ctx, sess); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	retrieved, err := store.Get(ctx, "sqlite-2")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if retrieved.Name != "Renamed" || retrieved.UserID != "user-42" {
		t.Fatalf("update did not persist: %#v", retrieved)
	}
}

func TestStorage_ListFilters(t *testing.T) {
	db := openTestDB(t)
	store := mustNewStorage(t, db)
	defer store.Close()

	ctx := context.Background()

	sessA := session.NewSession("sqlite-3", "agent-a")
	sessA.UserID = "user-1"
	sessB := session.NewSession("sqlite-4", "agent-b")
	sessB.UserID = "user-2"

	if err := store.Create(ctx, sessA); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if err := store.Create(ctx, sessB); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	results, err := store.ListByUser(ctx, "user-1")
	if err != nil {
		t.Fatalf("ListByUser() error = %v", err)
	}
	if len(results) != 1 || results[0].SessionID != "sqlite-3" {
		t.Fatalf("unexpected results: %#v", results)
	}
}

func TestStorage_ContextDeadline(t *testing.T) {
	db := openTestDB(t)
	store := mustNewStorage(t, db)
	defer store.Close()

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-1*time.Millisecond))
	defer cancel()

	err := store.Create(ctx, session.NewSession("deadline", "agent"))
	if err == nil {
		t.Fatalf("expected context error, got nil")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline exceeded, got %v", err)
	}
}

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", "file:test.db?mode=memory&cache=shared&_pragma=busy_timeout=5000")
	if err != nil {
		t.Fatalf("sql.Open failed: %v", err)
	}
	return db
}

func mustNewStorage(t *testing.T, db *sql.DB) *Storage {
	t.Helper()
	store, err := NewStorage(db, Config{})
	if err != nil {
		t.Fatalf("NewStorage() error = %v", err)
	}
	return store
}
