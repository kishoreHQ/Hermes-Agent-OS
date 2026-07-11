// Package scheduler runs cron-like agent jobs as missions.
package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/host"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/types"
)

// Job is a scheduled mission template.
type Job struct {
	ID             string            `json:"id"`
	Name           string            `json:"name"`
	Goal           string            `json:"goal"`
	IntervalSec    int               `json:"intervalSec"`
	PreferProvider string            `json:"preferProvider,omitempty"`
	PreferModel    string            `json:"preferModel,omitempty"`
	SkillIDs       []string          `json:"skillIds,omitempty"`
	Enabled        bool              `json:"enabled"`
	LastRunAt      *time.Time        `json:"lastRunAt,omitempty"`
	LastMissionID  string            `json:"lastMissionId,omitempty"`
	Labels         map[string]string `json:"labels,omitempty"`
}

type submitter interface {
	SubmitMission(ctx context.Context, m host.Mission) (types.MissionID, error)
}

// Service owns jobs and a ticker loop.
type Service struct {
	mu     sync.Mutex
	jobs   map[string]*Job
	seq    int64
	kernel submitter
	stop   chan struct{}
}

func New(k submitter) *Service {
	return &Service{jobs: map[string]*Job{}, kernel: k, stop: make(chan struct{})}
}

func (s *Service) List() []Job {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Job, 0, len(s.jobs))
	for _, j := range s.jobs {
		out = append(out, *j)
	}
	return out
}

func (s *Service) Upsert(j Job) Job {
	s.mu.Lock()
	defer s.mu.Unlock()
	if j.ID == "" {
		s.seq++
		j.ID = fmt.Sprintf("job_%d", s.seq)
	}
	if j.IntervalSec <= 0 {
		j.IntervalSec = 3600
	}
	cp := j
	s.jobs[j.ID] = &cp
	return cp
}

func (s *Service) Delete(id string) {
	s.mu.Lock()
	delete(s.jobs, id)
	s.mu.Unlock()
}

// Start begins background ticking (every 15s).
func (s *Service) Start() {
	go func() {
		t := time.NewTicker(15 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-s.stop:
				return
			case <-t.C:
				s.tick()
			}
		}
	}()
}

func (s *Service) Stop() { close(s.stop) }

func (s *Service) tick() {
	s.mu.Lock()
	var due []*Job
	now := time.Now().UTC()
	for _, j := range s.jobs {
		if !j.Enabled {
			continue
		}
		if j.LastRunAt == nil || now.Sub(*j.LastRunAt) >= time.Duration(j.IntervalSec)*time.Second {
			due = append(due, j)
		}
	}
	s.mu.Unlock()

	for _, j := range due {
		labels := map[string]string{"scheduler.job": j.ID, "route.preferRuntime": "runtime.agent.loop"}
		if j.PreferProvider != "" {
			labels["route.preferProvider"] = j.PreferProvider
		}
		if j.PreferModel != "" {
			labels["route.preferModel"] = j.PreferModel
		}
		mid, err := s.kernel.SubmitMission(context.Background(), host.Mission{
			Goal: j.Goal, Name: j.Name,
			RequiredCaps:   []types.Capability{"coding", "tools"},
			Labels:         labels,
			PreferProvider: types.PluginID(j.PreferProvider),
			PreferModel:    j.PreferModel,
		})
		s.mu.Lock()
		if jj, ok := s.jobs[j.ID]; ok {
			t := time.Now().UTC()
			jj.LastRunAt = &t
			if err == nil {
				jj.LastMissionID = string(mid)
			}
		}
		s.mu.Unlock()
	}
}

// RunNow forces a job immediately.
func (s *Service) RunNow(id string) (types.MissionID, error) {
	s.mu.Lock()
	j, ok := s.jobs[id]
	if !ok {
		s.mu.Unlock()
		return "", fmt.Errorf("unknown job")
	}
	goal, name, prov, model := j.Goal, j.Name, j.PreferProvider, j.PreferModel
	s.mu.Unlock()
	mid, err := s.kernel.SubmitMission(context.Background(), host.Mission{
		Goal: goal, Name: name,
		RequiredCaps:   []types.Capability{"coding", "tools"},
		PreferProvider: types.PluginID(prov),
		PreferModel:    model,
		Labels:         map[string]string{"scheduler.job": id, "route.preferRuntime": "runtime.agent.loop"},
	})
	if err != nil {
		return "", err
	}
	s.mu.Lock()
	if jj, ok := s.jobs[id]; ok {
		t := time.Now().UTC()
		jj.LastRunAt = &t
		jj.LastMissionID = string(mid)
	}
	s.mu.Unlock()
	return mid, nil
}
