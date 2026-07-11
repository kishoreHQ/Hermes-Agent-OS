// Package router implements capability-based routing (INV-03).
// Intent → Capabilities → Policy → Budget → Security → Availability → Provider → Model → Runtime
// Supports multi-provider candidate chains for failover.
package router

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/capability"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/provider"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/runtime"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/types"
)

// Decision is a replayable routing outcome.
type Decision struct {
	ProviderID          types.PluginID     `json:"providerId"`
	RuntimeID           types.PluginID     `json:"runtimeId"`
	ModelID             string             `json:"modelId"`
	Required            []types.Capability `json:"required"`
	CostTier            types.CostTier     `json:"costTier"`
	ComplexityScore     float64            `json:"complexityScore,omitempty"`
	Reason              string             `json:"reason"`
	PolicyID            string             `json:"policyId,omitempty"`
	ProvidersConsidered int                `json:"providersConsidered,omitempty"`
	RuntimesConsidered  int                `json:"runtimesConsidered,omitempty"`
	// CandidateRank is 0 for primary, 1+ for failover alternatives.
	CandidateRank int `json:"candidateRank,omitempty"`
}

// Options soft-steer routing without hardcoding vendors in the kernel.
// Preferences are optional; required capabilities remain the primary key.
type Options struct {
	PreferLocal    bool
	PreferProvider types.PluginID // soft preference among capable providers
	PreferRuntime  types.PluginID
	// PreferModel selects a model id when the provider exposes it (discovery or static).
	PreferModel string
	// AllowProviders if non-empty: only these provider ids (operator allowlist).
	AllowProviders map[types.PluginID]bool
	// RequireProvider if set: hard pin (failover still possible only if FailoverAfterRequire).
	RequireProvider types.PluginID
	ExcludeProvider map[types.PluginID]bool
	ExcludeRuntime  map[types.PluginID]bool
	// Failover enables multi-provider candidate chains (default true when using Candidates).
	Failover bool
	PolicyID string
}

type Router struct {
	Providers []provider.Provider
	Runtimes  []runtime.Runtime
	Caps      *capability.Engine
}

func New(providers []provider.Provider, runtimes []runtime.Runtime) *Router {
	return &Router{Providers: providers, Runtimes: runtimes, Caps: capability.New()}
}

// Route selects the best provider + model + runtime.
func (r *Router) Route(ctx context.Context, required []types.Capability, preferLocal bool) (Decision, error) {
	return r.RouteWith(ctx, required, Options{PreferLocal: preferLocal, Failover: true})
}

func (r *Router) RouteWith(ctx context.Context, required []types.Capability, opts Options) (Decision, error) {
	cands, err := r.Candidates(ctx, required, opts)
	if err != nil {
		return Decision{}, err
	}
	return cands[0], nil
}

// Candidates returns ordered provider/model/runtime decisions for failover.
// Primary is index 0; later entries are alternatives if primary fails at complete-time.
func (r *Router) Candidates(ctx context.Context, required []types.Capability, opts Options) ([]Decision, error) {
	req := r.Caps.Normalize(required)
	if len(req) == 0 {
		return nil, fmt.Errorf("no valid capabilities (INV-03: never model-name route)")
	}

	type pCand struct {
		p      provider.Provider
		desc   provider.Descriptor
		models []provider.ModelInfo
	}
	collect := func(localOnly bool) []pCand {
		var out []pCand
		for _, p := range r.Providers {
			if opts.ExcludeProvider != nil && opts.ExcludeProvider[p.ID()] {
				continue
			}
			if opts.RequireProvider != "" && p.ID() != opts.RequireProvider && !opts.Failover {
				continue
			}
			if len(opts.AllowProviders) > 0 && !opts.AllowProviders[p.ID()] {
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
			if localOnly && !d.Local {
				continue
			}
			models, _ := provider.DiscoverModels(ctx, p)
			if len(models) == 0 {
				models = d.Models
			}
			if opts.PreferModel != "" && len(models) > 0 && !hasModel(models, opts.PreferModel) {
				if opts.RequireProvider != p.ID() && opts.PreferProvider != p.ID() {
					continue
				}
			}
			out = append(out, pCand{p: p, desc: d, models: models})
		}
		return out
	}

	var providers []pCand
	if opts.PreferLocal {
		// Locals first; with failover, append non-local as lower-priority candidates
		providers = collect(true)
		if opts.Failover {
			seen := map[types.PluginID]bool{}
			for _, pc := range providers {
				seen[pc.p.ID()] = true
			}
			for _, pc := range collect(false) {
				if !seen[pc.p.ID()] {
					providers = append(providers, pc)
				}
			}
		}
		if len(providers) == 0 {
			providers = collect(false)
		}
	} else {
		providers = collect(false)
	}
	if len(providers) == 0 {
		return nil, fmt.Errorf("no provider for capabilities %v", req)
	}

	sort.SliceStable(providers, func(i, j int) bool {
		// Hard require first
		if opts.RequireProvider != "" {
			if providers[i].p.ID() == opts.RequireProvider && providers[j].p.ID() != opts.RequireProvider {
				return true
			}
			if providers[j].p.ID() == opts.RequireProvider && providers[i].p.ID() != opts.RequireProvider {
				return false
			}
		}
		if opts.PreferProvider != "" {
			if providers[i].p.ID() == opts.PreferProvider && providers[j].p.ID() != opts.PreferProvider {
				return true
			}
			if providers[j].p.ID() == opts.PreferProvider && providers[i].p.ID() != opts.PreferProvider {
				return false
			}
		}
		// Prefer providers that have requested model
		if opts.PreferModel != "" {
			hi, hj := hasModel(providers[i].models, opts.PreferModel), hasModel(providers[j].models, opts.PreferModel)
			if hi && !hj {
				return true
			}
			if hj && !hi {
				return false
			}
		}
		ri, rj := rankTier(providers[i].desc.CostTier), rankTier(providers[j].desc.CostTier)
		if ri != rj {
			return ri < rj
		}
		return string(providers[i].p.ID()) < string(providers[j].p.ID())
	})

	// If hard require without failover, keep only that provider
	if opts.RequireProvider != "" && !opts.Failover {
		var filtered []pCand
		for _, pc := range providers {
			if pc.p.ID() == opts.RequireProvider {
				filtered = append(filtered, pc)
			}
		}
		if len(filtered) == 0 {
			return nil, fmt.Errorf("required provider %s unavailable", opts.RequireProvider)
		}
		providers = filtered
	}

	rtID, nRT, err := r.pickRuntime(ctx, req, opts)
	if err != nil {
		return nil, err
	}

	var out []Decision
	for i, pc := range providers {
		model := pickModel(pc.models, opts.PreferModel)
		reason := "capability-match+tier-order"
		if opts.PreferProvider != "" || opts.PreferRuntime != "" || opts.PreferModel != "" {
			reason = "capability-match+tier-order+preference"
		}
		if opts.RequireProvider != "" {
			reason += "+require-provider"
		}
		if opts.PreferLocal {
			reason += "+prefer-local"
		}
		if i > 0 {
			reason += "+failover-candidate"
		}
		out = append(out, Decision{
			ProviderID: pc.p.ID(), RuntimeID: rtID, ModelID: model,
			Required: req, CostTier: pc.desc.CostTier, Reason: reason, PolicyID: opts.PolicyID,
			ProvidersConsidered: len(providers), RuntimesConsidered: nRT, CandidateRank: i,
		})
		if !opts.Failover {
			break
		}
	}
	return out, nil
}

func (r *Router) pickRuntime(ctx context.Context, req []types.Capability, opts Options) (types.PluginID, int, error) {
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
		if len(d.CapabilitiesIn) > 0 && !capability.Compatible(d.CapabilitiesIn, req) {
			continue
		}
		runtimes = append(runtimes, rCand{rt: rt, desc: d})
	}
	if len(runtimes) == 0 {
		return "", 0, fmt.Errorf("no runtime available for capabilities %v", req)
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
		return string(runtimes[i].rt.ID()) < string(runtimes[j].rt.ID())
	})
	return runtimes[0].rt.ID(), len(runtimes), nil
}

func hasModel(models []provider.ModelInfo, id string) bool {
	for _, m := range models {
		if m.ID == id || strings.EqualFold(m.ID, id) {
			return true
		}
	}
	return false
}

func pickModel(models []provider.ModelInfo, prefer string) string {
	if prefer != "" {
		for _, m := range models {
			if m.ID == prefer || strings.EqualFold(m.ID, prefer) {
				return m.ID
			}
		}
		// operator asked for model — still pass through (provider may accept)
		return prefer
	}
	if len(models) > 0 {
		return models[0].ID
	}
	return ""
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
