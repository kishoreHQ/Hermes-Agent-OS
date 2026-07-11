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
	caps []types.Capability
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
	return runtime.Descriptor{ID: r.id, Version: "0.0.1", CapabilitiesIn: r.caps}, nil
}
func (r *stubRuntime) Execute(ctx context.Context, env runtime.ContextEnvelope) (runtime.Result, error) {
	return runtime.Result{Status: "ok"}, nil
}

type deadError struct{}

func (deadError) Error() string { return "unhealthy" }

var errDead = deadError{}

func fleet() *Router {
	return New(
		[]provider.Provider{
			&stubProvider{
				id: "provider.premium", caps: []types.Capability{"coding", "tools"},
				tier: types.TierPremium, models: []provider.ModelInfo{{ID: "model-a"}},
			},
			&stubProvider{
				id: "provider.local", caps: []types.Capability{"coding", "tools"},
				tier: types.TierFreeLocal, local: true, models: []provider.ModelInfo{{ID: "local-model"}},
			},
			&stubProvider{
				id: "provider.budget", caps: []types.Capability{"coding", "tools"},
				tier: types.TierBudget, local: false, models: []provider.ModelInfo{{ID: "budget-model"}},
			},
		},
		[]runtime.Runtime{
			&stubRuntime{id: "runtime.steps", caps: []types.Capability{"coding", "tools"}},
			&stubRuntime{id: "runtime.echo", caps: []types.Capability{"coding", "tools"}},
		},
	)
}

func TestRoute_CapabilityMatchCheapestTier(t *testing.T) {
	r := fleet()
	d, err := r.Route(context.Background(), []types.Capability{"coding", "tools"}, false)
	if err != nil {
		t.Fatal(err)
	}
	if d.ProviderID != "provider.local" {
		t.Fatalf("want local free tier, got %s", d.ProviderID)
	}
	// stable alphabetical runtime when no preference: echo before steps
	if d.RuntimeID != "runtime.echo" {
		t.Fatalf("runtime %s", d.RuntimeID)
	}
	if d.ModelID != "local-model" {
		t.Fatalf("model %s", d.ModelID)
	}
	if d.Reason == "" {
		t.Fatal("reason required for replay")
	}
	if d.ProvidersConsidered < 2 || d.RuntimesConsidered < 2 {
		t.Fatalf("candidates p=%d r=%d", d.ProvidersConsidered, d.RuntimesConsidered)
	}
}

func TestRoute_RejectsModelNameOnly(t *testing.T) {
	r := fleet()
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

func TestRoute_ExcludeProviderSwaps(t *testing.T) {
	r := fleet()
	d, err := r.RouteWith(context.Background(), []types.Capability{"coding", "tools"}, Options{
		PreferLocal:     true,
		ExcludeProvider: map[types.PluginID]bool{"provider.local": true},
	})
	if err != nil {
		t.Fatal(err)
	}
	// preferLocal excludes non-local first, then escalates → budget beats premium
	if d.ProviderID != "provider.budget" {
		t.Fatalf("want budget after excluding local, got %s", d.ProviderID)
	}
}

func TestRoute_PreferRuntimeSwap(t *testing.T) {
	r := fleet()
	d, err := r.RouteWith(context.Background(), []types.Capability{"coding", "tools"}, Options{
		PreferLocal:   true,
		PreferRuntime: "runtime.steps",
	})
	if err != nil {
		t.Fatal(err)
	}
	if d.RuntimeID != "runtime.steps" {
		t.Fatalf("want steps runtime, got %s", d.RuntimeID)
	}
	if d.ProviderID != "provider.local" {
		t.Fatalf("provider should stay free-local, got %s", d.ProviderID)
	}
}

func TestRoute_ExcludeRuntimeSwaps(t *testing.T) {
	r := fleet()
	d, err := r.RouteWith(context.Background(), []types.Capability{"coding", "tools"}, Options{
		ExcludeRuntime: map[types.PluginID]bool{"runtime.echo": true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if d.RuntimeID != "runtime.steps" {
		t.Fatalf("want steps after exclude echo, got %s", d.RuntimeID)
	}
}
