// Package memorystore is Hermes unified memory (INV-06).
// All runtimes read/write through this surface — never vendor memory silos.
package memorystore

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/types"
)

// Kind of memory entry.
type Kind string

const (
	KindEpisodic   Kind = "episodic"
	KindSemantic   Kind = "semantic"
	KindProcedural Kind = "procedural"
	KindArtifact   Kind = "artifact"
	KindEvaluation Kind = "evaluation"
)

// Entry is a single memory record with trust and provenance.
type Entry struct {
	ID         string              `json:"id"`
	Kind       Kind                `json:"kind"`
	MissionID  types.MissionID     `json:"missionId,omitempty"`
	Content    string              `json:"content"`
	Trust      types.TrustLabel    `json:"trust"`
	Provenance map[string]string   `json:"provenance,omitempty"`
	Labels     map[string]string   `json:"labels,omitempty"`
	CreatedAt  time.Time           `json:"createdAt"`
}

// Query filters memory search.
type Query struct {
	MissionID types.MissionID
	Kind      Kind
	Text      string
	Limit     int
}

// Store is the memory plugin contract.
type Store interface {
	Write(ctx context.Context, e Entry) (Entry, error)
	Search(ctx context.Context, q Query) ([]Entry, error)
	Get(ctx context.Context, id string) (Entry, error)
}

// MemoryStore is an in-process store (dev/test).
type MemoryStore struct {
	mu   sync.Mutex
	byID map[string]Entry
	seq  int64
}

func New() *MemoryStore {
	return &MemoryStore{byID: map[string]Entry{}}
}

func (s *MemoryStore) Write(ctx context.Context, e Entry) (Entry, error) {
	if e.Content == "" {
		return Entry{}, fmt.Errorf("content required")
	}
	if e.Trust == "" {
		e.Trust = types.TrustAgent
	}
	if e.Kind == "" {
		e.Kind = KindEpisodic
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	if e.ID == "" {
		e.ID = fmt.Sprintf("mem_%d", s.seq)
	}
	e.CreatedAt = time.Now().UTC()
	s.byID[e.ID] = e
	return e, nil
}

func (s *MemoryStore) Search(ctx context.Context, q Query) ([]Entry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	limit := q.Limit
	if limit <= 0 {
		limit = 50
	}
	var out []Entry
	for _, e := range s.byID {
		if q.MissionID != "" && e.MissionID != q.MissionID {
			continue
		}
		if q.Kind != "" && e.Kind != q.Kind {
			continue
		}
		if q.Text != "" && !strings.Contains(strings.ToLower(e.Content), strings.ToLower(q.Text)) {
			continue
		}
		out = append(out, e)
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (s *MemoryStore) Get(ctx context.Context, id string) (Entry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.byID[id]
	if !ok {
		return Entry{}, fmt.Errorf("not found")
	}
	return e, nil
}

// AsMaps converts entries for ContextEnvelope.Memory.
func AsMaps(entries []Entry) []map[string]any {
	out := make([]map[string]any, 0, len(entries))
	for _, e := range entries {
		out = append(out, map[string]any{
			"id": e.ID, "kind": string(e.Kind), "content": e.Content,
			"trust": string(e.Trust), "missionId": string(e.MissionID),
			"provenance": e.Provenance, "labels": e.Labels,
		})
	}
	return out
}
