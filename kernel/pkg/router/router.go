// Package router implements capability-based routing (INV-03).
// Intent → Capabilities → Policy → Budget → Security → Availability → Provider → Model → Runtime
package router

import (
	"context"
	"fmt"
	"sort"

	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/capability"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/provider"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/runtime"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/types"
)

// Decision is a replayable routing outcome.
type Decision struct {
	ProviderID      types.PluginID    `json:"providerId"`
	RuntimeID       types.PluginID    `json:"runtimeId"`
	ModelID         string            `json:"modelId"`
	Required        []types.Capability `json:"required"`
	CostTier        types.CostTier    `json:"costTier"`
	ComplexityScore float64           `json:"complexityScore,omitempty"`
	Reason          string            `json:"reason"`
	PolicyID        string            `json:"policyId,omitempty"`
	// Candidates considered (for interchangeability audit)
	ProvidersConsidered int `json:"providersConsidered,omitempty"`
	RuntimesConsidered  int `json:"runtimesConsidered,omitempty"`
}

// Options soft-steer routing without hardcoding vendors in the kernel.
// Preferences are optional; required capabilities remain the primary key.
type Options struct {
	PreferLocal     bool
	PreferProvider  types.PluginID // soft preference among capable providers
	PreferRuntime   types.PluginID // soft preference among capable runtimes
	ExcludeProvider map[types.PluginID]bool
	ExcludeRuntime  map[types.PluginID]bool
	PolicyID        string
}

type Router struct {
	Providers []provider.Provider
	Runtimes  []runtime.Runtime
	Caps      *capability.Engine
}

func New(providers []provider.Provider, runtimes []runtime.Runtime) *Router {
	return &Router{Providers: providers, Runtimes: runtimes, Caps: capability.New()}
}

// Route selects provider + model + runtime from capabilities (never model-name primary key).
func (r *Router) Route(ctx context.Context, required []types.Capability, preferLocal bool) (Decision, error) {
	return r.RouteWith(ctx, required, Options{PreferLocal: preferLocal})
}

func (r *Router) RouteWith(ctx context.Context, required []types.Capability, opts Options) (Decision, error) {
	req := r.Caps.Normalize(required)
	if len(req) == 0 {
		return Decision{}, fmt.Errorf("no valid capabilities (INV-03: never model-name route)")
	}

	type pCand struct {
		p    provider.Provider
		desc provider.Descriptor
	}
	var providers []pCand
	for _, p := range r.Providers {
		if opts.ExcludeProvider != nil && opts.ExcludeProvider[p.ID()] {
			continue
		}
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
		if opts.PreferLocal && !d.Local {
			continue
		}
		providers = append(providers, pCand{p: p, desc: d})
	}
	if len(providers) == 0 && opts.PreferLocal {
		// escalate: drop local preference
		escalated := opts
		escalated.PreferLocal = false
		return r.RouteWith(ctx, req, escalated)
	}
	if len(providers) == 0 {
		return Decision{}, fmt.Errorf("no provider for capabilities %v", req)
	}

	// Order: soft prefer → cost tier → stable id
	sort.SliceStable(providers, func(i, j int) bool {
		if opts.PreferProvider != "" {
			if providers[i].p.ID() == opts.PreferProvider && providers[j].p.ID() != opts.PreferProvider {
				return true
			}
			if providers[j].p.ID() == opts.PreferProvider && providers[i].p.ID() != opts.PreferProvider {
				return false
			}
		}
		ri, rj := rankTier(providers[i].desc.CostTier), rankTier(providers[j].desc.CostTier)
		if ri != rj {
			return ri < rj
		}
		return string(providers[i].p.ID()) < string(providers[j].p.ID())
	})
	chosen := providers[0]
	model := ""
	if len(chosen.desc.Models) > 0 {
		model = chosen.desc.Models[0].ID
	}

	type rCand struct {
		rt   runtime.Runtime
		desc runtime.Descriptor
	}
	var runtimes []rCand
	for _, rt := range r.Runtimes {
		if opts.ExcludeRuntime != nil && opts.ExcludeRuntime[rt.ID()] {
			continue
		}
		if err := rt.Health(ctx); err != nil {
			continue
		}
		d, err := rt.Describe(ctx)
		if err != nil {
			continue
		}
		// If runtime declares capabilitiesIn, require match; empty means accept any.
		if len(d.CapabilitiesIn) > 0 && !capability.Compatible(d.CapabilitiesIn, req) {
			continue
		}
		runtimes = append(runtimes, rCand{rt: rt, desc: d})
	}
	if len(runtimes) == 0 {
		return Decision{}, fmt.Errorf("no runtime available for capabilities %v", req)
	}
	sort.SliceStable(runtimes, func(i, j int) bool {
		if opts.PreferRuntime != "" {
			if runtimes[i].rt.ID() == opts.PreferRuntime && runtimes[j].rt.ID() != opts.PreferRuntime {
				return true
			}
			if runtimes[j].rt.ID() == opts.PreferRuntime && runtimes[i].rt.ID() != opts.PreferRuntime {
				return false
			}
		}
		// Prefer stricter sandbox when equal preference (container > process-pty is inverted for "safer first" later;
		// H4: stable alphabetical id for determinism.
		return string(runtimes[i].rt.ID()) < string(runtimes[j].rt.ID())
	})
	rtChosen := runtimes[0]

	reason := "capability-match+tier-order"
	if opts.PreferProvider != "" || opts.PreferRuntime != "" {
		reason = "capability-match+tier-order+preference"
	}
	if opts.PreferLocal {
		reason += "+prefer-local"
	}

	return Decision{
		ProviderID:          chosen.p.ID(),
		RuntimeID:           rtChosen.rt.ID(),
		ModelID:             model,
		Required:            req,
		CostTier:            chosen.desc.CostTier,
		Reason:              reason,
		PolicyID:            opts.PolicyID,
		ProvidersConsidered: len(providers),
		RuntimesConsidered:  len(runtimes),
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
