// Package a2a is agent-to-agent peer registry + task handoff (AESP-0015 / INT-A2A).
package a2a

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/host"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/types"
)

// Peer is a remote or local agent endpoint.
type Peer struct {
	ID           types.PluginID     `json:"id"`
	Name         string             `json:"name"`
	Endpoint     string             `json:"endpoint,omitempty"`
	AgentID      types.PluginID     `json:"agentId,omitempty"` // maps to agentregistry
	Capabilities []types.Capability `json:"capabilities,omitempty"`
	Status       string             `json:"status"` // online|offline
	RegisteredAt time.Time          `json:"registeredAt"`
}

// Task is a handoff request to a peer.
type Task struct {
	ID        string         `json:"id"`
	PeerID    types.PluginID `json:"peerId"`
	Goal      string         `json:"goal"`
	Status    string         `json:"status"` // pending|accepted|done|failed
	MissionID string         `json:"missionId,omitempty"`
	CreatedAt time.Time      `json:"createdAt"`
	Result    string         `json:"result,omitempty"`
}

// MissionRunner optionally executes peer work as a Hermes mission (real multi-agent path).
type MissionRunner interface {
	SubmitMission(ctx context.Context, m host.Mission) (types.MissionID, error)
	GetMission(ctx context.Context, id types.MissionID) (host.Mission, error)
}

// Registry of peers and tasks.
type Registry struct {
	mu     sync.Mutex
	peers  map[types.PluginID]*Peer
	tasks  map[string]*Task
	seq    int
	runner MissionRunner
}

func New() *Registry {
	r := &Registry{peers: map[types.PluginID]*Peer{}, tasks: map[string]*Task{}}
	_ = r.Register(Peer{
		ID: "peer.local.builder", Name: "Local Builder Peer", AgentID: "agent.default",
		Endpoint: "local://agent.default", Capabilities: []types.Capability{"coding", "tools"},
		Status: "online",
	})
	_ = r.Register(Peer{
		ID: "peer.local.reviewer", Name: "Local Reviewer Peer", AgentID: "agent.reviewer",
		// Caps must be satisfiable by registered providers (capability routing)
		Endpoint: "local://agent.reviewer", Capabilities: []types.Capability{"coding", "tools"},
		Status: "online",
	})
	return r
}

// SetRunner wires kernel mission execution for peer tasks.
func (r *Registry) SetRunner(runner MissionRunner) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.runner = runner
}

func (r *Registry) Register(p Peer) error {
	if p.ID == "" {
		return fmt.Errorf("peer id required")
	}
	if p.RegisteredAt.IsZero() {
		p.RegisteredAt = time.Now().UTC()
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := p
	r.peers[p.ID] = &cp
	return nil
}

func (r *Registry) List() []Peer {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]Peer, 0, len(r.peers))
	for _, p := range r.peers {
		out = append(out, *p)
	}
	return out
}

// OfferTask creates a peer task; if runner is set, executes a real mission under that peer's agent.
func (r *Registry) OfferTask(peerID types.PluginID, goal string) (*Task, error) {
	return r.OfferTaskCtx(context.Background(), peerID, goal)
}

// OfferTaskCtx is context-aware multi-agent handoff.
func (r *Registry) OfferTaskCtx(ctx context.Context, peerID types.PluginID, goal string) (*Task, error) {
	r.mu.Lock()
	p, ok := r.peers[peerID]
	if !ok {
		r.mu.Unlock()
		return nil, fmt.Errorf("unknown peer")
	}
	peer := *p
	runner := r.runner
	r.seq++
	id := fmt.Sprintf("a2a_%d", r.seq)
	r.mu.Unlock()

	t := &Task{
		ID: id, PeerID: peerID, Goal: goal,
		Status: "accepted", CreatedAt: time.Now().UTC(),
	}

	if runner != nil {
		caps := peer.Capabilities
		if len(caps) == 0 {
			caps = []types.Capability{"coding", "tools"}
		}
		mid, err := runner.SubmitMission(ctx, host.Mission{
			Goal:         goal,
			RequiredCaps: caps,
			Labels: map[string]string{
				"a2a.peerId":  string(peerID),
				"a2a.agentId": string(peer.AgentID),
				"a2a.taskId":  id,
			},
		})
		if err != nil {
			t.Status = "failed"
			t.Result = err.Error()
			r.mu.Lock()
			r.tasks[t.ID] = t
			r.mu.Unlock()
			return t, err
		}
		m, _ := runner.GetMission(ctx, mid)
		t.MissionID = string(mid)
		t.Result = m.Output
		if m.State == host.StateFailed {
			t.Status = "failed"
		} else {
			t.Status = "done"
		}
	} else {
		t.Status = "done"
		t.Result = "accepted locally (no runner): " + goal
	}

	r.mu.Lock()
	r.tasks[t.ID] = t
	r.mu.Unlock()
	cp := *t
	return &cp, nil
}

func (r *Registry) Tasks() []Task {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]Task, 0, len(r.tasks))
	for _, t := range r.tasks {
		out = append(out, *t)
	}
	return out
}
