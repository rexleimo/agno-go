package agentos

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rexleimo/agno-go/pkg/hno/session"
	"github.com/rexleimo/agno-go/pkg/hno/skills"
)

func TestOpsEndpoints(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Build a registry with one skill.
	// 构建含一个技能的注册表。
	dir := t.TempDir()
	skillDir := dir + "/web-research"
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	skillMD := `---
name: web-research
description: Research a topic on the web.
---

# Web Research
Steps here.
`
	if err := os.WriteFile(skillDir+"/SKILL.md", []byte(skillMD), 0o644); err != nil {
		t.Fatal(err)
	}

	reg, err := skills.NewRegistry(skills.NewLoader(os.DirFS(dir)), ".")
	if err != nil {
		t.Fatalf("registry: %v", err)
	}

	srv, err := NewServer(&Config{SessionStorage: session.NewMemoryStorage(), SkillsRegistry: reg})
	if err != nil {
		t.Fatalf("server: %v", err)
	}

	// 1. GET /ops/skills
	// 列出技能。
	w := performRequest(srv, http.MethodGet, "/api/v1/ops/skills", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("skills status = %d", w.Code)
	}
	var skillsResp struct {
		Skills []struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		} `json:"skills"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &skillsResp); err != nil {
		t.Fatalf("skills decode: %v", err)
	}
	if len(skillsResp.Skills) != 1 || skillsResp.Skills[0].Name != "web-research" {
		t.Errorf("skills = %+v, want 1 web-research", skillsResp.Skills)
	}

	// 2. GET /ops/observability
	// 运行状态。
	w = performRequest(srv, http.MethodGet, "/api/v1/ops/observability", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("observability status = %d", w.Code)
	}
	var obsResp struct {
		Uptime string `json:"uptime"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &obsResp); err != nil {
		t.Fatalf("observability decode: %v", err)
	}
	if obsResp.Uptime == "" {
		t.Error("uptime missing")
	}

	// 3. POST /ops/eval-runs then GET
	// 提交评估记录后再查询。
	payload := `{"model":"gpt-4o-mini","runs":10,"successes":9,"duration":"1.2s"}`
	w = performRequest(srv, http.MethodPost, "/api/v1/ops/eval-runs", []byte(payload))
	if w.Code != http.StatusCreated {
		t.Fatalf("record status = %d, body=%s", w.Code, w.Body.String())
	}
	var recResp struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &recResp); err != nil {
		t.Fatalf("record decode: %v", err)
	}
	if recResp.ID == "" {
		t.Error("record id missing")
	}

	w = performRequest(srv, http.MethodGet, "/api/v1/ops/eval-runs", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("list status = %d", w.Code)
	}
	var listResp struct {
		Runs []evalRun `json:"eval_runs"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("list decode: %v", err)
	}
	if len(listResp.Runs) != 1 || listResp.Runs[0].Model != "gpt-4o-mini" {
		t.Errorf("runs = %+v, want 1 gpt-4o-mini", listResp.Runs)
	}

	// 4. Invalid eval payload rejected.
	// 无效评估记录被拒绝。
	w = performRequest(srv, http.MethodPost, "/api/v1/ops/eval-runs", []byte(`{not json`))
	if w.Code != http.StatusBadRequest {
		t.Errorf("invalid payload status = %d, want 400", w.Code)
	}
}

func performRequest(srv *Server, method, path string, body []byte) *httptest.ResponseRecorder {
	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(method, path, bytes.NewReader(body))
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	w := httptest.NewRecorder()
	srv.router.ServeHTTP(w, req)
	return w
}
