package contract

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/rexleimo/agno-go/internal/http/router"
	"github.com/rexleimo/agno-go/internal/session/dto"
	"github.com/rexleimo/agno-go/internal/session/service"
	"github.com/rexleimo/agno-go/internal/session/store"
)

func TestListSessionsMatchesFixture(t *testing.T) {
	listFixture := loadListFixture(t, "get_sessions_agent.json")
	store := newFixtureStore()
	store.listRecords = recordsFromListFixture(t, listFixture)
	store.listTotal = listFixture.Meta.TotalCount

	svc := serviceFromStore(store)
	server := httptest.NewServer(router.New(svc))
	defer server.Close()

	resp := doRequest(t, http.MethodGet, server.URL+"/sessions?type=agent", nil)
	defer resp.Body.Close()

	var got map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&got))

	expected := map[string]any{}
	fixtureBytes := loadRawFixture(t, "get_sessions_agent.json")
	require.NoError(t, json.Unmarshal(fixtureBytes, &expected))

	assertJSONEqual(t, expected, got)
}

func TestGetSessionDetailMatchesFixture(t *testing.T) {
	detailFixture := loadDetailFixture(t, "get_session_detail_agent.json")
	runsFixture := loadRunsFixture(t, "get_session_runs.json")
	store := newFixtureStore()
	record := recordFromDetailFixture(t, detailFixture)
	record.Runs = runsFixture
	store.records[record.SessionID] = record

	svc := serviceFromStore(store)
	server := httptest.NewServer(router.New(svc))
	defer server.Close()

	url := server.URL + "/sessions/" + record.SessionID + "?type=agent"
	resp := doRequest(t, http.MethodGet, url, nil)
	defer resp.Body.Close()

	var got map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&got))

	expected := map[string]any{}
	require.NoError(t, json.Unmarshal(loadRawFixture(t, "get_session_detail_agent.json"), &expected))

	assertJSONEqual(t, expected, got)
}

func TestCreateSessionMatchesFixture(t *testing.T) {
	createFixture := loadCreateFixture(t, "create_session_agent.json")
	store := newFixtureStore()
	store.upsertResponse = recordFromCreateFixture(createFixture)
	store.records[store.upsertResponse.SessionID] = store.upsertResponse

	svc := serviceFromStore(store)
	server := httptest.NewServer(router.New(svc))
	defer server.Close()

	requestPayload := map[string]any{
		"session_id":    createFixture.SessionID,
		"session_name":  createFixture.SessionName,
		"session_state": createFixture.SessionState,
		"metadata":      createFixture.Metadata,
		"agent_id":      createFixture.AgentID,
		"user_id":       createFixture.UserID,
	}
	body, _ := json.Marshal(requestPayload)

	resp := doRequest(t, http.MethodPost, server.URL+"/sessions?type=agent", body)
	defer resp.Body.Close()

	var got map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&got))

	expected := map[string]any{}
	require.NoError(t, json.Unmarshal(loadRawFixture(t, "create_session_agent.json"), &expected))

	assertJSONEqual(t, expected, got)
}

func TestRenameSessionMatchesFixture(t *testing.T) {
	renameFixture := loadDetailFixture(t, "rename_session_agent.json")
	store := newFixtureStore()
	store.renameResponse = recordFromDetailFixture(t, renameFixture)
	store.records[store.renameResponse.SessionID] = store.renameResponse

	svc := serviceFromStore(store)
	server := httptest.NewServer(router.New(svc))
	defer server.Close()

	payload := map[string]string{"session_name": renameFixture.SessionName}
	body, _ := json.Marshal(payload)

	url := server.URL + "/sessions/" + renameFixture.SessionID + "/rename?type=agent"
	resp := doRequest(t, http.MethodPost, url, body)
	defer resp.Body.Close()

	var got map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&got))

	expected := map[string]any{}
	require.NoError(t, json.Unmarshal(loadRawFixture(t, "rename_session_agent.json"), &expected))

	assertJSONEqual(t, expected, got)
}

func TestGetSessionRunsMatchesFixture(t *testing.T) {
	runsFixture := loadRunsFixture(t, "get_session_runs.json")
	detailFixture := loadDetailFixture(t, "get_session_detail_agent.json")
	store := newFixtureStore()
	record := recordFromDetailFixture(t, detailFixture)
	record.Runs = runsFixture
	store.records[record.SessionID] = record

	svc := serviceFromStore(store)
	server := httptest.NewServer(router.New(svc))
	defer server.Close()

	url := server.URL + "/sessions/" + record.SessionID + "/runs?type=agent"
	resp := doRequest(t, http.MethodGet, url, nil)
	defer resp.Body.Close()

	var got map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&got))

	expectedRuns := []any{}
	require.NoError(t, json.Unmarshal(loadRawFixture(t, "get_session_runs.json"), &expectedRuns))

	assertJSONEqual(t, map[string]any{"runs": expectedRuns}, got)
}

func TestListSessionsInvalidTypeMatchesErrorFixture(t *testing.T) {
	store := newFixtureStore()
	svc := serviceFromStore(store)
	server := httptest.NewServer(router.New(svc))
	defer server.Close()

	resp := doRequest(t, http.MethodGet, server.URL+"/sessions?type=invalid", nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400 for invalid type, got %d", resp.StatusCode)
	}

	var got map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&got))

	expected := map[string]any{}
	require.NoError(t, json.Unmarshal(loadRawFixture(t, "get_sessions_invalid_type_error.json"), &expected))

	assertJSONEqual(t, expected, got)
}

func TestGetSessionDetailNotFoundMatchesErrorFixture(t *testing.T) {
	store := newFixtureStore()
	// Do not seed any records so the store returns ErrNotFound.

	svc := serviceFromStore(store)
	server := httptest.NewServer(router.New(svc))
	defer server.Close()

	url := server.URL + "/sessions/non-existent-session?type=agent"
	resp := doRequest(t, http.MethodGet, url, nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected status 404 for missing session, got %d", resp.StatusCode)
	}

	var got map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&got))

	expected := map[string]any{}
	require.NoError(t, json.Unmarshal(loadRawFixture(t, "get_session_not_found_error.json"), &expected))

	assertJSONEqual(t, expected, got)
}

func TestListSessionsDatabaseRequiredMatchesErrorFixture(t *testing.T) {
	// Configure service with multiple stores and no default so db_id is required.
	st := newFixtureStore()
	svc, err := service.New(service.Config{
		Stores: map[string]store.Store{
			"primary":   st,
			"secondary": st,
		},
	})
	if err != nil {
		t.Fatalf("failed to init service with multiple stores: %v", err)
	}

	server := httptest.NewServer(router.New(svc))
	defer server.Close()

	// Missing db_id when multiple databases are configured should produce DATABASE_REQUIRED.
	resp := doRequest(t, http.MethodGet, server.URL+"/sessions?type=agent", nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400 for missing db_id, got %d", resp.StatusCode)
	}

	var got map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&got))

	expected := map[string]any{}
	require.NoError(t, json.Unmarshal(loadRawFixture(t, "list_sessions_database_required_error.json"), &expected))

	assertJSONEqual(t, expected, got)
}

func TestListSessionsDatabaseNotFoundMatchesErrorFixture(t *testing.T) {
	// Configure service with a known store but request an unknown db_id.
	st := newFixtureStore()
	svc, err := service.New(service.Config{
		Stores: map[string]store.Store{
			"default": st,
		},
	})
	if err != nil {
		t.Fatalf("failed to init service with single store: %v", err)
	}

	server := httptest.NewServer(router.New(svc))
	defer server.Close()

	resp := doRequest(t, http.MethodGet, server.URL+"/sessions?type=agent&db_id=unknown", nil)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected status 404 for unknown db_id, got %d", resp.StatusCode)
	}

	var got map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&got))

	expected := map[string]any{}
	require.NoError(t, json.Unmarshal(loadRawFixture(t, "list_sessions_database_not_found_error.json"), &expected))

	assertJSONEqual(t, expected, got)
}

func doRequest(t *testing.T, method, url string, payload []byte) *http.Response {
	t.Helper()
	var bodyReader io.Reader
	if payload != nil {
		bodyReader = bytes.NewReader(payload)
	}
	req, err := http.NewRequest(method, url, bodyReader)
	require.NoError(t, err)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	return resp
}

type fixtureStore struct {
	listRecords    []*dto.SessionRecord
	listTotal      int
	records        map[string]*dto.SessionRecord
	upsertResponse *dto.SessionRecord
	renameResponse *dto.SessionRecord
}

func newFixtureStore() *fixtureStore {
	return &fixtureStore{
		records: make(map[string]*dto.SessionRecord),
	}
}

func (f *fixtureStore) UpsertSession(_ context.Context, record *dto.SessionRecord, _ bool) (*dto.SessionRecord, error) {
	f.records[record.SessionID] = record
	if f.upsertResponse != nil {
		return cloneRecord(f.upsertResponse), nil
	}
	return cloneRecord(record), nil
}

func (f *fixtureStore) ListSessions(_ context.Context, _ store.ListSessionsOptions) ([]*dto.SessionRecord, int, error) {
	if f.listRecords == nil {
		return []*dto.SessionRecord{}, 0, nil
	}
	clones := make([]*dto.SessionRecord, 0, len(f.listRecords))
	for _, record := range f.listRecords {
		clones = append(clones, cloneRecord(record))
	}
	return clones, f.listTotal, nil
}

func (f *fixtureStore) GetSession(_ context.Context, sessionID string, _ dto.SessionType) (*dto.SessionRecord, error) {
	record, ok := f.records[sessionID]
	if !ok {
		return nil, store.ErrNotFound
	}
	return cloneRecord(record), nil
}

func (f *fixtureStore) DeleteSession(_ context.Context, sessionID string, _ dto.SessionType) error {
	delete(f.records, sessionID)
	return nil
}

func (f *fixtureStore) RenameSession(_ context.Context, sessionID string, _ dto.SessionType, _ string) (*dto.SessionRecord, error) {
	if f.renameResponse != nil {
		return cloneRecord(f.renameResponse), nil
	}
	return f.GetSession(context.Background(), sessionID, dto.SessionTypeAgent)
}

func serviceFromStore(st store.Store) *service.Service {
	svc, err := service.New(service.Config{Stores: map[string]store.Store{"default": st}, DefaultDB: "default"})
	if err != nil {
		panic(err)
	}
	return svc
}

func loadListFixture(t *testing.T, name string) listFixture {
	var fixture listFixture
	require.NoError(t, json.Unmarshal(loadRawFixture(t, name), &fixture))
	return fixture
}

func loadDetailFixture(t *testing.T, name string) detailFixture {
	var fixture detailFixture
	require.NoError(t, json.Unmarshal(loadRawFixture(t, name), &fixture))
	return fixture
}

func loadCreateFixture(t *testing.T, name string) createFixture {
	var fixture createFixture
	require.NoError(t, json.Unmarshal(loadRawFixture(t, name), &fixture))
	return fixture
}

func loadRunsFixture(t *testing.T, name string) []map[string]any {
	var runs []map[string]any
	require.NoError(t, json.Unmarshal(loadRawFixture(t, name), &runs))
	return runs
}

func loadRawFixture(t *testing.T, name string) []byte {
	t.Helper()
	_, currentFile, _, _ := runtime.Caller(0)
	fixturesDir := filepath.Join(filepath.Dir(currentFile), "../../../../contract-fixtures")
	fixturePath := filepath.Join(fixturesDir, name)
	data, err := os.ReadFile(fixturePath)
	if os.IsNotExist(err) {
		t.Skipf("contract fixture %q is not available; provide %s to run parity tests", name, fixturesDir)
	}
	require.NoError(t, err)
	return data
}

func assertJSONEqual(t *testing.T, expected, actual any) {
	t.Helper()
	if !reflect.DeepEqual(expected, actual) {
		expectedJSON, _ := json.MarshalIndent(expected, "", "  ")
		actualJSON, _ := json.MarshalIndent(actual, "", "  ")
		t.Fatalf("expected JSON:\n%s\nactual JSON:\n%s", expectedJSON, actualJSON)
	}
}

func cloneRecord(record *dto.SessionRecord) *dto.SessionRecord {
	if record == nil {
		return nil
	}
	clone, err := record.Clone()
	if err != nil {
		panic(err)
	}
	return clone
}

func recordsFromListFixture(t *testing.T, fixture listFixture) []*dto.SessionRecord {
	records := make([]*dto.SessionRecord, 0, len(fixture.Data))
	for _, item := range fixture.Data {
		createdAt := parseRFC3339(t, item.CreatedAt)
		updatedAt := parseRFC3339(t, item.UpdatedAt)
		records = append(records, &dto.SessionRecord{
			SessionID:   item.SessionID,
			SessionType: dto.SessionTypeAgent,
			SessionData: map[string]any{
				"session_name":  item.SessionName,
				"session_state": item.SessionState,
			},
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		})
	}
	return records
}

func recordFromDetailFixture(t *testing.T, fixture detailFixture) *dto.SessionRecord {
	createdAt := parseRFC3339(t, fixture.CreatedAt)
	updatedAt := parseRFC3339(t, fixture.UpdatedAt)
	var agentID *string
	if fixture.AgentID != "" {
		id := fixture.AgentID
		agentID = &id
	}
	var userID *string
	if fixture.UserID != "" {
		id := fixture.UserID
		userID = &id
	}

	sessionData := map[string]any{
		"session_name":  fixture.SessionName,
		"session_state": fixture.SessionState,
	}
	if fixture.Metrics != nil {
		sessionData["session_metrics"] = fixture.Metrics
	}
	if fixture.ChatHistory != nil {
		sessionData["chat_history"] = fixture.ChatHistory
	}

	return &dto.SessionRecord{
		SessionID:   fixture.SessionID,
		SessionType: dto.SessionTypeAgent,
		AgentID:     agentID,
		UserID:      userID,
		SessionData: sessionData,
		Metadata:    fixture.Metadata,
		AgentData:   fixture.AgentData,
		Summary:     fixture.SessionSummary,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}
}

func recordFromCreateFixture(fixture createFixture) *dto.SessionRecord {
	createdAt := parseRFC3339NoError(fixture.CreatedAt)
	updatedAt := parseRFC3339NoError(fixture.UpdatedAt)
	var agentID *string
	if fixture.AgentID != "" {
		agentID = &fixture.AgentID
	}
	var userID *string
	if fixture.UserID != "" {
		userID = &fixture.UserID
	}
	return &dto.SessionRecord{
		SessionID:   fixture.SessionID,
		SessionType: dto.SessionTypeAgent,
		AgentID:     agentID,
		UserID:      userID,
		SessionData: map[string]any{
			"session_name":  fixture.SessionName,
			"session_state": fixture.SessionState,
		},
		Metadata:  fixture.Metadata,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}
}

func parseRFC3339(t *testing.T, value string) time.Time {
	t.Helper()
	tm, err := time.Parse(time.RFC3339, value)
	require.NoError(t, err)
	return tm
}

func parseRFC3339NoError(value string) time.Time {
	if value == "" {
		return time.Now().UTC()
	}
	tm, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Now().UTC()
	}
	return tm
}

type listFixture struct {
	Data []struct {
		SessionID    string         `json:"session_id"`
		SessionName  string         `json:"session_name"`
		SessionState map[string]any `json:"session_state"`
		CreatedAt    string         `json:"created_at"`
		UpdatedAt    string         `json:"updated_at"`
	} `json:"data"`
	Meta struct {
		Page       int `json:"page"`
		Limit      int `json:"limit"`
		TotalCount int `json:"total_count"`
		TotalPages int `json:"total_pages"`
	} `json:"meta"`
}

type detailFixture struct {
	UserID         string         `json:"user_id"`
	SessionID      string         `json:"session_id"`
	SessionName    string         `json:"session_name"`
	SessionState   map[string]any `json:"session_state"`
	SessionSummary map[string]any `json:"session_summary"`
	AgentID        string         `json:"agent_id"`
	AgentData      map[string]any `json:"agent_data"`
	Metrics        map[string]any `json:"metrics"`
	Metadata       map[string]any `json:"metadata"`
	ChatHistory    []any          `json:"chat_history"`
	CreatedAt      string         `json:"created_at"`
	UpdatedAt      string         `json:"updated_at"`
}

type createFixture struct {
	UserID       string         `json:"user_id"`
	SessionID    string         `json:"session_id"`
	SessionName  string         `json:"session_name"`
	SessionState map[string]any `json:"session_state"`
	Metadata     map[string]any `json:"metadata"`
	AgentID      string         `json:"agent_id"`
	CreatedAt    string         `json:"created_at"`
	UpdatedAt    string         `json:"updated_at"`
}
