package surreal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/rexleimo/agno-go/pkg/hno/session"
)

var (
	validTable = regexp.MustCompile(`^[A-Za-z0-9_]+$`)
	nowFunc    = time.Now
)

// StorageConfig 配置 SurrealDB 存储
type StorageConfig struct {
	Table string
}

// Storage SurrealDB 会话存储实现
type Storage struct {
	client   *Client
	table    string
	tableRef string
}

const defaultTableName = "sessions"

// NewStorage 创建 SurrealDB 存储
func NewStorage(client *Client, cfg *StorageConfig) (*Storage, error) {
	if client == nil {
		return nil, errors.New("surreal: client cannot be nil")
	}

	table := defaultTableName
	if cfg != nil && cfg.Table != "" {
		table = cfg.Table
	}
	if !validTable.MatchString(table) {
		return nil, fmt.Errorf("surreal: invalid table name %q", table)
	}

	return &Storage{
		client:   client,
		table:    table,
		tableRef: table,
	}, nil
}

// Create 创建会话
func (s *Storage) Create(ctx context.Context, sess *session.Session) error {
	if sess == nil {
		return errors.New("surreal: session cannot be nil")
	}
	if strings.TrimSpace(sess.SessionID) == "" {
		return session.ErrInvalidSessionID
	}

	now := nowFunc().UTC()
	if sess.CreatedAt.IsZero() {
		sess.CreatedAt = now
	}
	if sess.UpdatedAt.IsZero() {
		sess.UpdatedAt = now
	}

	payload, err := sessionToPayload(sess, s.table)
	if err != nil {
		return err
	}

	query := fmt.Sprintf(`
LET $rid = type::thing('%s', $session_id);
UPDATE $rid CONTENT $data RETURN AFTER;
`, s.tableRef)

	vars := map[string]interface{}{
		"session_id": sess.SessionID,
		"data":       payload,
	}

	var result []map[string]interface{}
	if err := s.client.querySingle(ctx, query, vars, &result); err != nil {
		return err
	}
	if len(result) > 0 {
		return applyPayloadToSession(result[0], sess)
	}
	return nil
}

// Get 根据 ID 获取会话
func (s *Storage) Get(ctx context.Context, sessionID string) (*session.Session, error) {
	if strings.TrimSpace(sessionID) == "" {
		return nil, session.ErrInvalidSessionID
	}

	query := fmt.Sprintf(`SELECT * FROM type::thing('%s', $session_id);`, s.tableRef)
	vars := map[string]interface{}{
		"session_id": sessionID,
	}

	var result []map[string]interface{}
	if err := s.client.querySingle(ctx, query, vars, &result); err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return nil, session.ErrSessionNotFound
	}

	sess := &session.Session{}
	if err := applyPayloadToSession(result[0], sess); err != nil {
		return nil, err
	}
	return sess, nil
}

// Update 更新会话
func (s *Storage) Update(ctx context.Context, sess *session.Session) error {
	if sess == nil || strings.TrimSpace(sess.SessionID) == "" {
		return session.ErrInvalidSessionID
	}

	// 确保存在
	if _, err := s.Get(ctx, sess.SessionID); err != nil {
		return err
	}

	sess.UpdatedAt = nowFunc().UTC()
	payload, err := sessionToPayload(sess, s.table)
	if err != nil {
		return err
	}

	query := fmt.Sprintf(`
LET $rid = type::thing('%s', $session_id);
UPDATE $rid CONTENT $data RETURN AFTER;
`, s.tableRef)

	vars := map[string]interface{}{
		"session_id": sess.SessionID,
		"data":       payload,
	}

	var result []map[string]interface{}
	if err := s.client.querySingle(ctx, query, vars, &result); err != nil {
		return err
	}
	if len(result) > 0 {
		return applyPayloadToSession(result[0], sess)
	}
	return nil
}

// Delete 删除会话
func (s *Storage) Delete(ctx context.Context, sessionID string) error {
	if strings.TrimSpace(sessionID) == "" {
		return session.ErrInvalidSessionID
	}

	query := fmt.Sprintf(`DELETE type::thing('%s', $session_id);`, s.tableRef)
	vars := map[string]interface{}{
		"session_id": sessionID,
	}

	var result []map[string]interface{}
	if err := s.client.querySingle(ctx, query, vars, &result); err != nil {
		return err
	}
	if len(result) == 0 {
		return session.ErrSessionNotFound
	}
	return nil
}

// List 列出会话（支持过滤条件）
func (s *Storage) List(ctx context.Context, filters map[string]interface{}) ([]*session.Session, error) {
	builder := strings.Builder{}
	builder.WriteString(fmt.Sprintf("SELECT * FROM %s", s.tableRef))

	var conditions []string
	vars := map[string]interface{}{}

	for key, value := range filters {
		if value == nil {
			continue
		}
		switch key {
		case "agent_id", "user_id", "team_id", "workflow_id":
			param := "$" + key
			conditions = append(conditions, fmt.Sprintf("%s = %s", key, param))
			vars[key] = value
		}
	}

	if len(conditions) > 0 {
		builder.WriteString(" WHERE ")
		builder.WriteString(strings.Join(conditions, " AND "))
	}
	builder.WriteString(" ORDER BY updated_at DESC;")

	var records []map[string]interface{}
	if err := s.client.querySingle(ctx, builder.String(), vars, &records); err != nil {
		return nil, err
	}
	sessions := make([]*session.Session, 0, len(records))
	for _, payload := range records {
		sess := &session.Session{}
		if err := applyPayloadToSession(payload, sess); err != nil {
			return nil, err
		}
		sessions = append(sessions, sess)
	}
	return sessions, nil
}

// ListByAgent 根据 Agent 列出会话
func (s *Storage) ListByAgent(ctx context.Context, agentID string) ([]*session.Session, error) {
	return s.List(ctx, map[string]interface{}{"agent_id": agentID})
}

// ListByUser 根据用户列出会话
func (s *Storage) ListByUser(ctx context.Context, userID string) ([]*session.Session, error) {
	return s.List(ctx, map[string]interface{}{"user_id": userID})
}

// Close 关闭存储（无资源可释放）
func (s *Storage) Close() error {
	return nil
}

// BulkUpsertSessions 批量写入/更新会话
func (s *Storage) BulkUpsertSessions(ctx context.Context, sessions []*session.Session) error {
	if len(sessions) == 0 {
		return nil
	}

	payload := make([]map[string]interface{}, 0, len(sessions))
	for _, sess := range sessions {
		if sess == nil || strings.TrimSpace(sess.SessionID) == "" {
			return session.ErrInvalidSessionID
		}
		if sess.CreatedAt.IsZero() {
			sess.CreatedAt = nowFunc().UTC()
		}
		if sess.UpdatedAt.IsZero() {
			sess.UpdatedAt = sess.CreatedAt
		}

		data, err := sessionToPayload(sess, s.table)
		if err != nil {
			return err
		}
		payload = append(payload, data)
	}

	query := fmt.Sprintf(`
LET $records = $records;
FOR $item IN $records {
    LET $rid = type::thing('%s', $item.session_id);
    UPDATE $rid CONTENT $item;
};
`, s.tableRef)

	vars := map[string]interface{}{
		"records": payload,
	}
	_, err := s.client.execute(ctx, query, vars)
	return err
}

// Metrics 会话指标
type Metrics struct {
	TotalSessions   int
	ActiveLast24h   int
	UpdatedLastHour int
}

// Metrics 返回总体与最近活跃的会话数量
func (s *Storage) Metrics(ctx context.Context) (*Metrics, error) {
	total, err := s.count(ctx, fmt.Sprintf("SELECT count() AS total FROM %s;", s.tableRef), nil)
	if err != nil {
		return nil, err
	}

	active24hQuery := fmt.Sprintf(
		`SELECT count() AS total FROM %s WHERE updated_at >= time::now() - 24h;`,
		s.tableRef,
	)
	active24h, err := s.count(ctx, active24hQuery, nil)
	if err != nil {
		return nil, err
	}

	active1hQuery := fmt.Sprintf(
		`SELECT count() AS total FROM %s WHERE updated_at >= time::now() - 1h;`,
		s.tableRef,
	)
	active1h, err := s.count(ctx, active1hQuery, nil)
	if err != nil {
		return nil, err
	}

	return &Metrics{
		TotalSessions:   total,
		ActiveLast24h:   active24h,
		UpdatedLastHour: active1h,
	}, nil
}

func (s *Storage) count(ctx context.Context, query string, vars map[string]interface{}) (int, error) {
	var result []map[string]interface{}
	if err := s.client.querySingle(ctx, query, vars, &result); err != nil {
		return 0, err
	}
	if len(result) == 0 {
		return 0, nil
	}
	value, ok := result[0]["total"]
	if !ok {
		value, ok = result[0]["count"]
	}
	if !ok {
		return 0, nil
	}
	switch v := value.(type) {
	case float64:
		return int(v), nil
	case int:
		return v, nil
	case json.Number:
		i, err := v.Int64()
		if err != nil {
			return 0, err
		}
		return int(i), nil
	default:
		return 0, nil
	}
}

func sessionToPayload(sess *session.Session, table string) (map[string]interface{}, error) {
	copy := *sess
	copy.CreatedAt = copy.CreatedAt.UTC()
	copy.UpdatedAt = copy.UpdatedAt.UTC()

	raw, err := json.Marshal(copy)
	if err != nil {
		return nil, fmt.Errorf("surreal: failed to encode session: %w", err)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("surreal: failed to decode session payload: %w", err)
	}

	payload["created_at"] = copy.CreatedAt.Format(time.RFC3339Nano)
	payload["updated_at"] = copy.UpdatedAt.Format(time.RFC3339Nano)
	payload["id"] = fmt.Sprintf("%s:%s", table, copy.SessionID)
	payload["session_id"] = copy.SessionID
	return payload, nil
}

func applyPayloadToSession(payload map[string]interface{}, target *session.Session) error {
	if payload == nil {
		return errors.New("surreal: payload is nil")
	}

	if _, ok := payload["session_id"]; !ok {
		if idRaw, ok := payload["id"].(string); ok {
			if parts := strings.SplitN(idRaw, ":", 2); len(parts) == 2 {
				payload["session_id"] = parts[1]
			}
		}
	}
	if created, ok := payload["created_at"].(string); ok && created != "" {
		payload["created_at"] = created
	}
	if updated, ok := payload["updated_at"].(string); ok && updated != "" {
		payload["updated_at"] = updated
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("surreal: failed to encode payload: %w", err)
	}
	if err := json.Unmarshal(raw, target); err != nil {
		return fmt.Errorf("surreal: failed to decode payload: %w", err)
	}

	if target.CreatedAt.Location() == time.UTC {
		return nil
	}
	target.CreatedAt = target.CreatedAt.UTC()
	target.UpdatedAt = target.UpdatedAt.UTC()
	return nil
}
