package router

import (
	"context"
	"fmt"
	"testing"

	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/provider"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/runtime"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/types"
)

type failProvider struct {
	stubProvider
	failComplete bool
}

func (p *failProvider) Complete(ctx context.Context, req provider.CompletionRequest) (provider.CompletionResponse, error) {
	if p.failComplete {
		return provider.CompletionResponse{}, fmt.Errorf("complete failed")
	}
	return provider.CompletionResponse{ProviderID: p.id, ModelID: "m", Content: "ok"}, nil
}

func TestCandidatesFailoverChain(t *testing.T) {
	r := New(
		[]provider.Provider{
			&stubProvider{
				id: "provider.local", caps: []types.Capability{"coding"},
				tier: types.TierFreeLocal, local: true, models: []provider.ModelInfo{{ID: "local-1"}},
			},
			&stubProvider{
				id: "provider.budget", caps: []types.Capability{"coding"},
				tier: types.TierBudget, models: []provider.ModelInfo{{ID: "budget-1"}},
			},
		},
		[]runtime.Runtime{&stubRuntime{id: "runtime.echo"}},
	)
	cands, err := r.Candidates(context.Background(), []types.Capability{"coding"}, Options{Failover: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(cands) < 2 {
		t.Fatalf("want failover chain, got %d", len(cands))
	}
	if cands[0].ProviderID != "provider.local" {
		t.Fatalf("primary %s", cands[0].ProviderID)
	}
	if cands[1].ProviderID != "provider.budget" {
		t.Fatalf("secondary %s", cands[1].ProviderID)
	}
}

func TestPreferModelAndProvider(t *testing.T) {
	r := New(
		[]provider.Provider{
			&stubProvider{
				id: "provider.a", caps: []types.Capability{"coding"},
				tier: types.TierPremium, models: []provider.ModelInfo{{ID: "gpt-x"}, {ID: "other"}},
			},
			&stubProvider{
				id: "provider.b", caps: []types.Capability{"coding"},
				tier: types.TierBudget, models: []provider.ModelInfo{{ID: "budget-1"}},
			},
		},
		[]runtime.Runtime{&stubRuntime{id: "runtime.echo"}},
	)
	d, err := r.RouteWith(context.Background(), []types.Capability{"coding"}, Options{
		PreferProvider: "provider.a", PreferModel: "gpt-x", Failover: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if d.ProviderID != "provider.a" || d.ModelID != "gpt-x" {
		t.Fatalf("%+v", d)
	}
}

func TestAllowlist(t *testing.T) {
	r := New(
		[]provider.Provider{
			&stubProvider{id: "provider.a", caps: []types.Capability{"coding"}, tier: types.TierFreeLocal, local: true, models: []provider.ModelInfo{{ID: "a"}}},
			&stubProvider{id: "provider.b", caps: []types.Capability{"coding"}, tier: types.TierBudget, models: []provider.ModelInfo{{ID: "b"}}},
		},
		[]runtime.Runtime{&stubRuntime{id: "r"}},
	)
	d, err := r.RouteWith(context.Background(), []types.Capability{"coding"}, Options{
		AllowProviders: map[types.PluginID]bool{"provider.b": true},
		Failover:       true,
	})
	if err != nil || d.ProviderID != "provider.b" {
		t.Fatalf("%+v %v", d, err)
	}
}
