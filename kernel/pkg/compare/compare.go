// Package compare runs the same goal across multiple providers for side-by-side evaluation.
package compare

import (
	"context"
	"fmt"
	"sync"

	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/host"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/types"
)

type submitter interface {
	SubmitMission(ctx context.Context, m host.Mission) (types.MissionID, error)
	GetMission(ctx context.Context, id types.MissionID) (host.Mission, error)
}

// Request for multi-provider compare.
type Request struct {
	Goal      string   `json:"goal"`
	Providers []string `json:"providers"` // plugin ids
	Model     string   `json:"model,omitempty"`
}

// Result one arm.
type Result struct {
	ProviderID string `json:"providerId"`
	MissionID  string `json:"missionId"`
	State      string `json:"state"`
	Output     string `json:"output"`
	Error      string `json:"error,omitempty"`
}

// Run executes goal on each provider (sequential to avoid rate storms).
func Run(ctx context.Context, k submitter, req Request) ([]Result, error) {
	if req.Goal == "" {
		return nil, fmt.Errorf("goal required")
	}
	if len(req.Providers) == 0 {
		return nil, fmt.Errorf("providers required")
	}
	out := make([]Result, 0, len(req.Providers))
	var mu sync.Mutex
	// sequential for predictable e2e
	for _, pid := range req.Providers {
		r := Result{ProviderID: pid}
		labels := map[string]string{
			"route.requireProvider": pid,
			"route.failover":        "false",
			"route.preferRuntime":   "runtime.agent.loop",
			"kind":                  "compare",
		}
		if req.Model != "" {
			labels["route.preferModel"] = req.Model
		}
		mid, err := k.SubmitMission(ctx, host.Mission{
			Goal:           req.Goal,
			Name:           "Compare: " + pid,
			RequiredCaps:   []types.Capability{"coding", "tools"},
			Labels:         labels,
			PreferProvider: types.PluginID(pid),
			PreferModel:    req.Model,
			RequireProvider: types.PluginID(pid),
			Failover:       boolPtr(false),
		})
		if err != nil {
			r.Error = err.Error()
			r.State = "failed"
			mu.Lock()
			out = append(out, r)
			mu.Unlock()
			continue
		}
		m, err := k.GetMission(ctx, mid)
		if err != nil {
			r.Error = err.Error()
			r.State = "failed"
		} else {
			r.MissionID = string(m.ID)
			r.State = string(m.State)
			r.Output = m.Output
		}
		out = append(out, r)
	}
	return out, nil
}

func boolPtr(b bool) *bool { return &b }
