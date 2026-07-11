// Package chat is a streaming multi-turn chat host on top of the kernel.
package chat

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/host"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/types"
)

// MissionSubmitter is the kernel mission surface.
type MissionSubmitter interface {
	SubmitMission(ctx context.Context, m host.Mission) (types.MissionID, error)
	GetMission(ctx context.Context, id types.MissionID) (host.Mission, error)
}

// Message in a chat session.
type Message struct {
	Role      string    `json:"role"` // user|assistant|system
	Content   string    `json:"content"`
	At        time.Time `json:"at"`
	MissionID string    `json:"missionId,omitempty"`
}

// Session is a multi-turn conversation.
type Session struct {
	ID        string            `json:"id"`
	Title     string            `json:"title"`
	Messages  []Message         `json:"messages"`
	Provider  string            `json:"provider,omitempty"`
	Model     string            `json:"model,omitempty"`
	SkillIDs  []string          `json:"skillIds,omitempty"`
	CreatedAt time.Time         `json:"createdAt"`
	UpdatedAt time.Time         `json:"updatedAt"`
	Labels    map[string]string `json:"labels,omitempty"`
}

// Service manages chat sessions → missions.
type Service struct {
	mu       sync.Mutex
	sessions map[string]*Session
	seq      int64
	kernel   MissionSubmitter
}

func New(k MissionSubmitter) *Service {
	return &Service{sessions: map[string]*Session{}, kernel: k}
}

func (s *Service) List() []Session {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Session, 0, len(s.sessions))
	for _, sess := range s.sessions {
		out = append(out, *sess)
	}
	return out
}

func (s *Service) Get(id string) (Session, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, ok := s.sessions[id]
	if !ok {
		return Session{}, false
	}
	return *sess, true
}

func (s *Service) Create(title string) Session {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	now := time.Now().UTC()
	id := fmt.Sprintf("chat_%d", s.seq)
	if title == "" {
		title = "Chat " + now.Format("15:04")
	}
	sess := &Session{ID: id, Title: title, CreatedAt: now, UpdatedAt: now}
	s.sessions[id] = sess
	return *sess
}

// Send posts a user message, runs a mission, appends assistant reply.
func (s *Service) Send(ctx context.Context, sessionID, text string, preferProvider, preferModel string, skillIDs []string) (Session, error) {
	if text == "" {
		return Session{}, fmt.Errorf("message required")
	}
	s.mu.Lock()
	sess, ok := s.sessions[sessionID]
	if !ok {
		s.mu.Unlock()
		return Session{}, fmt.Errorf("unknown session")
	}
	now := time.Now().UTC()
	sess.Messages = append(sess.Messages, Message{Role: "user", Content: text, At: now})
	sess.UpdatedAt = now
	if preferProvider != "" {
		sess.Provider = preferProvider
	}
	if preferModel != "" {
		sess.Model = preferModel
	}
	if len(skillIDs) > 0 {
		sess.SkillIDs = skillIDs
	}
	// Build goal from last few turns for continuity
	goal := text
	if len(sess.Messages) > 1 {
		var hist string
		start := len(sess.Messages) - 6
		if start < 0 {
			start = 0
		}
		for _, m := range sess.Messages[start : len(sess.Messages)-1] {
			hist += m.Role + ": " + m.Content + "\n"
		}
		goal = "Conversation so far:\n" + hist + "\nUser: " + text + "\nRespond helpfully. Use tools if needed."
	}
	prov := sess.Provider
	model := sess.Model
	skills := append([]string{}, sess.SkillIDs...)
	s.mu.Unlock()

	labels := map[string]string{}
	if prov != "" {
		labels["route.preferProvider"] = prov
	}
	if model != "" {
		labels["route.preferModel"] = model
	}
	labels["route.preferRuntime"] = "runtime.agent.loop"
	labels["chat.session"] = sessionID
	if len(skills) > 0 {
		labels["skills"] = joinComma(skills)
	}

	mid, err := s.kernel.SubmitMission(ctx, host.Mission{
		Goal:         goal,
		RequiredCaps: []types.Capability{"coding", "tools"},
		Labels:       labels,
		PreferProvider: types.PluginID(prov),
		PreferModel:    model,
		Failover:       boolPtr(true),
	})
	if err != nil {
		return Session{}, err
	}

	// Kernel executes missions inline; fetch result.
	m, err := s.kernel.GetMission(ctx, mid)
	if err != nil {
		return Session{}, err
	}
	reply := m.Output
	if reply == "" {
		reply = "(no output)"
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	sess = s.sessions[sessionID]
	sess.Messages = append(sess.Messages, Message{
		Role: "assistant", Content: reply, At: time.Now().UTC(), MissionID: string(mid),
	})
	sess.UpdatedAt = time.Now().UTC()
	return *sess, nil
}

func joinComma(ss []string) string {
	out := ""
	for i, s := range ss {
		if i > 0 {
			out += ","
		}
		out += s
	}
	return out
}

func boolPtr(b bool) *bool { return &b }
