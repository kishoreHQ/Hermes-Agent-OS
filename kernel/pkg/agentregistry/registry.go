// Package agentregistry is agent principal + roles (AESP-0002 / CORE-ROLES).
package agentregistry

import (
	"fmt"
	"sync"
	"time"

	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/types"
)

// Agent is a registered principal.
type Agent struct {
	ID           types.PluginID     `json:"id"`
	Name         string             `json:"name"`
	Roles        []string           `json:"roles"`
	Capabilities []types.Capability `json:"capabilities,omitempty"`
	Enabled      bool               `json:"enabled"`
	CreatedAt    time.Time          `json:"createdAt"`
}

// Registry of agents and role bindings.
type Registry struct {
	mu     sync.RWMutex
	agents map[types.PluginID]*Agent
}

func New() *Registry {
	r := &Registry{agents: map[types.PluginID]*Agent{}}
	// Seed default principals
	_ = r.Register(Agent{
		ID: "agent.default", Name: "Default Builder",
		Roles: []string{"operator", "builder"}, Capabilities: []types.Capability{"coding", "tools"}, Enabled: true,
	})
	_ = r.Register(Agent{
		ID: "agent.reviewer", Name: "Reviewer",
		Roles: []string{"reviewer"}, Capabilities: []types.Capability{"reasoning", "tools"}, Enabled: true,
	})
	return r
}

func (r *Registry) Register(a Agent) error {
	if a.ID == "" {
		return fmt.Errorf("agent id required")
	}
	if a.CreatedAt.IsZero() {
		a.CreatedAt = time.Now().UTC()
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := a
	r.agents[a.ID] = &cp
	return nil
}

func (r *Registry) Get(id types.PluginID) (Agent, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	a, ok := r.agents[id]
	if !ok {
		return Agent{}, false
	}
	return *a, true
}

func (r *Registry) List() []Agent {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Agent, 0, len(r.agents))
	for _, a := range r.agents {
		out = append(out, *a)
	}
	return out
}

// HasRole reports whether agent has role.
func (r *Registry) HasRole(id types.PluginID, role string) bool {
	a, ok := r.Get(id)
	if !ok {
		return false
	}
	for _, x := range a.Roles {
		if x == role {
			return true
		}
	}
	return false
}
