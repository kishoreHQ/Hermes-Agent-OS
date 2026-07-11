// Package deploy orchestrates deployment sessions (AESP-0009 / DEP-ROLLOUT).
package deploy

import (
	"fmt"
	"sync"
	"time"

	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/types"
)

// Session is a progressive rollout session.
type Session struct {
	ID           string                 `json:"id"`
	Artifact     types.ArtifactDigest   `json:"artifactDigest"`
	Strategy     string                 `json:"strategy"` // rolling|canary|blue-green
	Status       string                 `json:"status"`   // pending|running|succeeded|failed|gated
	Gates        []string               `json:"gates,omitempty"`
	Environment  string                 `json:"environment"`
	CreatedAt    time.Time              `json:"createdAt"`
	UpdatedAt    time.Time              `json:"updatedAt"`
	Log          []string               `json:"log,omitempty"`
}

// Service manages deploy sessions.
type Service struct {
	mu   sync.Mutex
	byID map[string]*Session
	seq  int
}

func New() *Service {
	return &Service{byID: map[string]*Session{}}
}

type CreateReq struct {
	Artifact    string   `json:"artifactDigest"`
	Strategy    string   `json:"strategy"`
	Environment string   `json:"environment"`
	Gates       []string `json:"gates"`
}

func (s *Service) Create(req CreateReq) (*Session, error) {
	if req.Artifact == "" {
		return nil, fmt.Errorf("artifactDigest required")
	}
	if req.Strategy == "" {
		req.Strategy = "rolling"
	}
	if req.Environment == "" {
		req.Environment = "dev"
	}
	now := time.Now().UTC()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	sess := &Session{
		ID: fmt.Sprintf("dep_%d", s.seq), Artifact: types.ArtifactDigest(req.Artifact),
		Strategy: req.Strategy, Status: "pending", Gates: req.Gates, Environment: req.Environment,
		CreatedAt: now, UpdatedAt: now, Log: []string{"session created"},
	}
	s.byID[sess.ID] = sess
	cp := *sess
	return &cp, nil
}

// Advance moves session through gate checks (digest continuity).
func (s *Service) Advance(id string, approveGate bool) (*Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, ok := s.byID[id]
	if !ok {
		return nil, fmt.Errorf("unknown session")
	}
	switch sess.Status {
	case "pending":
		sess.Status = "running"
		sess.Log = append(sess.Log, "started "+sess.Strategy+" rollout of "+string(sess.Artifact))
	case "running":
		if len(sess.Gates) > 0 && !approveGate {
			sess.Status = "gated"
			sess.Log = append(sess.Log, "awaiting gate approval")
		} else {
			sess.Status = "succeeded"
			sess.Log = append(sess.Log, "rollout complete")
		}
	case "gated":
		if !approveGate {
			return nil, fmt.Errorf("gate not approved")
		}
		sess.Status = "succeeded"
		sess.Log = append(sess.Log, "gate approved; complete")
	default:
		return nil, fmt.Errorf("terminal status %s", sess.Status)
	}
	sess.UpdatedAt = time.Now().UTC()
	cp := *sess
	return &cp, nil
}

func (s *Service) List() []Session {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Session, 0, len(s.byID))
	for _, x := range s.byID {
		out = append(out, *x)
	}
	return out
}

func (s *Service) Get(id string) (Session, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	x, ok := s.byID[id]
	if !ok {
		return Session{}, false
	}
	return *x, true
}
