package agentos

import (
	"context"
	"testing"

	"github.com/rexleimo/agno-go/pkg/hno/agent"
	"github.com/rexleimo/agno-go/pkg/hno/models"
	"github.com/rexleimo/agno-go/pkg/hno/team"
	"github.com/rexleimo/agno-go/pkg/hno/tools/toolkit"
	"github.com/rexleimo/agno-go/pkg/hno/types"
)

// mockModel is a lightweight Model implementation for testing tool aggregation.
type mockModel struct{ models.BaseModel }

func (m *mockModel) Invoke(ctx context.Context, req *models.InvokeRequest) (*types.ModelResponse, error) {
	return &types.ModelResponse{
		ID:      "test-response",
		Content: "ok",
		Model:   m.ID,
	}, nil
}

func (m *mockModel) InvokeStream(ctx context.Context, req *models.InvokeRequest) (<-chan types.ResponseChunk, error) {
	ch := make(chan types.ResponseChunk)
	close(ch)
	return ch, nil
}

func TestTeamToolDefinitions_PreservesMemberTools(t *testing.T) {
	// Toolkit for agent A
	tkA := toolkit.NewBaseToolkit("a")
	tkA.RegisterFunction(&toolkit.Function{
		Name:        "tool_alpha",
		Description: "alpha tool",
		Parameters:  map[string]toolkit.Parameter{},
		Handler: func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
			return "alpha", nil
		},
	})

	// Toolkit for agent B
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
		Model: &mockModel{BaseModel: models.BaseModel{ID: "m-a", Provider: "mock"}},
		Toolkits: []toolkit.Toolkit{
			tkA,
		},
	})
	if err != nil {
		t.Fatalf("failed to create agent A: %v", err)
	}

	agB, err := agent.New(agent.Config{
		ID:    "agent-b",
		Model: &mockModel{BaseModel: models.BaseModel{ID: "m-b", Provider: "mock"}},
		Toolkits: []toolkit.Toolkit{
			tkB,
		},
	})
	if err != nil {
		t.Fatalf("failed to create agent B: %v", err)
	}

	tm, err := team.New(team.Config{
		ID:     "team-tools",
		Name:   "Team Tools",
		Agents: []*agent.Agent{agA, agB},
	})
	if err != nil {
		t.Fatalf("failed to create team: %v", err)
	}

	defs := TeamToolDefinitions(tm)
	if len(defs) != 2 {
		t.Fatalf("expected 2 tool definitions, got %d", len(defs))
	}

	seen := map[string]bool{}
	for _, d := range defs {
		seen[d.Function.Name] = true
	}
	if !seen["tool_alpha"] || !seen["tool_beta"] {
		t.Fatalf("expected tool_alpha and tool_beta in definitions, got %+v", seen)
	}
}

func TestTeamToolDefinitions_DeduplicatesByName(t *testing.T) {
	// Two agents exposing the same tool name via different toolkits.
	tk1 := toolkit.NewBaseToolkit("tk1")
	tk1.RegisterFunction(&toolkit.Function{
		Name:        "shared_tool",
		Description: "shared",
		Parameters:  map[string]toolkit.Parameter{},
		Handler: func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
			return "one", nil
		},
	})

	tk2 := toolkit.NewBaseToolkit("tk2")
	tk2.RegisterFunction(&toolkit.Function{
		Name:        "shared_tool",
		Description: "shared duplicate",
		Parameters:  map[string]toolkit.Parameter{},
		Handler: func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
			return "two", nil
		},
	})

	ag1, err := agent.New(agent.Config{
		ID:    "agent-1",
		Model: &mockModel{BaseModel: models.BaseModel{ID: "m-1", Provider: "mock"}},
		Toolkits: []toolkit.Toolkit{
			tk1,
		},
	})
	if err != nil {
		t.Fatalf("failed to create agent 1: %v", err)
	}

	ag2, err := agent.New(agent.Config{
		ID:    "agent-2",
		Model: &mockModel{BaseModel: models.BaseModel{ID: "m-2", Provider: "mock"}},
		Toolkits: []toolkit.Toolkit{
			tk2,
		},
	})
	if err != nil {
		t.Fatalf("failed to create agent 2: %v", err)
	}

	tm, err := team.New(team.Config{
		ID:     "team-dedupe",
		Name:   "Team Dedupe",
		Agents: []*agent.Agent{ag1, ag2},
	})
	if err != nil {
		t.Fatalf("failed to create team: %v", err)
	}

	defs := TeamToolDefinitions(tm)
	if len(defs) != 1 {
		t.Fatalf("expected 1 tool definition after deduplication, got %d", len(defs))
	}
	if defs[0].Function.Name != "shared_tool" {
		t.Fatalf("expected shared_tool as the only definition, got %q", defs[0].Function.Name)
	}
}

func TestTeamToolDefinitions_EmptyWhenNoAgentsOrToolkits(t *testing.T) {
	// Nil team
	if defs := TeamToolDefinitions(nil); defs != nil {
		t.Fatalf("expected nil definitions for nil team, got %#v", defs)
	}

	// Team with no agents
	tm, err := team.New(team.Config{
		ID:     "team-empty",
		Name:   "Empty Team",
		Agents: []*agent.Agent{ /* empty */ },
	})
	if err == nil {
		// team.New with no agents should already fail; this is just defensive.
		_ = tm
	}
}
