// Package workflow orchestrates multi-step / multi-agent plans (AESP-0005 / WF-ORCH).
package workflow

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/host"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/planner"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/types"
)

// Runner submits child missions for plan steps.
type Runner interface {
	SubmitMission(ctx context.Context, m host.Mission) (types.MissionID, error)
	GetMission(ctx context.Context, id types.MissionID) (host.Mission, error)
}

// RunResult is orchestration outcome.
type RunResult struct {
	PlanID       string              `json:"planId"`
	Status       string              `json:"status"`
	StepMissions map[string]string   `json:"stepMissions"`
	Errors       []string            `json:"errors,omitempty"`
	StartedAt    time.Time           `json:"startedAt"`
	FinishedAt   time.Time           `json:"finishedAt"`
}

// Orchestrator executes plan steps respecting dependsOn (DAG, sequential when linear).
type Orchestrator struct {
	mu     sync.Mutex
	plans  *planner.Store
	runner Runner
	runs   []RunResult
}

func New(plans *planner.Store, runner Runner) *Orchestrator {
	return &Orchestrator{plans: plans, runner: runner}
}

// RunPlan executes all steps when dependencies are satisfied.
func (o *Orchestrator) RunPlan(ctx context.Context, planID string) (RunResult, error) {
	p, ok := o.plans.Get(planID)
	if !ok {
		return RunResult{}, fmt.Errorf("unknown plan")
	}
	res := RunResult{
		PlanID: planID, Status: "running", StepMissions: map[string]string{},
		StartedAt: time.Now().UTC(),
	}
	done := map[string]bool{}
	remaining := append([]planner.Step{}, p.Steps...)

	for len(remaining) > 0 {
		progress := false
		var next []planner.Step
		for _, st := range remaining {
			ready := true
			for _, d := range st.DependsOn {
				if !done[d] {
					ready = false
					break
				}
			}
			if !ready {
				next = append(next, st)
				continue
			}
			progress = true
			caps := st.Capabilities
			if len(caps) == 0 {
				caps = []types.Capability{"coding"}
			}
			labels := map[string]string{
				"workflow.planId": planID, "workflow.stepId": st.ID,
			}
			if st.AgentID != "" {
				labels["workflow.agentId"] = string(st.AgentID)
			}
			mid, err := o.runner.SubmitMission(ctx, host.Mission{
				Goal: st.Description, RequiredCaps: caps, Labels: labels,
			})
			if err != nil {
				res.Errors = append(res.Errors, st.ID+": "+err.Error())
				res.Status = "failed"
				_ = o.plans.SetStatus(planID, "failed")
				res.FinishedAt = time.Now().UTC()
				o.mu.Lock()
				o.runs = append(o.runs, res)
				o.mu.Unlock()
				return res, err
			}
			res.StepMissions[st.ID] = string(mid)
			m, _ := o.runner.GetMission(ctx, mid)
			if m.State == host.StateFailed {
				res.Errors = append(res.Errors, st.ID+": mission failed")
				res.Status = "failed"
				_ = o.plans.SetStatus(planID, "failed")
				res.FinishedAt = time.Now().UTC()
				o.mu.Lock()
				o.runs = append(o.runs, res)
				o.mu.Unlock()
				return res, fmt.Errorf("step %s failed", st.ID)
			}
			done[st.ID] = true
		}
		if !progress {
			res.Status = "failed"
			res.Errors = append(res.Errors, "dependency cycle or stuck DAG")
			_ = o.plans.SetStatus(planID, "failed")
			res.FinishedAt = time.Now().UTC()
			return res, fmt.Errorf("stuck DAG")
		}
		remaining = next
	}
	res.Status = "completed"
	res.FinishedAt = time.Now().UTC()
	_ = o.plans.SetStatus(planID, "completed")
	o.mu.Lock()
	o.runs = append(o.runs, res)
	o.mu.Unlock()
	return res, nil
}

func (o *Orchestrator) History() []RunResult {
	o.mu.Lock()
	defer o.mu.Unlock()
	return append([]RunResult{}, o.runs...)
}
