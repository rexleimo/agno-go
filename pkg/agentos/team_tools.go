package agentos

import (
	"github.com/rexleimo/agno-go/pkg/hno/models"
	"github.com/rexleimo/agno-go/pkg/hno/team"
	"github.com/rexleimo/agno-go/pkg/hno/tools/toolkit"
)

// TeamToolDefinitions returns the effective tool definitions available to a team.
// It aggregates tools from all member agents' toolkits and de-duplicates them
// by function name so the result can be used for OS schema/tooling purposes.
func TeamToolDefinitions(t *team.Team) []models.ToolDefinition {
	if t == nil {
		return nil
	}

	agents := t.GetAgents()
	if len(agents) == 0 {
		return nil
	}

	var result []models.ToolDefinition
	seen := make(map[string]struct{})

	for _, ag := range agents {
		if ag == nil || len(ag.Toolkits) == 0 {
			continue
		}

		defs := toolkit.ToModelToolDefinitions(ag.Toolkits)
		for _, d := range defs {
			name := d.Function.Name
			if name == "" {
				continue
			}
			if _, ok := seen[name]; ok {
				continue
			}
			seen[name] = struct{}{}
			result = append(result, d)
		}
	}

	return result
}
