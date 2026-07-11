package deck

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/host"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/types"
)

type SessionStatus string

const (
	SessionIdle    SessionStatus = "idle"
	SessionWorking SessionStatus = "working"
	SessionError   SessionStatus = "error"
	SessionStopped SessionStatus = "stopped"
)

// Session is a live agent work surface (maps messages → missions).
type Session struct {
	ID          string        `json:"id"`
	RuntimeID   string        `json:"runtimeId"`
	ProviderID  string        `json:"providerId,omitempty"`
	Status      SessionStatus `json:"status"`
	CreatedAt   time.Time     `json:"createdAt"`
	UpdatedAt   time.Time     `json:"updatedAt"`
	LastMessage string        `json:"lastMessage,omitempty"`
	LastMission string        `json:"lastMissionId,omitempty"`
	Tokens      int64         `json:"tokens"`
	CostUSD     float64       `json:"costUsd"`
	Messages    []ChatTurn    `json:"messages,omitempty"`
}

type ChatTurn struct {
	Role    string    `json:"role"`
	Content string    `json:"content"`
	At      time.Time `json:"at"`
}

type CreateSessionRequest struct {
	RuntimeID  string   `json:"runtime"`
	ProviderID string   `json:"provider,omitempty"`
	Caps       []string `json:"capabilities,omitempty"`
}

// MissionRunner submits missions (kernel).
type MissionRunner interface {
	SubmitMission(ctx context.Context, m host.Mission) (types.MissionID, error)
	GetMission(ctx context.Context, id types.MissionID) (host.Mission, error)
}

// SessionsService is K3 live sessions.
type SessionsService struct {
	mu       sync.Mutex
	sessions map[string]*Session
	runner   MissionRunner
}

func NewSessions(runner MissionRunner) *SessionsService {
	return &SessionsService{sessions: map[string]*Session{}, runner: runner}
}

func (s *SessionsService) List() []Session {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Session, 0, len(s.sessions))
	for _, sess := range s.sessions {
		cp := *sess
		cp.Messages = append([]ChatTurn{}, sess.Messages...)
		out = append(out, cp)
	}
	return out
}

func (s *SessionsService) Get(id string) (Session, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, ok := s.sessions[id]
	if !ok {
		return Session{}, false
	}
	cp := *sess
	cp.Messages = append([]ChatTurn{}, sess.Messages...)
	return cp, true
}

func (s *SessionsService) Create(ctx context.Context, req CreateSessionRequest) (*Session, error) {
	if req.RuntimeID == "" {
		req.RuntimeID = "runtime.example.echo"
	}
	now := time.Now().UTC()
	sess := &Session{
		ID: fmt.Sprintf("sess_%d", now.UnixNano()), RuntimeID: req.RuntimeID, ProviderID: req.ProviderID,
		Status: SessionIdle, CreatedAt: now, UpdatedAt: now,
	}
	s.mu.Lock()
	s.sessions[sess.ID] = sess
	s.mu.Unlock()
	return sess, nil
}

func (s *SessionsService) Message(ctx context.Context, id, text string) (*Session, error) {
	if text == "" {
		return nil, fmt.Errorf("text required")
	}
	s.mu.Lock()
	sess, ok := s.sessions[id]
	if !ok {
		s.mu.Unlock()
		return nil, fmt.Errorf("unknown session")
	}
	sess.Status = SessionWorking
	sess.LastMessage = text
	sess.UpdatedAt = time.Now().UTC()
	sess.Messages = append(sess.Messages, ChatTurn{Role: "user", Content: text, At: time.Now().UTC()})
	runtimeID := sess.RuntimeID
	providerID := sess.ProviderID
	s.mu.Unlock()

	if s.runner == nil {
		return nil, fmt.Errorf("no mission runner")
	}
	labels := map[string]string{"route.preferRuntime": runtimeID, "deck.session": id}
	if providerID != "" {
		labels["route.preferProvider"] = providerID
	}
	mid, err := s.runner.SubmitMission(ctx, host.Mission{
		Goal: text, RequiredCaps: []types.Capability{"coding", "tools"}, Labels: labels,
	})
	s.mu.Lock()
	defer s.mu.Unlock()
	sess = s.sessions[id]
	if err != nil {
		sess.Status = SessionError
		sess.Messages = append(sess.Messages, ChatTurn{Role: "assistant", Content: err.Error(), At: time.Now().UTC()})
		return sess, err
	}
	m, _ := s.runner.GetMission(ctx, mid)
	sess.LastMission = string(mid)
	sess.CostUSD += m.CostUSD
	out := m.Output
	if out == "" {
		out = string(m.State)
	}
	sess.Messages = append(sess.Messages, ChatTurn{Role: "assistant", Content: out, At: time.Now().UTC()})
	if m.State == host.StateFailed {
		sess.Status = SessionError
	} else {
		sess.Status = SessionIdle
	}
	sess.UpdatedAt = time.Now().UTC()
	cp := *sess
	cp.Messages = append([]ChatTurn{}, sess.Messages...)
	return &cp, nil
}

func (s *SessionsService) Stop(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, ok := s.sessions[id]
	if !ok {
		return fmt.Errorf("unknown session")
	}
	sess.Status = SessionStopped
	sess.UpdatedAt = time.Now().UTC()
	return nil
}
