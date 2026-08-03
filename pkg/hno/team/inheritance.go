package team

import (
	"context"

	"github.com/rexleimo/agno-go/pkg/hno/agent"
	"github.com/rexleimo/agno-go/pkg/hno/run"
)

// inheritanceRecord tracks which model an agent inherited and from where.
// inheritanceRecord 记录 agent 继承的模型及其来源。
type inheritanceRecord struct {
	ModelID string
	Source  string
}

// modelScope holds the model override applied for one invocation and how to
// restore the agent afterwards.
// modelScope 保存单次调用应用的模型覆盖及调用后的恢复方式。
type modelScope struct {
	restore func()
	record  *inheritanceRecord
}

// invokeAgent runs a single agent with the team's model inheritance applied.
func (t *Team) invokeAgent(ctx context.Context, ag *agent.Agent, input string) (*agent.RunOutput, error) {
	scope := t.prepareAgentModel(ag)
	output, err := ag.Run(ctx, input)
	if scope.restore != nil {
		scope.restore()
	}
	if scope.record != nil {
		t.recordInheritance(ag.ID, scope.record)
	}
	if output != nil && len(output.Events) > 0 {
		annotateEventsWithTeam(t.ID, output.Events)
	}
	return output, err
}

func annotateEventsWithTeam(teamID string, events run.Events) {
	if teamID == "" {
		return
	}
	for _, evt := range events {
		switch e := evt.(type) {
		case *run.RunContentEvent:
			if e.TeamID == "" {
				e.TeamID = teamID
			}
		case *run.RunCompletedEvent:
			if e.TeamID == "" {
				e.TeamID = teamID
			}
		}
	}
}

func (t *Team) prepareAgentModel(ag *agent.Agent) modelScope {
	scope := modelScope{
		restore: func() {},
	}
	if ag == nil {
		return scope
	}

	original := ag.Model
	scope.restore = func() {
		ag.Model = original
	}

	if override, ok := t.modelOverrides[ag.ID]; ok && override != nil {
		ag.Model = override
		scope.record = &inheritanceRecord{
			ModelID: override.GetID(),
			Source:  "override",
		}
		return scope
	}

	if !t.inheritModel || t.sharedModel == nil {
		return scope
	}

	if _, skip := t.skipInheritance[ag.ID]; skip {
		return scope
	}

	ag.Model = t.sharedModel
	scope.record = &inheritanceRecord{
		ModelID: t.sharedModel.GetID(),
		Source:  "team",
	}
	return scope
}

func (t *Team) recordInheritance(agentID string, record *inheritanceRecord) {
	if agentID == "" || record == nil {
		return
	}
	t.inheritanceMu.Lock()
	defer t.inheritanceMu.Unlock()

	if t.inheritanceTrace == nil {
		t.inheritanceTrace = make(map[string]inheritanceRecord)
	}
	t.inheritanceTrace[agentID] = *record
}

func (t *Team) snapshotInheritance() map[string]map[string]string {
	t.inheritanceMu.Lock()
	defer t.inheritanceMu.Unlock()

	if len(t.inheritanceTrace) == 0 {
		return nil
	}

	result := make(map[string]map[string]string, len(t.inheritanceTrace))
	for agentID, record := range t.inheritanceTrace {
		result[agentID] = map[string]string{
			"model_id": record.ModelID,
			"source":   record.Source,
		}
	}
	return result
}

func (t *Team) appendInheritanceMetadata(metadata map[string]interface{}) {
	if metadata == nil {
		return
	}
	if trace := t.snapshotInheritance(); len(trace) > 0 {
		metadata["model_inheritance"] = trace
	}
}

func (t *Team) resetInheritance() {
	t.inheritanceMu.Lock()
	defer t.inheritanceMu.Unlock()
	if t.inheritanceTrace == nil {
		t.inheritanceTrace = make(map[string]inheritanceRecord)
		return
	}
	for k := range t.inheritanceTrace {
		delete(t.inheritanceTrace, k)
	}
}
