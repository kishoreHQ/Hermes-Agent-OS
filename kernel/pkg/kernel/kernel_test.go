package kernel

import (
	"context"
	"testing"

	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/adapters/echo"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/host"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/plugin"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/types"
)

func seedKernel(t *testing.T) *Kernel {
	t.Helper()
	reg := plugin.NewMemoryRegistry()
	pm := plugin.Manifest{
		APIVersion: "hermes.plugin/v1",
		Kind:       plugin.KindProvider,
		Metadata:   plugin.Metadata{ID: "provider.example.echo", Version: "0.0.1"},
		Spec: map[string]any{
			"capabilities": []any{"coding", "tools"},
			"local":        true,
			"costTier":     "free-local",
			"models":       []any{map[string]any{"id": "echo-1"}},
		},
		Labels: map[string]string{"hermes.driver": "echo-provider"},
	}
	p, err := echo.NewProvider(pm)
	if err != nil {
		t.Fatal(err)
	}
	_ = reg.Register(pm, p)
	rm := plugin.Manifest{
		APIVersion: "hermes.plugin/v1",
		Kind:       plugin.KindRuntime,
		Metadata:   plugin.Metadata{ID: "runtime.example.echo", Version: "0.0.1"},
		Spec:       map[string]any{"sandboxTier": "process-pty", "capabilitiesIn": []any{"coding", "tools"}},
		Labels:     map[string]string{"hermes.driver": "echo-runtime"},
	}
	rt, err := echo.NewRuntime(rm)
	if err != nil {
		t.Fatal(err)
	}
	_ = reg.Register(rm, rt)
	return New(reg)
}

func TestSubmitMission_RequiresCapabilities(t *testing.T) {
	k := New(plugin.NewMemoryRegistry())
	_, err := k.SubmitMission(context.Background(), host.Mission{
		ID: "m1", Goal: "do something",
	})
	if err == nil {
		t.Fatal("expected requiredCapabilities error")
	}
}

func TestSubmitMission_RejectsModelNamesOnly(t *testing.T) {
	k := seedKernel(t)
	_, err := k.SubmitMission(context.Background(), host.Mission{
		Goal: "ship", RequiredCaps: []types.Capability{"gpt-4", "claude"},
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSubmitMission_ExecutesAndSucceeds(t *testing.T) {
	k := seedKernel(t)
	id, err := k.SubmitMission(context.Background(), host.Mission{
		Goal: "ship", RequiredCaps: []types.Capability{"coding"},
	})
	if err != nil {
		t.Fatal(err)
	}
	m, err := k.GetMission(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if m.State != host.StateSucceeded {
		t.Fatalf("state %s output=%s", m.State, m.Output)
	}
	if m.ProviderID != "provider.example.echo" {
		t.Fatalf("provider %s", m.ProviderID)
	}
	evs, err := k.EventsSince(context.Background(), 0, string(id))
	if err != nil {
		t.Fatal(err)
	}
	if len(evs) < 3 {
		t.Fatalf("expected journal events, got %d", len(evs))
	}
	if evs[0].Seq != 1 {
		t.Fatalf("seq not monotonic: %+v", evs)
	}
}

func TestCancelUnknown(t *testing.T) {
	k := seedKernel(t)
	if err := k.CancelMission(context.Background(), "nope", "test"); err == nil {
		t.Fatal("expected error")
	}
}

func TestCancelMission(t *testing.T) {
	k := seedKernel(t)
	id, _ := k.SubmitMission(context.Background(), host.Mission{
		Goal: "x", RequiredCaps: []types.Capability{"coding"},
	})
	if err := k.CancelMission(context.Background(), id, "user"); err != nil {
		t.Fatal(err)
	}
	m, _ := k.GetMission(context.Background(), id)
	if m.State != host.StateCancelled {
		t.Fatalf("%s", m.State)
	}
}

func TestListMissions_Filter(t *testing.T) {
	k := seedKernel(t)
	id, _ := k.SubmitMission(context.Background(), host.Mission{
		Goal: "a", RequiredCaps: []types.Capability{"coding"},
	})
	_ = k.CancelMission(context.Background(), id, "done")
	_, _ = k.SubmitMission(context.Background(), host.Mission{
		Goal: "b", RequiredCaps: []types.Capability{"tools"},
	})
	cancelled, _ := k.ListMissions(context.Background(), "cancelled")
	if len(cancelled) != 1 {
		t.Fatalf("cancelled %d", len(cancelled))
	}
	succeeded, _ := k.ListMissions(context.Background(), "succeeded")
	if len(succeeded) != 1 {
		t.Fatalf("succeeded %d", len(succeeded))
	}
}

func TestNoProviderFailsMission(t *testing.T) {
	k := New(plugin.NewMemoryRegistry())
	id, err := k.SubmitMission(context.Background(), host.Mission{
		Goal: "orphan", RequiredCaps: []types.Capability{"coding"},
	})
	if err != nil {
		t.Fatal(err)
	}
	m, _ := k.GetMission(context.Background(), id)
	if m.State != host.StateFailed {
		t.Fatalf("want failed got %s", m.State)
	}
}
