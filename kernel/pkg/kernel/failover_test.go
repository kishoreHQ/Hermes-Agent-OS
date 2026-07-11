package kernel

import (
	"context"
	"fmt"
	"testing"

	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/adapters/echo"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/host"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/plugin"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/provider"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/types"
)

// flakyProvider fails Complete once then succeeds — simulates failover target after primary.
type flakyProvider struct {
	*echo.Provider
	failsLeft int
}

func (f *flakyProvider) Complete(ctx context.Context, req provider.CompletionRequest) (provider.CompletionResponse, error) {
	if f.failsLeft > 0 {
		f.failsLeft--
		return provider.CompletionResponse{}, fmt.Errorf("simulated outage")
	}
	return f.Provider.Complete(ctx, req)
}

func TestProviderFailoverOnComplete(t *testing.T) {
	reg := plugin.NewMemoryRegistry()
	// Primary: free-local but will fail complete
	pm1 := plugin.Manifest{
		APIVersion: "hermes.plugin/v1", Kind: plugin.KindProvider,
		Metadata: plugin.Metadata{ID: "provider.flaky", Version: "1"},
		Spec: map[string]any{
			"capabilities": []any{"coding", "tools"}, "local": true, "costTier": "free-local",
			"models": []any{map[string]any{"id": "flaky-1"}},
		},
	}
	base1, _ := echo.NewProvider(pm1)
	flaky := &flakyProvider{Provider: base1, failsLeft: 100} // always fail
	_ = reg.Register(pm1, flaky)

	pm2 := plugin.Manifest{
		APIVersion: "hermes.plugin/v1", Kind: plugin.KindProvider,
		Metadata: plugin.Metadata{ID: "provider.backup", Version: "1"},
		Spec: map[string]any{
			"capabilities": []any{"coding", "tools"}, "local": false, "costTier": "budget",
			"models": []any{map[string]any{"id": "backup-1"}},
		},
	}
	p2, _ := echo.NewProvider(pm2)
	_ = reg.Register(pm2, p2)

	rm := plugin.Manifest{
		APIVersion: "hermes.plugin/v1", Kind: plugin.KindRuntime,
		Metadata: plugin.Metadata{ID: "runtime.example.echo", Version: "1"},
		Spec:     map[string]any{"capabilitiesIn": []any{"coding", "tools"}},
	}
	rt, _ := echo.NewRuntime(rm)
	_ = reg.Register(rm, rt)

	k := New(reg)
	id, err := k.SubmitMission(context.Background(), host.Mission{
		Goal: "failover please", RequiredCaps: []types.Capability{"coding", "tools"},
	})
	if err != nil {
		t.Fatal(err)
	}
	m, _ := k.GetMission(context.Background(), id)
	if m.State != host.StateSucceeded {
		t.Fatalf("state %s out=%s", m.State, m.Output)
	}
	if m.ProviderID != "provider.backup" {
		t.Fatalf("expected backup provider, got %s", m.ProviderID)
	}
	// Journal should show provider.failed then success path
	evs, _ := k.Replay(context.Background(), id)
	var sawFail, sawFO bool
	for _, e := range evs {
		if e.Type == "provider.failed" {
			sawFail = true
		}
		if e.Type == "provider.failover" {
			sawFO = true
		}
	}
	if !sawFail || !sawFO {
		t.Fatalf("expected failover events fail=%v fo=%v", sawFail, sawFO)
	}
}
