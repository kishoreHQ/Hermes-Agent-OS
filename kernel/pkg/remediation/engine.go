// Package remediation is playbook-driven remediation (AESP-0012 / REM-PLAYBOOK).
package remediation

import (
	"fmt"
	"sync"
	"time"
)

// Playbook is a named remediation procedure.
type Playbook struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Trigger     string   `json:"trigger"` // e.g. provider.unhealthy
	Steps       []string `json:"steps"`
	Guardrails  []string `json:"guardrails,omitempty"`
	Enabled     bool     `json:"enabled"`
}

// Run is an execution of a playbook.
type Run struct {
	ID         string    `json:"id"`
	PlaybookID string    `json:"playbookId"`
	Status     string    `json:"status"` // ok|denied|failed
	Log        []string  `json:"log"`
	At         time.Time `json:"at"`
}

// Engine holds playbooks.
type Engine struct {
	mu        sync.Mutex
	playbooks map[string]*Playbook
	runs      []Run
	seq       int
}

func New() *Engine {
	e := &Engine{playbooks: map[string]*Playbook{}}
	_ = e.Register(Playbook{
		ID: "pb.provider.restart", Name: "Provider health remediation",
		Trigger: "provider.unhealthy",
		Steps:   []string{"check health", "rotate credential handle", "re-probe connection"},
		Guardrails: []string{"no-auto-prod-delete", "freeze-window-respect"},
		Enabled: true,
	})
	return e
}

func (e *Engine) Register(p Playbook) error {
	if p.ID == "" {
		return fmt.Errorf("id required")
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	cp := p
	e.playbooks[p.ID] = &cp
	return nil
}

func (e *Engine) List() []Playbook {
	e.mu.Lock()
	defer e.mu.Unlock()
	out := make([]Playbook, 0, len(e.playbooks))
	for _, p := range e.playbooks {
		out = append(out, *p)
	}
	return out
}

// Run executes playbook steps as an audit log (no destructive side effects by default).
func (e *Engine) Run(id string, freezeWindow bool) (Run, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	p, ok := e.playbooks[id]
	if !ok {
		return Run{}, fmt.Errorf("unknown playbook")
	}
	e.seq++
	run := Run{ID: fmt.Sprintf("rem_%d", e.seq), PlaybookID: id, At: time.Now().UTC()}
	if freezeWindow {
		run.Status = "denied"
		run.Log = []string{"denied: freeze-window active"}
		e.runs = append(e.runs, run)
		return run, fmt.Errorf("freeze-window deny")
	}
	if !p.Enabled {
		run.Status = "denied"
		run.Log = []string{"playbook disabled"}
		e.runs = append(e.runs, run)
		return run, fmt.Errorf("disabled")
	}
	for _, step := range p.Steps {
		run.Log = append(run.Log, "ok: "+step)
	}
	run.Status = "ok"
	e.runs = append(e.runs, run)
	return run, nil
}

func (e *Engine) History() []Run {
	e.mu.Lock()
	defer e.mu.Unlock()
	return append([]Run{}, e.runs...)
}
