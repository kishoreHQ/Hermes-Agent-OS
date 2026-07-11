// Package planner stores versioned plan artifacts (AESP-0015 / INT-PLAN).
package planner

import (
	"fmt"
	"sync"
	"time"

	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/types"
)

// Step is one planned work unit.
type Step struct {
	ID           string             `json:"id"`
	Description  string             `json:"description"`
	AgentID      types.PluginID     `json:"agentId,omitempty"`
	Capabilities []types.Capability `json:"capabilities,omitempty"`
	DependsOn    []string           `json:"dependsOn,omitempty"`
}

// Plan is a versioned multi-step plan.
type Plan struct {
	ID        string            `json:"id"`
	MissionID types.MissionID   `json:"missionId,omitempty"`
	Version   int               `json:"version"`
	Goal      string            `json:"goal"`
	Steps     []Step            `json:"steps"`
	Status    string            `json:"status"` // draft|active|completed|failed
	Labels    map[string]string `json:"labels,omitempty"`
	CreatedAt time.Time         `json:"createdAt"`
	UpdatedAt time.Time         `json:"updatedAt"`
}

// Store of plans.
type Store struct {
	mu    sync.Mutex
	byID  map[string]*Plan
	seq   int
}

func New() *Store {
	return &Store{byID: map[string]*Plan{}}
}

// Create builds a simple sequential plan from goal (or explicit steps).
func (s *Store) Create(mission types.MissionID, goal string, steps []Step) (*Plan, error) {
	if goal == "" {
		return nil, fmt.Errorf("goal required")
	}
	if len(steps) == 0 {
		steps = []Step{
			{ID: "s1", Description: "Analyze: " + goal, Capabilities: []types.Capability{"coding"}},
			{ID: "s2", Description: "Execute: " + goal, Capabilities: []types.Capability{"coding", "tools"}, DependsOn: []string{"s1"}},
			{ID: "s3", Description: "Verify: " + goal, Capabilities: []types.Capability{"tools"}, DependsOn: []string{"s2"}},
		}
	}
	now := time.Now().UTC()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	p := &Plan{
		ID: fmt.Sprintf("plan_%d", s.seq), MissionID: mission, Version: 1, Goal: goal,
		Steps: steps, Status: "active", CreatedAt: now, UpdatedAt: now,
	}
	s.byID[p.ID] = p
	return clone(p), nil
}

func (s *Store) Get(id string) (*Plan, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.byID[id]
	if !ok {
		return nil, false
	}
	return clone(p), true
}

func (s *Store) List(mission types.MissionID) []Plan {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []Plan
	for _, p := range s.byID {
		if mission == "" || p.MissionID == mission {
			out = append(out, *clone(p))
		}
	}
	return out
}

func (s *Store) BumpVersion(id string, steps []Step) (*Plan, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.byID[id]
	if !ok {
		return nil, fmt.Errorf("unknown plan")
	}
	p.Version++
	if len(steps) > 0 {
		p.Steps = steps
	}
	p.UpdatedAt = time.Now().UTC()
	return clone(p), nil
}

func (s *Store) SetStatus(id, status string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.byID[id]
	if !ok {
		return fmt.Errorf("unknown plan")
	}
	p.Status = status
	p.UpdatedAt = time.Now().UTC()
	return nil
}

func clone(p *Plan) *Plan {
	cp := *p
	cp.Steps = append([]Step{}, p.Steps...)
	return &cp
}
