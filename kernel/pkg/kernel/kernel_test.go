package kernel

import (
	"context"
	"testing"

	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/host"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/plugin"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/types"
)

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
	k := New(nil)
	_, err := k.SubmitMission(context.Background(), host.Mission{
		Goal: "ship", RequiredCaps: []types.Capability{"gpt-4", "claude"},
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSubmitMission_OK_EmitsEvents(t *testing.T) {
	k := New(nil)
	id, err := k.SubmitMission(context.Background(), host.Mission{
		Goal: "ship", RequiredCaps: []types.Capability{"coding"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if id == "" {
		t.Fatal("empty id")
	}
	m, err := k.GetMission(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if m.State != host.StateRunning {
		t.Fatalf("state %s", m.State)
	}
	evs, err := k.EventsSince(context.Background(), 0, string(id))
	if err != nil {
		t.Fatal(err)
	}
	if len(evs) < 2 {
		t.Fatalf("expected journal events, got %d", len(evs))
	}
	if evs[0].Seq != 1 || evs[1].Seq != 2 {
		t.Fatalf("seq not monotonic: %+v", evs)
	}
}

func TestCancelUnknown(t *testing.T) {
	k := New(nil)
	if err := k.CancelMission(context.Background(), "nope", "test"); err == nil {
		t.Fatal("expected error")
	}
}

func TestCancelMission(t *testing.T) {
	k := New(nil)
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
	k := New(nil)
	id, _ := k.SubmitMission(context.Background(), host.Mission{
		Goal: "a", RequiredCaps: []types.Capability{"coding"},
	})
	_ = k.CancelMission(context.Background(), id, "done")
	_, _ = k.SubmitMission(context.Background(), host.Mission{
		Goal: "b", RequiredCaps: []types.Capability{"tools"},
	})
	cancelled, _ := k.ListMissions(context.Background(), "cancelled")
	if len(cancelled) != 1 {
		t.Fatalf("%d", len(cancelled))
	}
	running, _ := k.ListMissions(context.Background(), "running")
	if len(running) != 1 {
		t.Fatalf("%d", len(running))
	}
}
