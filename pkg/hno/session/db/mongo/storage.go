package mongo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/rexleimo/agno-go/pkg/hno/agent"
	"github.com/rexleimo/agno-go/pkg/hno/session"
)

const defaultOperationTimeout = 200 * time.Millisecond

// Config controls Mongo-based session storage behaviour.
type Config struct {
	Database         string
	Collection       string
	OperationTimeout time.Duration
}

// replaceResult captures the outcome of a replace operation.
type replaceResult struct {
	matchedCount  int64
	upsertedCount int64
}

type replaceOneFn func(ctx context.Context, filter interface{}, replacement interface{}, opts ...*options.ReplaceOptions) (replaceResult, error)
type findOneFn func(ctx context.Context, filter interface{}, out interface{}) error
type deleteOneFn func(ctx context.Context, filter interface{}) (int64, error)
type findFn func(ctx context.Context, filter interface{}) ([]*mongoSession, error)

// Storage implements session.Storage backed by MongoDB.
type Storage struct {
	replaceOne replaceOneFn
	findOne    findOneFn
	deleteOne  deleteOneFn
	find       findFn
	timeout    time.Duration
}

// NewStorage constructs a Mongo session storage instance.
func NewStorage(client *mongo.Client, cfg Config) (*Storage, error) {
	if client == nil {
		return nil, fmt.Errorf("mongo client cannot be nil")
	}
	if cfg.Database == "" {
		return nil, fmt.Errorf("database name is required")
	}
	if cfg.Collection == "" {
		cfg.Collection = "sessions"
	}

	collection := client.Database(cfg.Database).Collection(cfg.Collection)

	replace := func(ctx context.Context, filter interface{}, replacement interface{}, opts ...*options.ReplaceOptions) (replaceResult, error) {
		res, err := collection.ReplaceOne(ctx, filter, replacement, opts...)
		if err != nil {
			return replaceResult{}, err
		}
		return replaceResult{
			matchedCount:  res.MatchedCount,
			upsertedCount: res.UpsertedCount,
		}, nil
	}

	find := func(ctx context.Context, filter interface{}, out interface{}) error {
		return collection.FindOne(ctx, filter).Decode(out)
	}

	delete := func(ctx context.Context, filter interface{}) (int64, error) {
		res, err := collection.DeleteOne(ctx, filter)
		if err != nil {
			return 0, err
		}
		return res.DeletedCount, nil
	}

	findMany := func(ctx context.Context, filter interface{}) ([]*mongoSession, error) {
		cursor, err := collection.Find(ctx, filter)
		if err != nil {
			return nil, err
		}
		defer cursor.Close(ctx)

		var sessions []*mongoSession
		for cursor.Next(ctx) {
			var ms mongoSession
			if err := cursor.Decode(&ms); err != nil {
				return nil, err
			}
			sessions = append(sessions, ms.clone())
		}

		if err := cursor.Err(); err != nil {
			return nil, err
		}

		return sessions, nil
	}

	timeout := cfg.OperationTimeout
	if timeout <= 0 {
		timeout = defaultOperationTimeout
	}

	return &Storage{
		replaceOne: replace,
		findOne:    find,
		deleteOne:  delete,
		find:       findMany,
		timeout:    timeout,
	}, nil
}

// Create upserts a session document.
func (s *Storage) Create(ctx context.Context, sess *session.Session) error {
	if err := s.validateSession(sess); err != nil {
		return err
	}
	return s.upsert(ctx, sess, true)
}

// Get fetches a session by ID.
func (s *Storage) Get(ctx context.Context, sessionID string) (*session.Session, error) {
	if err := ensureContext(ctx); err != nil {
		return nil, err
	}
	if sessionID == "" {
		return nil, session.ErrInvalidSessionID
	}

	ctx, cancel := s.applyTimeout(ctx)
	defer cancel()

	var ms mongoSession
	if err := s.findOne(ctx, bson.M{"session_id": sessionID}, &ms); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, session.ErrSessionNotFound
		}
		return nil, err
	}

	return ms.toSession(), nil
}

// Update persists changes to an existing session.
func (s *Storage) Update(ctx context.Context, sess *session.Session) error {
	if err := s.validateSession(sess); err != nil {
		return err
	}
	return s.upsert(ctx, sess, false)
}

// Delete removes a session by ID.
func (s *Storage) Delete(ctx context.Context, sessionID string) error {
	if err := ensureContext(ctx); err != nil {
		return err
	}
	if sessionID == "" {
		return session.ErrInvalidSessionID
	}

	ctx, cancel := s.applyTimeout(ctx)
	defer cancel()

	deleted, err := s.deleteOne(ctx, bson.M{"session_id": sessionID})
	if err != nil {
		return err
	}
	if deleted == 0 {
		return session.ErrSessionNotFound
	}
	return nil
}

// List returns sessions that match the provided filters.
func (s *Storage) List(ctx context.Context, filters map[string]interface{}) ([]*session.Session, error) {
	if err := ensureContext(ctx); err != nil {
		return nil, err
	}

	ctx, cancel := s.applyTimeout(ctx)
	defer cancel()

	bsonFilter := buildFilter(filters)
	results, err := s.find(ctx, bsonFilter)
	if err != nil {
		return nil, err
	}

	sessions := make([]*session.Session, 0, len(results))
	for _, ms := range results {
		sessions = append(sessions, ms.toSession())
	}
	return sessions, nil
}

// ListByAgent lists sessions for a specific agent.
func (s *Storage) ListByAgent(ctx context.Context, agentID string) ([]*session.Session, error) {
	return s.List(ctx, map[string]interface{}{"agent_id": agentID})
}

// ListByUser lists sessions for a specific user.
func (s *Storage) ListByUser(ctx context.Context, userID string) ([]*session.Session, error) {
	return s.List(ctx, map[string]interface{}{"user_id": userID})
}

// Close releases storage resources (noop for Mongo storage).
func (s *Storage) Close() error {
	return nil
}

func (s *Storage) upsert(ctx context.Context, sess *session.Session, creating bool) error {
	if err := ensureContext(ctx); err != nil {
		return err
	}

	ctx, cancel := s.applyTimeout(ctx)
	defer cancel()

	model := toMongoSession(sess)
	if creating && model.CreatedAt.IsZero() {
		model.CreatedAt = time.Now().UTC()
	}
	if model.UpdatedAt.IsZero() {
		model.UpdatedAt = time.Now().UTC()
	} else {
		model.UpdatedAt = time.Now().UTC()
	}

	opts := options.Replace().SetUpsert(true)
	res, err := s.replaceOne(ctx, bson.M{"session_id": model.SessionID}, model, opts)
	if err != nil {
		return err
	}

	if !creating && res.matchedCount == 0 {
		return session.ErrSessionNotFound
	}

	return nil
}

func (s *Storage) applyTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if s.timeout <= 0 {
		return ctx, func() {}
	}

	if deadline, ok := ctx.Deadline(); ok {
		if time.Until(deadline) <= s.timeout {
			return ctx, func() {}
		}
	}

	return context.WithTimeout(ctx, s.timeout)
}

func (s *Storage) validateSession(sess *session.Session) error {
	if sess == nil {
		return fmt.Errorf("session cannot be nil")
	}
	if sess.SessionID == "" {
		return session.ErrInvalidSessionID
	}
	return nil
}

func ensureContext(ctx context.Context) error {
	if ctx == nil {
		return nil
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

func buildFilter(filters map[string]interface{}) interface{} {
	if len(filters) == 0 {
		return bson.M{}
	}

	bsonFilter := bson.M{}
	for key, value := range filters {
		switch key {
		case "agent_id", "user_id", "team_id", "workflow_id", "session_id":
			if str, ok := value.(string); ok && str != "" {
				bsonFilter[key] = str
			}
		}
	}
	return bsonFilter
}

type mongoSession struct {
	SessionID  string                  `bson:"session_id"`
	AgentID    string                  `bson:"agent_id,omitempty"`
	TeamID     string                  `bson:"team_id,omitempty"`
	WorkflowID string                  `bson:"workflow_id,omitempty"`
	UserID     string                  `bson:"user_id,omitempty"`
	Name       string                  `bson:"name,omitempty"`
	Metadata   map[string]interface{}  `bson:"metadata,omitempty"`
	State      map[string]interface{}  `bson:"state,omitempty"`
	AgentData  map[string]interface{}  `bson:"agent_data,omitempty"`
	Runs       []*agent.RunOutput      `bson:"runs,omitempty"`
	Summary    *session.SessionSummary `bson:"summary,omitempty"`
	CreatedAt  time.Time               `bson:"created_at"`
	UpdatedAt  time.Time               `bson:"updated_at"`
}

func (m *mongoSession) toSession() *session.Session {
	if m == nil {
		return nil
	}
	return &session.Session{
		SessionID:  m.SessionID,
		AgentID:    m.AgentID,
		TeamID:     m.TeamID,
		WorkflowID: m.WorkflowID,
		UserID:     m.UserID,
		Name:       m.Name,
		Metadata:   copyMap(m.Metadata),
		State:      copyMap(m.State),
		AgentData:  copyMap(m.AgentData),
		Runs:       copyRuns(m.Runs),
		Summary:    copySummary(m.Summary),
		CreatedAt:  m.CreatedAt,
		UpdatedAt:  m.UpdatedAt,
	}
}

func (m *mongoSession) clone() *mongoSession {
	if m == nil {
		return nil
	}
	clone := *m
	clone.Metadata = copyMap(m.Metadata)
	clone.State = copyMap(m.State)
	clone.AgentData = copyMap(m.AgentData)
	clone.Runs = copyRuns(m.Runs)
	clone.Summary = copySummary(m.Summary)
	return &clone
}

func toMongoSession(sess *session.Session) *mongoSession {
	if sess == nil {
		return nil
	}
	return &mongoSession{
		SessionID:  sess.SessionID,
		AgentID:    sess.AgentID,
		TeamID:     sess.TeamID,
		WorkflowID: sess.WorkflowID,
		UserID:     sess.UserID,
		Name:       sess.Name,
		Metadata:   copyMap(sess.Metadata),
		State:      copyMap(sess.State),
		AgentData:  copyMap(sess.AgentData),
		Runs:       copyRuns(sess.Runs),
		Summary:    copySummary(sess.Summary),
		CreatedAt:  sess.CreatedAt,
		UpdatedAt:  sess.UpdatedAt,
	}
}

func copyMap(src map[string]interface{}) map[string]interface{} {
	if src == nil {
		return nil
	}
	dst := make(map[string]interface{}, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func copyRuns(src []*agent.RunOutput) []*agent.RunOutput {
	if len(src) == 0 {
		return nil
	}
	cloned := make([]*agent.RunOutput, len(src))
	copy(cloned, src)
	return cloned
}

func copySummary(src *session.SessionSummary) *session.SessionSummary {
	if src == nil {
		return nil
	}
	clone := *src
	return &clone
}
