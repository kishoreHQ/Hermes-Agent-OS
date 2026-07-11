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

func TestSubmitMission_OK(t *testing.T) {
	k := New(nil)
	id, err := k.SubmitMission(context.Background(), host.Mission{
		ID: "m1", Goal: "ship", RequiredCaps: []types.Capability{"coding"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if id != "m1" {
		t.Fatalf("%s", id)
	}
}

func TestCancelUnknown(t *testing.T) {
	k := New(nil)
	if err := k.CancelMission(context.Background(), "nope", "test"); err == nil {
		t.Fatal("expected error")
	}
}
