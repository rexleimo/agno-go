package contract_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rexleimo/agno-go/internal/agent"
	"github.com/rexleimo/agno-go/internal/model"
	"github.com/rexleimo/agno-go/internal/runtime"
	rtmiddleware "github.com/rexleimo/agno-go/internal/runtime/middleware"
	"github.com/rexleimo/agno-go/pkg/memory"
	"github.com/rexleimo/agno-go/pkg/providers/stub"
)

// FR-004: only API Key header is accepted; Basic/OAuth/JWT/custom auth must be rejected.
func TestAuthMiddlewareRejectsNonAPIKeySchemes(t *testing.T) {
	base := repoRoot(t)
	coverageLog := filepath.Join(base, "specs", "001-agno-agents-refactor", "artifacts", "coverage.txt")

	router := model.NewRouter()
	router.RegisterChatProvider(stub.New(agent.ProviderOpenAI, model.ProviderAvailable, nil))
	svc := runtime.NewService(memory.NewInMemoryStore(), router)

	apiKey := "test-api-key"
	server := runtime.NewServer(
		router.Statuses,
		"dev",
		svc,
		runtime.WithAPIKeyAuth(apiKey, rtmiddleware.DefaultAPIKeyHeader),
		runtime.WithTimeout(5*time.Second),
	)

	cases := []struct {
		name       string
		headers    map[string]string
		wantStatus int
		wantBody   string
	}{
		{name: "missing", headers: nil, wantStatus: http.StatusUnauthorized, wantBody: "missing API key"},
		{name: "basic", headers: map[string]string{"Authorization": "Basic abc"}, wantStatus: http.StatusUnauthorized, wantBody: "unsupported auth"},
		{name: "bearer", headers: map[string]string{"Authorization": "Bearer token"}, wantStatus: http.StatusUnauthorized, wantBody: "unsupported auth"},
		{name: "custom", headers: map[string]string{"Authorization": "Custom token"}, wantStatus: http.StatusUnauthorized, wantBody: "unsupported auth"},
		{name: "valid", headers: map[string]string{rtmiddleware.DefaultAPIKeyHeader: apiKey}, wantStatus: http.StatusCreated},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body := `{"name":"auth-agent","model":{"provider":"openai","modelId":"stub","stream":true}}`
			req := httptest.NewRequest(http.MethodPost, "/agents", bytes.NewBufferString(body))
			req.Header.Set("Content-Type", "application/json")
			for k, v := range tc.headers {
				req.Header.Set(k, v)
			}

			rr := httptest.NewRecorder()
			server.Router.ServeHTTP(rr, req)

			if rr.Code != tc.wantStatus {
				t.Fatalf("%s: status=%d body=%s", tc.name, rr.Code, rr.Body.String())
			}
			if tc.wantBody != "" && !strings.Contains(strings.ToLower(rr.Body.String()), strings.ToLower(tc.wantBody)) {
				t.Fatalf("%s: expected body to contain %q got %s", tc.name, tc.wantBody, rr.Body.String())
			}

			status := "blocked"
			if tc.wantStatus < 300 {
				status = "allowed"
			}
			_ = appendFile(coverageLog, []byte("auth_scheme="+tc.name+" status="+status+"\n"))
		})
	}
}
