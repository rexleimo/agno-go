package agentos

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rexleimo/agno-go/pkg/hno/agent"
	"github.com/rexleimo/agno-go/pkg/hno/models"
	"github.com/rexleimo/agno-go/pkg/hno/team"
	"github.com/rexleimo/agno-go/pkg/hno/tools/toolkit"
)

func TestHandleTeamTools_ReturnsAggregatedTools(t *testing.T) {
	server, err := NewServer(nil)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	// Construct a team with two agents, each exposing a distinct tool.
	tkA := toolkit.NewBaseToolkit("a")
	tkA.RegisterFunction(&toolkit.Function{
		Name:        "tool_alpha",
		Description: "alpha tool",
		Parameters:  map[string]toolkit.Parameter{},
		Handler: func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
			return "alpha", nil
		},
	})

	tkB := toolkit.NewBaseToolkit("b")
	tkB.RegisterFunction(&toolkit.Function{
		Name:        "tool_beta",
		Description: "beta tool",
		Parameters:  map[string]toolkit.Parameter{},
		Handler: func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
			return "beta", nil
		},
	})

	agA, err := agent.New(agent.Config{
		ID:    "agent-a",
		Model: &simpleModel{BaseModel: models.BaseModel{ID: "m-a", Provider: "mock"}},
		Toolkits: []toolkit.Toolkit{
			tkA,
		},
	})
	if err != nil {
		t.Fatalf("failed to create agent A: %v", err)
	}

	agB, err := agent.New(agent.Config{
		ID:    "agent-b",
		Model: &simpleModel{BaseModel: models.BaseModel{ID: "m-b", Provider: "mock"}},
		Toolkits: []toolkit.Toolkit{
			tkB,
		},
	})
	if err != nil {
		t.Fatalf("failed to create agent B: %v", err)
	}

	tm, err := team.New(team.Config{
		ID:     "team-http",
		Name:   "Team HTTP",
		Agents: []*agent.Agent{agA, agB},
	})
	if err != nil {
		t.Fatalf("failed to create team: %v", err)
	}

	if err := server.RegisterTeam("team-http", tm); err != nil {
		t.Fatalf("failed to register team: %v", err)
	}

	req, _ := http.NewRequest("GET", "/api/v1/teams/team-http/tools", nil)
	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp TeamToolsResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp.TeamID != "team-http" {
		t.Fatalf("TeamID = %q, want %q", resp.TeamID, "team-http")
	}
	if len(resp.Tools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(resp.Tools))
	}
	seen := map[string]bool{}
	for _, d := range resp.Tools {
		seen[d.Function.Name] = true
	}
	if !seen["tool_alpha"] || !seen["tool_beta"] {
		t.Fatalf("expected tool_alpha and tool_beta, got %+v", seen)
	}
}

func TestHandleTeamTools_NotFound(t *testing.T) {
	server, err := NewServer(nil)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	req, _ := http.NewRequest("GET", "/api/v1/teams/unknown-team/tools", nil)
	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
	}

	var resp ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal error response: %v", err)
	}
	if resp.Code != "TEAM_NOT_FOUND" {
		t.Fatalf("error code = %q, want TEAM_NOT_FOUND", resp.Code)
	}
}
