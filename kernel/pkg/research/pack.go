// Package research provides a deep-research mission helper (skill + tools orchestration).
package research

import (
	"context"
	"fmt"

	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/host"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/types"
)

type submitter interface {
	SubmitMission(ctx context.Context, m host.Mission) (types.MissionID, error)
	GetMission(ctx context.Context, id types.MissionID) (host.Mission, error)
}

// Request for a research run.
type Request struct {
	Topic          string `json:"topic"`
	PreferProvider string `json:"preferProvider,omitempty"`
	PreferModel    string `json:"preferModel,omitempty"`
}

// Run submits a structured deep-research mission via agent loop + web tools.
func Run(ctx context.Context, k submitter, req Request) (host.Mission, error) {
	if req.Topic == "" {
		return host.Mission{}, fmt.Errorf("topic required")
	}
	goal := fmt.Sprintf(`Deep research task on: %s

Follow the research skill:
1) research.outline
2) web.search for key queries
3) web.fetch best sources
4) Produce a report with: Executive summary, Key findings (bullets with source URLs), Conflicting views, Open questions.

Be factual and cite URLs.`, req.Topic)

	labels := map[string]string{
		"route.preferRuntime": "runtime.agent.loop",
		"skills":              "research",
		"kind":                "deep-research",
	}
	if req.PreferProvider != "" {
		labels["route.preferProvider"] = req.PreferProvider
	}
	if req.PreferModel != "" {
		labels["route.preferModel"] = req.PreferModel
	}
	mid, err := k.SubmitMission(ctx, host.Mission{
		Goal:           goal,
		Name:           "Research: " + req.Topic,
		RequiredCaps:   []types.Capability{"coding", "tools"},
		Labels:         labels,
		PreferProvider: types.PluginID(req.PreferProvider),
		PreferModel:    req.PreferModel,
		Failover:       boolPtr(true),
	})
	if err != nil {
		return host.Mission{}, err
	}
	m, err := k.GetMission(ctx, mid)
	if err != nil {
		return host.Mission{}, err
	}
	return m, nil
}

func boolPtr(b bool) *bool { return &b }
