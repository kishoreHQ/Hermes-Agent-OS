// Package approval provides HITL approval gates for dangerous tools / assist mode.
package approval

import (
	"fmt"
	"sync"
	"time"
)

// Request is a pending approval.
type Request struct {
	ID        string         `json:"id"`
	MissionID string         `json:"missionId,omitempty"`
	ToolID    string         `json:"toolId,omitempty"`
	Reason    string         `json:"reason"`
	Input     map[string]any `json:"input,omitempty"`
	Status    string         `json:"status"` // pending|approved|denied
	CreatedAt time.Time      `json:"createdAt"`
	ResolvedAt *time.Time    `json:"resolvedAt,omitempty"`
}

// Gate holds pending approvals.
type Gate struct {
	mu   sync.Mutex
	byID map[string]*Request
	seq  int64
}

func New() *Gate {
	return &Gate{byID: map[string]*Request{}}
}

func (g *Gate) Request(missionID, toolID, reason string, input map[string]any) Request {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.seq++
	id := fmt.Sprintf("appr_%d", g.seq)
	r := &Request{
		ID: id, MissionID: missionID, ToolID: toolID, Reason: reason,
		Input: input, Status: "pending", CreatedAt: time.Now().UTC(),
	}
	g.byID[id] = r
	return *r
}

func (g *Gate) List(status string) []Request {
	g.mu.Lock()
	defer g.mu.Unlock()
	var out []Request
	for _, r := range g.byID {
		if status != "" && r.Status != status {
			continue
		}
		out = append(out, *r)
	}
	return out
}

func (g *Gate) Resolve(id, decision string) (Request, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	r, ok := g.byID[id]
	if !ok {
		return Request{}, fmt.Errorf("unknown approval")
	}
	if r.Status != "pending" {
		return *r, fmt.Errorf("already resolved")
	}
	switch decision {
	case "approved", "approve":
		r.Status = "approved"
	case "denied", "deny":
		r.Status = "denied"
	default:
		return Request{}, fmt.Errorf("decision must be approve|deny")
	}
	now := time.Now().UTC()
	r.ResolvedAt = &now
	return *r, nil
}

// IsDangerous returns true for tools that may need HITL in assist mode.
func IsDangerous(toolID string) bool {
	switch toolID {
	case "shell.exec", "fs.write", "http.request":
		return true
	}
	return false
}
