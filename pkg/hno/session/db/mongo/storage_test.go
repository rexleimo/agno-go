package mongo

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/rexleimo/agno-go/pkg/hno/session"
)

func TestNewStorage_NilClient(t *testing.T) {
	if _, err := NewStorage(nil, Config{}); err == nil {
		t.Fatalf("expected error when client is nil")
	}
}

func TestStorage_CreateAndGet(t *testing.T) {
	store, fake := newTestStorage()
	ctx := context.Background()

	sess := session.NewSession("mongo-1", "agent-1")
	sess.Metadata["topic"] = "demo"

	if err := store.Create(ctx, sess); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	retrieved, err := store.Get(ctx, "mongo-1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if retrieved.SessionID != "mongo-1" {
		t.Fatalf("unexpected session id: %s", retrieved.SessionID)
	}

	if retrieved.Metadata["topic"] != "demo" {
		t.Fatalf("metadata not persisted: %#v", retrieved.Metadata)
	}

	if len(fake.docs) != 1 {
		t.Fatalf("expected 1 stored doc, got %d", len(fake.docs))
	}
}

func TestStorage_UpdateNonexistent(t *testing.T) {
	store, _ := newTestStorage()
	ctx := context.Background()

	sess := session.NewSession("missing", "agent")

	err := store.Update(ctx, sess)
	if !errors.Is(err, session.ErrSessionNotFound) {
		t.Fatalf("expected ErrSessionNotFound, got %v", err)
	}
}

func TestStorage_Delete(t *testing.T) {
	store, _ := newTestStorage()
	ctx := context.Background()

	sess := session.NewSession("mongo-2", "agent-1")
	if err := store.Create(ctx, sess); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if err := store.Delete(ctx, "mongo-2"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	if _, err := store.Get(ctx, "mongo-2"); !errors.Is(err, session.ErrSessionNotFound) {
		t.Fatalf("expected ErrSessionNotFound, got %v", err)
	}
}

func TestStorage_ListFilters(t *testing.T) {
	store, _ := newTestStorage()
	ctx := context.Background()

	sessionA := session.NewSession("mongo-3", "agent-a")
	sessionA.UserID = "user-1"
	sessionB := session.NewSession("mongo-4", "agent-b")
	sessionB.UserID = "user-2"

	if err := store.Create(ctx, sessionA); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if err := store.Create(ctx, sessionB); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	results, err := store.ListByUser(ctx, "user-1")
	if err != nil {
		t.Fatalf("ListByUser() error = %v", err)
	}
	if len(results) != 1 || results[0].SessionID != "mongo-3" {
		t.Fatalf("unexpected list results: %#v", results)
	}
}

func TestStorage_ContextDeadline(t *testing.T) {
	store, _ := newTestStorage()
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-1*time.Millisecond))
	defer cancel()

	err := store.Create(ctx, session.NewSession("mongo-dead", "agent"))
	if err == nil {
		t.Fatalf("expected context error, got nil")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline exceeded, got %v", err)
	}
}

type fakeCollection struct {
	mu   sync.Mutex
	docs map[string]*mongoSession
}

func newTestStorage() (*Storage, *fakeCollection) {
	fake := &fakeCollection{docs: make(map[string]*mongoSession)}
	store := &Storage{
		replaceOne: fake.replaceOne,
		findOne:    fake.findOne,
		deleteOne:  fake.deleteOne,
		find:       fake.findMany,
		timeout:    10 * time.Millisecond,
	}
	return store, fake
}

func (f *fakeCollection) replaceOne(ctx context.Context, filter interface{}, replacement interface{}, opts ...*options.ReplaceOptions) (replaceResult, error) {
	if err := ctx.Err(); err != nil {
		return replaceResult{}, err
	}

	ms, ok := replacement.(*mongoSession)
	if !ok {
		return replaceResult{}, errors.New("replacement must be mongoSession")
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	_, existed := f.docs[ms.SessionID]
	f.docs[ms.SessionID] = ms.clone()

	res := replaceResult{}
	if existed {
		res.matchedCount = 1
	} else {
		res.upsertedCount = 1
	}

	return res, nil
}

func (f *fakeCollection) findOne(ctx context.Context, filter interface{}, out interface{}) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	filterMap, ok := filter.(bson.M)
	if !ok {
		return errors.New("unexpected filter type")
	}
	val, ok := filterMap["session_id"].(string)
	if !ok {
		return errors.New("missing session_id filter")
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	ms, exists := f.docs[val]
	if !exists {
		return mongo.ErrNoDocuments
	}

	target, ok := out.(*mongoSession)
	if !ok {
		return errors.New("out must be *mongoSession")
	}

	*target = *ms.clone()
	return nil
}

func (f *fakeCollection) deleteOne(ctx context.Context, filter interface{}) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}

	filterMap, ok := filter.(bson.M)
	if !ok {
		return 0, errors.New("unexpected filter type")
	}
	val, ok := filterMap["session_id"].(string)
	if !ok {
		return 0, errors.New("missing session_id filter")
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	if _, exists := f.docs[val]; !exists {
		return 0, nil
	}

	delete(f.docs, val)
	return 1, nil
}

func (f *fakeCollection) findMany(ctx context.Context, filter interface{}) ([]*mongoSession, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	filterMap, _ := filter.(bson.M)

	f.mu.Lock()
	defer f.mu.Unlock()

	var results []*mongoSession
	for _, ms := range f.docs {
		if matchFilter(ms, filterMap) {
			results = append(results, ms.clone())
		}
	}

	return results, nil
}

func matchFilter(ms *mongoSession, filter bson.M) bool {
	if len(filter) == 0 {
		return true
	}
	for key, value := range filter {
		switch key {
		case "agent_id":
			if ms.AgentID != value {
				return false
			}
		case "user_id":
			if ms.UserID != value {
				return false
			}
		case "team_id":
			if ms.TeamID != value {
				return false
			}
		case "workflow_id":
			if ms.WorkflowID != value {
				return false
			}
		case "session_id":
			if ms.SessionID != value {
				return false
			}
		}
	}
	return true
}
