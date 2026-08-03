package agentos

import (
	"fmt"
	"sync"

	"github.com/rexleimo/agno-go/pkg/hno/team"
)

// TeamRegistry manages registered teams.
type TeamRegistry struct {
	mu    sync.RWMutex
	teams map[string]*team.Team
}

// NewTeamRegistry creates a new team registry.
func NewTeamRegistry() *TeamRegistry {
	return &TeamRegistry{
		teams: make(map[string]*team.Team),
	}
}

// Register registers a team with the given ID.
func (r *TeamRegistry) Register(teamID string, tm *team.Team) error {
	if teamID == "" {
		return fmt.Errorf("team ID cannot be empty")
	}
	if tm == nil {
		return fmt.Errorf("team cannot be nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.teams[teamID]; exists {
		return fmt.Errorf("team with ID '%s' already registered", teamID)
	}

	r.teams[teamID] = tm
	return nil
}

// Get retrieves a team by ID.
func (r *TeamRegistry) Get(teamID string) (*team.Team, error) {
	if teamID == "" {
		return nil, fmt.Errorf("team ID cannot be empty")
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	tm, exists := r.teams[teamID]
	if !exists {
		return nil, fmt.Errorf("team with ID '%s' not found", teamID)
	}
	return tm, nil
}

// Exists checks if a team with the given ID is registered.
func (r *TeamRegistry) Exists(teamID string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.teams[teamID]
	return exists
}

// List returns a copy of all registered teams.
func (r *TeamRegistry) List() map[string]*team.Team {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make(map[string]*team.Team, len(r.teams))
	for id, tm := range r.teams {
		out[id] = tm
	}
	return out
}

// Clear removes all teams from the registry.
func (r *TeamRegistry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.teams = make(map[string]*team.Team)
}

