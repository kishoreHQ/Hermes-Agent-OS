package deck

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/host"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/types"
)

// Routine is a scheduled mission template (K5). Fire is explicit (no background cron dep).
type Routine struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Schedule     string    `json:"schedule"` // cron-like label or @hourly / manual
	Prompt       string    `json:"prompt,omitempty"`
	RuntimeID    string    `json:"runtimeId,omitempty"`
	Capabilities []string  `json:"capabilities,omitempty"`
	Paused       bool      `json:"paused"`
	LastRunAt    time.Time `json:"lastRunAt,omitempty"`
	LastStatus   string    `json:"lastStatus,omitempty"`
	LastMission  string    `json:"lastMissionId,omitempty"`
	History      []Run     `json:"history,omitempty"`
}

type Run struct {
	At        time.Time `json:"at"`
	MissionID string    `json:"missionId"`
	Status    string    `json:"status"`
}

type CreateRoutineRequest struct {
	Name         string   `json:"name"`
	Schedule     string   `json:"schedule"`
	Prompt       string   `json:"prompt"`
	RuntimeID    string   `json:"runtime"`
	Capabilities []string `json:"capabilities"`
}

// RoutinesService stores routines and fires missions on demand.
type RoutinesService struct {
	mu       sync.Mutex
	routines map[string]*Routine
	runner   MissionRunner
}

func NewRoutines(runner MissionRunner) *RoutinesService {
	s := &RoutinesService{routines: map[string]*Routine{}, runner: runner}
	_, _ = s.Create(CreateRoutineRequest{
		Name: "Hourly health sweep", Schedule: "@hourly",
		Prompt: "Run fleet health check and log status", Capabilities: []string{"tools", "coding"},
	})
	return s
}

func (s *RoutinesService) Create(req CreateRoutineRequest) (*Routine, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("name required")
	}
	if req.Schedule == "" {
		req.Schedule = "manual"
	}
	r := &Routine{
		ID: fmt.Sprintf("rtn_%d", time.Now().UnixNano()), Name: req.Name, Schedule: req.Schedule,
		Prompt: req.Prompt, RuntimeID: req.RuntimeID, Capabilities: req.Capabilities,
	}
	s.mu.Lock()
	s.routines[r.ID] = r
	s.mu.Unlock()
	return r, nil
}

func (s *RoutinesService) List() []Routine {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Routine, 0, len(s.routines))
	for _, r := range s.routines {
		cp := *r
		cp.History = append([]Run{}, r.History...)
		out = append(out, cp)
	}
	return out
}

func (s *RoutinesService) Fire(ctx context.Context, id string) (*Routine, error) {
	s.mu.Lock()
	r, ok := s.routines[id]
	if !ok {
		s.mu.Unlock()
		return nil, fmt.Errorf("unknown routine")
	}
	if r.Paused {
		s.mu.Unlock()
		return nil, fmt.Errorf("routine paused")
	}
	prompt := r.Prompt
	if prompt == "" {
		prompt = r.Name
	}
	caps := r.Capabilities
	runtimeID := r.RuntimeID
	s.mu.Unlock()

	if s.runner == nil {
		return nil, fmt.Errorf("no mission runner")
	}
	reqCaps := make([]types.Capability, 0, len(caps))
	for _, c := range caps {
		reqCaps = append(reqCaps, types.Capability(c))
	}
	if len(reqCaps) == 0 {
		reqCaps = []types.Capability{"coding", "tools"}
	}
	labels := map[string]string{"deck.routine": id}
	if runtimeID != "" {
		labels["route.preferRuntime"] = runtimeID
	}
	mid, err := s.runner.SubmitMission(ctx, host.Mission{
		Goal: prompt, RequiredCaps: reqCaps, Labels: labels,
	})
	status := "succeeded"
	if err != nil {
		status = "failed"
	} else {
		m, _ := s.runner.GetMission(ctx, mid)
		status = string(m.State)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	r = s.routines[id]
	r.LastRunAt = time.Now().UTC()
	r.LastStatus = status
	r.LastMission = string(mid)
	r.History = append(r.History, Run{At: r.LastRunAt, MissionID: string(mid), Status: status})
	if len(r.History) > 20 {
		r.History = r.History[len(r.History)-20:]
	}
	cp := *r
	cp.History = append([]Run{}, r.History...)
	if err != nil {
		return &cp, err
	}
	return &cp, nil
}

func (s *RoutinesService) SetPaused(id string, paused bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.routines[id]
	if !ok {
		return fmt.Errorf("unknown routine")
	}
	r.Paused = paused
	return nil
}
