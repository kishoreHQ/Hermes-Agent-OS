package router

import (
	"context"
	"testing"

	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/provider"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/runtime"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/types"
)

type stubProvider struct {
	id     types.PluginID
	caps   []types.Capability
	tier   types.CostTier
	local  bool
	models []provider.ModelInfo
	dead   bool
}

func (p *stubProvider) ID() types.PluginID { return p.id }
func (p *stubProvider) Health(ctx context.Context) error {
	if p.dead {
		return errDead
	}
	return nil
}
func (p *stubProvider) Describe(ctx context.Context) (provider.Descriptor, error) {
	return provider.Descriptor{
		ID: p.id, Capabilities: p.caps, Models: p.models, Local: p.local, CostTier: p.tier,
	}, nil
}
func (p *stubProvider) Complete(ctx context.Context, req provider.CompletionRequest) (provider.CompletionResponse, error) {
	return provider.CompletionResponse{}, nil
}

type stubRuntime struct {
	id   types.PluginID
	dead bool
}

func (r *stubRuntime) ID() types.PluginID { return r.id }
func (r *stubRuntime) Health(ctx context.Context) error {
	if r.dead {
		return errDead
	}
	return nil
}
func (r *stubRuntime) Describe(ctx context.Context) (runtime.Descriptor, error) {
	return runtime.Descriptor{ID: r.id, Version: "0.0.1"}, nil
}
func (r *stubRuntime) Execute(ctx context.Context, env runtime.ContextEnvelope) (runtime.Result, error) {
	return runtime.Result{Status: "ok"}, nil
}

type deadError struct{}

func (deadError) Error() string { return "unhealthy" }

var errDead = deadError{}

func TestRoute_CapabilityMatchCheapestTier(t *testing.T) {
	r := New(
		[]provider.Provider{
			&stubProvider{
				id: "provider.premium", caps: []types.Capability{"coding", "tools"},
				tier: types.TierPremium, models: []provider.ModelInfo{{ID: "model-a"}},
			},
			&stubProvider{
				id: "provider.local", caps: []types.Capability{"coding", "tools"},
				tier: types.TierFreeLocal, local: true, models: []provider.ModelInfo{{ID: "local-model"}},
			},
		},
		[]runtime.Runtime{&stubRuntime{id: "runtime.echo"}},
	)
	d, err := r.Route(context.Background(), []types.Capability{"coding", "tools"}, false)
	if err != nil {
		t.Fatal(err)
	}
	if d.ProviderID != "provider.local" {
		t.Fatalf("want local free tier, got %s", d.ProviderID)
	}
	if d.RuntimeID != "runtime.echo" {
		t.Fatalf("runtime %s", d.RuntimeID)
	}
	if d.ModelID != "local-model" {
		t.Fatalf("model %s", d.ModelID)
	}
	if d.Reason == "" {
		t.Fatal("reason required for replay")
	}
}

func TestRoute_RejectsModelNameOnly(t *testing.T) {
	r := New(
		[]provider.Provider{
			&stubProvider{id: "p", caps: []types.Capability{"coding"}, tier: types.TierBudget, models: []provider.ModelInfo{{ID: "x"}}},
		},
		[]runtime.Runtime{&stubRuntime{id: "r"}},
	)
	_, err := r.Route(context.Background(), []types.Capability{"gpt-4", "claude"}, false)
	if err == nil {
		t.Fatal("expected error for model-name routing")
	}
}

func TestRoute_PreferLocalEscalates(t *testing.T) {
	r := New(
		[]provider.Provider{
			&stubProvider{
				id: "provider.cloud", caps: []types.Capability{"coding"},
				tier: types.TierStandard, local: false, models: []provider.ModelInfo{{ID: "cloud"}},
			},
		},
		[]runtime.Runtime{&stubRuntime{id: "runtime.echo"}},
	)
	d, err := r.Route(context.Background(), []types.Capability{"coding"}, true)
	if err != nil {
		t.Fatal(err)
	}
	if d.ProviderID != "provider.cloud" {
		t.Fatalf("expected escalate to cloud, got %s", d.ProviderID)
	}
}
