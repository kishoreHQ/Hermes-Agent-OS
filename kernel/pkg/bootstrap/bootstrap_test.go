package bootstrap

import (
	"context"
	"testing"

	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/host"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/memorystore"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/types"
)

func TestBootstrapSeedAndExecute(t *testing.T) {
	res, err := New(Options{SeedBuiltins: true, PluginRoots: []string{"/nonexistent"}})
	if err != nil {
		t.Fatal(err)
	}
	if res.Loaded < 3 {
		t.Fatalf("loaded %d", res.Loaded)
	}
	id, err := res.Kernel.SubmitMission(context.Background(), host.Mission{
		Goal:         "prove plugin path",
		RequiredCaps: []types.Capability{"coding", "tools"},
	})
	if err != nil {
		t.Fatal(err)
	}
	m, err := res.Kernel.GetMission(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if m.State != host.StateSucceeded {
		t.Fatalf("state=%s output=%s", m.State, m.Output)
	}
	if m.ProviderID == "" || m.RuntimeID == "" {
		t.Fatalf("routing not recorded: %+v", m)
	}
	// free-local echo should win over budget
	if m.ProviderID != "provider.example.echo" {
		t.Fatalf("expected free-local provider, got %s", m.ProviderID)
	}
	if m.Output == "" {
		t.Fatal("empty output")
	}
	memHits, err := res.Kernel.Memory().Search(context.Background(), memorystore.Query{MissionID: id})
	if err != nil {
		t.Fatal(err)
	}
	if len(memHits) == 0 {
		t.Fatal("expected episodic memory write")
	}
	evs, _ := res.Kernel.Replay(context.Background(), id)
	var sawRoute bool
	for _, e := range evs {
		if e.Type == "route.decided" {
			sawRoute = true
		}
	}
	if !sawRoute {
		t.Fatalf("events: %+v", evs)
	}
}
