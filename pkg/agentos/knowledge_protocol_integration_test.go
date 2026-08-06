package agentos

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
)

type protocolDocument struct {
	ID       string
	Content  string
	Metadata map[string]interface{}
}

type chromaProtocolFixture struct {
	t            *testing.T
	mu           sync.Mutex
	documents    []protocolDocument
	lastQuery    map[string]interface{}
	collectionID string
}

func newChromaProtocolServer(t *testing.T) (*httptest.Server, *chromaProtocolFixture) {
	t.Helper()
	fixture := &chromaProtocolFixture{
		t:            t,
		collectionID: "8ecf0f7e-e806-47f8-96a1-4732ef42359e",
	}
	return httptest.NewServer(http.HandlerFunc(fixture.serveHTTP)), fixture
}

func (f *chromaProtocolFixture) serveHTTP(w http.ResponseWriter, r *http.Request) {
	const collectionPath = "/api/v2/tenants/tenant_alpha/databases/knowledge_db/collections"
	collectionIDPath := collectionPath + "/8ecf0f7e-e806-47f8-96a1-4732ef42359e"

	w.Header().Set("Content-Type", "application/json")
	switch {
	case r.Method == http.MethodPost && r.URL.Path == collectionPath:
		var request map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			f.t.Errorf("decode collection request: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if request["name"] != "physical_docs" || request["get_or_create"] != true {
			f.t.Errorf("unexpected collection request: %#v", request)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"id":        f.collectionID,
			"name":      "physical_docs",
			"tenant":    "tenant_alpha",
			"database":  "knowledge_db",
			"dimension": 1536,
		})
	case r.Method == http.MethodGet && r.URL.Path == "/api/v2/pre-flight-checks":
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"max_batch_size":           1000,
			"supports_base64_encoding": false,
		})
	case r.Method == http.MethodPost && r.URL.Path == collectionIDPath+"/add":
		var request struct {
			IDs       []string                 `json:"ids"`
			Documents []string                 `json:"documents"`
			Metadatas []map[string]interface{} `json:"metadatas"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			f.t.Errorf("decode add request: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if len(request.IDs) != len(request.Documents) {
			f.t.Errorf("ids/documents mismatch: %#v", request)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		f.mu.Lock()
		for index, id := range request.IDs {
			metadata := map[string]interface{}{}
			if index < len(request.Metadatas) && request.Metadatas[index] != nil {
				metadata = request.Metadatas[index]
			}
			f.documents = append(f.documents, protocolDocument{
				ID:       id,
				Content:  request.Documents[index],
				Metadata: metadata,
			})
		}
		f.mu.Unlock()
		writeJSON(w, http.StatusOK, map[string]interface{}{})
	case r.Method == http.MethodGet && r.URL.Path == collectionIDPath+"/count":
		f.mu.Lock()
		count := len(f.documents)
		f.mu.Unlock()
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(strconv.Itoa(count)))
	case r.Method == http.MethodPost && r.URL.Path == collectionIDPath+"/query":
		var request map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			f.t.Errorf("decode query request: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		f.mu.Lock()
		f.lastQuery = request
		documents := append([]protocolDocument(nil), f.documents...)
		f.mu.Unlock()

		ids := make([]string, 0, len(documents))
		contents := make([]string, 0, len(documents))
		metadatas := make([]map[string]interface{}, 0, len(documents))
		distances := make([]float64, 0, len(documents))
		for _, document := range documents {
			ids = append(ids, document.ID)
			contents = append(contents, document.Content)
			metadatas = append(metadatas, document.Metadata)
			distances = append(distances, 0)
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"ids":       [][]string{ids},
			"documents": [][]string{contents},
			"metadatas": [][]map[string]interface{}{metadatas},
			"distances": [][]float64{distances},
		})
	default:
		f.t.Errorf("unexpected Chroma request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}
}

func newEmbeddingProtocolServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/embeddings" {
			t.Errorf("unexpected embedding request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		var request struct {
			Input []string `json:"input"`
			Model string   `json:"model"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("decode embedding request: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		data := make([]map[string]interface{}, len(request.Input))
		for index := range request.Input {
			vector := make([]float32, 1536)
			vector[index%len(vector)] = 1
			data[index] = map[string]interface{}{
				"object":    "embedding",
				"index":     index,
				"embedding": vector,
			}
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"object": "list",
			"data":   data,
			"model":  request.Model,
			"usage":  map[string]int{"prompt_tokens": len(data), "total_tokens": len(data)},
		})
	}))
}

func writeJSON(w http.ResponseWriter, status int, value interface{}) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func TestKnowledgeAPIChromaProtocolEndToEnd(t *testing.T) {
	chromaServer, chromaFixture := newChromaProtocolServer(t)
	defer chromaServer.Close()
	embeddingServer := newEmbeddingProtocolServer(t)
	defer embeddingServer.Close()

	server, err := NewServer(&Config{
		VectorDBConfig: &VectorDBConfig{
			Type:           "chromadb",
			BaseURL:        chromaServer.URL,
			CollectionName: "physical_docs",
			Database:       "knowledge_db",
			Tenant:         "tenant_alpha",
		},
		EmbeddingConfig: &EmbeddingConfig{
			Provider: "openai",
			APIKey:   "test-key",
			Model:    "text-embedding-3-small",
			BaseURL:  embeddingServer.URL + "/v1",
		},
		KnowledgeAPI: &KnowledgeAPIOptions{
			EnableHealth:       true,
			AllowedCollections: []string{"team_docs"},
		},
	})
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	defer func() { _ = server.Shutdown(t.Context()) }()

	ingestBody := bytes.NewBufferString(`{"content":"Knowledge is stored in Chroma.","collection_name":"team_docs"}`)
	ingestReq := httptest.NewRequest(http.MethodPost, "/api/v1/knowledge/content", ingestBody)
	ingestReq.Header.Set("Content-Type", "application/json")
	ingestW := httptest.NewRecorder()
	server.router.ServeHTTP(ingestW, ingestReq)
	if ingestW.Code != http.StatusCreated {
		t.Fatalf("ingestion status = %d; body=%s", ingestW.Code, ingestW.Body.String())
	}

	healthReq := httptest.NewRequest(http.MethodGet, "/api/v1/knowledge/health", nil)
	healthW := httptest.NewRecorder()
	server.router.ServeHTTP(healthW, healthReq)
	if healthW.Code != http.StatusOK {
		t.Fatalf("health status = %d; body=%s", healthW.Code, healthW.Body.String())
	}

	searchBody := bytes.NewBufferString(`{"query":"Where is knowledge stored?","collection_name":"team_docs"}`)
	searchReq := httptest.NewRequest(http.MethodPost, "/api/v1/knowledge/search", searchBody)
	searchReq.Header.Set("Content-Type", "application/json")
	searchW := httptest.NewRecorder()
	server.router.ServeHTTP(searchW, searchReq)
	if searchW.Code != http.StatusOK {
		t.Fatalf("search status = %d; body=%s", searchW.Code, searchW.Body.String())
	}

	var response VectorSearchResponse
	if err := json.Unmarshal(searchW.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode search response: %v", err)
	}
	if len(response.Results) != 1 || response.Results[0].Content != "Knowledge is stored in Chroma." {
		t.Fatalf("unexpected search response: %#v", response.Results)
	}

	chromaFixture.mu.Lock()
	defer chromaFixture.mu.Unlock()
	where, ok := chromaFixture.lastQuery["where"].(map[string]interface{})
	predicate, hasPredicate := where["collection_name"].(map[string]interface{})
	if !ok || !hasPredicate || predicate["$eq"] != "team_docs" {
		t.Fatalf("collection filter was not forwarded: %#v", where)
	}
}
