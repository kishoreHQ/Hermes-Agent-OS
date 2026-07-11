// Package a2a is agent-to-agent peer registry + task handoff (AESP-0015 / INT-A2A).
package a2a

import (
	"fmt"
	"sync"
	"time"

	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/types"
)

// Peer is a remote or local agent endpoint.
type Peer struct {
	ID           types.PluginID     `json:"id"`
	Name         string             `json:"name"`
	Endpoint     string             `json:"endpoint,omitempty"`
	Capabilities []types.Capability `json:"capabilities,omitempty"`
	Status       string             `json:"status"` // online|offline
	RegisteredAt time.Time          `json:"registeredAt"`
}

// Task is a handoff request to a peer.
type Task struct {
	ID        string          `json:"id"`
	PeerID    types.PluginID  `json:"peerId"`
	Goal      string          `json:"goal"`
	Status    string          `json:"status"` // pending|accepted|done|failed
	CreatedAt time.Time       `json:"createdAt"`
	Result    string          `json:"result,omitempty"`
}

// Registry of peers and tasks.
type Registry struct {
	mu    sync.Mutex
	peers map[types.PluginID]*Peer
	tasks map[string]*Task
	seq   int
}

func New() *Registry {
	r := &Registry{peers: map[types.PluginID]*Peer{}, tasks: map[string]*Task{}}
	_ = r.Register(Peer{
		ID: "peer.local.reviewer", Name: "Local Reviewer Peer",
		Endpoint: "local://agent.reviewer", Capabilities: []types.Capability{"reasoning", "tools"},
		Status: "online",
	})
	return r
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

// OfferTask creates a peer task (local stub accepts immediately).
func (r *Registry) OfferTask(peerID types.PluginID, goal string) (*Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.peers[peerID]; !ok {
		return nil, fmt.Errorf("unknown peer")
	}
	r.seq++
	t := &Task{
		ID: fmt.Sprintf("a2a_%d", r.seq), PeerID: peerID, Goal: goal,
		Status: "accepted", CreatedAt: time.Now().UTC(), Result: "accepted locally: " + goal,
	}
	// local peer completes immediately for fixture
	t.Status = "done"
	r.tasks[t.ID] = t
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
