// Package router implements capability-based routing (INV-03).
// Intent → Capabilities → Policy → Budget → Security → Availability → Provider → Model → Runtime
package router

import (
	"context"
	"fmt"

	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/capability"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/provider"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/runtime"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/types"
)

type Decision struct {
	ProviderID     types.PluginID  `json:"providerId"`
	RuntimeID      types.PluginID  `json:"runtimeId"`
	ModelID        string          `json:"modelId"`
	Required       []types.Capability `json:"required"`
	CostTier        types.CostTier   `json:"costTier"`
	ComplexityScore float64         `json:"complexityScore,omitempty"`
	Reason         string          `json:"reason"`
	// Replayable audit fields
	PolicyID string `json:"policyId,omitempty"`
}

type Router struct {
	Providers []provider.Provider
	Runtimes  []runtime.Runtime
	Caps      *capability.Engine
}

func New(providers []provider.Provider, runtimes []runtime.Runtime) *Router {
	return &Router{Providers: providers, Runtimes: runtimes, Caps: capability.New()}
}

func (r *Router) Route(ctx context.Context, required []types.Capability, preferLocal bool) (Decision, error) {
	req := r.Caps.Normalize(required)
	if len(req) == 0 {
		return Decision{}, fmt.Errorf("no valid capabilities (INV-03: never model-name route)")
	}
	var chosen provider.Provider
	var desc provider.Descriptor
	for _, p := range r.Providers {
		if err := p.Health(ctx); err != nil {
			continue
		}
		d, err := p.Describe(ctx)
		if err != nil {
			continue
		}
		if !capability.Compatible(d.Capabilities, req) {
			continue
		}
		if preferLocal && !d.Local {
			continue
		}
		if chosen == nil || rankTier(d.CostTier) < rankTier(desc.CostTier) {
			chosen, desc = p, d
		}
	}
	if chosen == nil && preferLocal {
		// escalate: try non-local
		return r.Route(ctx, req, false)
	}
	if chosen == nil {
		return Decision{}, fmt.Errorf("no provider for capabilities %v", req)
	}
	model := ""
	if len(desc.Models) > 0 {
		model = desc.Models[0].ID
	}
	var rtID types.PluginID
	for _, rt := range r.Runtimes {
		if rt.Health(ctx) != nil {
			continue
		}
		rtID = rt.ID()
		break
	}
	if rtID == "" {
		return Decision{}, fmt.Errorf("no runtime available")
	}
	return Decision{
		ProviderID: chosen.ID(), RuntimeID: rtID, ModelID: model,
		Required: req, CostTier: desc.CostTier,
		Reason: "capability-match+tier-order",
	}, nil
}

func rankTier(t types.CostTier) int {
	order := []types.CostTier{
		types.TierFreeLocal, types.TierFreeHosted, types.TierBudget, types.TierStandard, types.TierPremium,
	}
	for i, x := range order {
		if x == t {
			return i
		}
	}
	return 99
}
