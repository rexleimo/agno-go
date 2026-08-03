package agentos

import (
	"fmt"
	"sync"

	"github.com/rexleimo/agno-go/pkg/hno/agent"
)

// AgentRegistry manages registered agents
type AgentRegistry struct {
	mu     sync.RWMutex
	agents map[string]*agent.Agent
}

// NewAgentRegistry creates a new agent registry
func NewAgentRegistry() *AgentRegistry {
	return &AgentRegistry{
		agents: make(map[string]*agent.Agent),
	}
}

// Register registers an agent with the given ID
func (r *AgentRegistry) Register(agentID string, ag *agent.Agent) error {
	if agentID == "" {
		return fmt.Errorf("agent ID cannot be empty")
	}

	if ag == nil {
		return fmt.Errorf("agent cannot be nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if agent already exists
	if _, exists := r.agents[agentID]; exists {
		return fmt.Errorf("agent with ID '%s' already registered", agentID)
	}

	r.agents[agentID] = ag
	return nil
}

// Unregister removes an agent from the registry
func (r *AgentRegistry) Unregister(agentID string) error {
	if agentID == "" {
		return fmt.Errorf("agent ID cannot be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.agents[agentID]; !exists {
		return fmt.Errorf("agent with ID '%s' not found", agentID)
	}

	delete(r.agents, agentID)
	return nil
}

// Get retrieves an agent by ID
func (r *AgentRegistry) Get(agentID string) (*agent.Agent, error) {
	if agentID == "" {
		return nil, fmt.Errorf("agent ID cannot be empty")
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	ag, exists := r.agents[agentID]
	if !exists {
		return nil, fmt.Errorf("agent with ID '%s' not found", agentID)
	}

	return ag, nil
}

// List returns all registered agents
func (r *AgentRegistry) List() map[string]*agent.Agent {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Return a copy to prevent external modification
	result := make(map[string]*agent.Agent, len(r.agents))
	for id, ag := range r.agents {
		result[id] = ag
	}

	return result
}

// Count returns the number of registered agents
func (r *AgentRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.agents)
}

// Exists checks if an agent with the given ID is registered
func (r *AgentRegistry) Exists(agentID string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.agents[agentID]
	return exists
}

// Clear removes all agents from the registry
func (r *AgentRegistry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.agents = make(map[string]*agent.Agent)
}

// GetIDs returns a list of all registered agent IDs
func (r *AgentRegistry) GetIDs() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ids := make([]string, 0, len(r.agents))
	for id := range r.agents {
		ids = append(ids, id)
	}

	return ids
}
