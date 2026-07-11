// Package interchange proves H4: providers and runtimes are interchangeable
// without kernel source changes — only plugin registry / route labels differ.
package interchange_test

import (
	"context"
	"testing"

	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/bootstrap"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/host"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/plugin"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/types"
)

const goal = "H4 interchangeability proof mission"

func caps() []types.Capability {
	return []types.Capability{"coding", "tools"}
}

func boot(t *testing.T) *bootstrap.Result {
	t.Helper()
	res, err := bootstrap.New(bootstrap.Options{SeedBuiltins: true, PluginRoots: []string{"/no-disk-plugins"}})
	if err != nil {
		t.Fatal(err)
	}
	// H4 floor: ≥2 providers, ≥2 runtimes
	if n := len(res.Registry.List(plugin.KindProvider)); n < 2 {
		t.Fatalf("providers %d", n)
	}
	if n := len(res.Registry.List(plugin.KindRuntime)); n < 2 {
		t.Fatalf("runtimes %d", n)
	}
	return res
}

func submit(t *testing.T, k interface {
	SubmitMission(ctx context.Context, m host.Mission) (types.MissionID, error)
	GetMission(ctx context.Context, id types.MissionID) (host.Mission, error)
	Replay(ctx context.Context, id types.MissionID) ([]host.Event, error)
}, labels map[string]string) host.Mission {
	t.Helper()
	id, err := k.SubmitMission(context.Background(), host.Mission{
		Goal: goal, RequiredCaps: caps(), Labels: labels,
	})
	if err != nil {
		t.Fatal(err)
	}
	m, err := k.GetMission(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if m.State != host.StateSucceeded {
		t.Fatalf("state=%s output=%s", m.State, m.Output)
	}
	// Replay must explain capability path
	evs, err := k.Replay(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	var route map[string]any
	for _, e := range evs {
		if e.Type == "route.decided" {
			route = e.Data
			break
		}
	}
	if route == nil {
		t.Fatal("missing route.decided in replay")
	}
	req, _ := route["required"].([]any)
	if len(req) == 0 {
		// may be []string depending on encoding — also check string slice via assertion
		if _, ok := route["required"].([]string); !ok {
			// still ok if required present as []types.Capability serialized to []any of strings
			if route["required"] == nil {
				t.Fatalf("route missing required caps: %+v", route)
			}
		}
	}
	if route["reason"] == nil || route["reason"] == "" {
		t.Fatal("route reason empty")
	}
	// Must not use model-name as primary routing key
	if reason, _ := route["reason"].(string); reason == "gpt-4" || reason == "claude" {
		t.Fatalf("vendor model as reason: %s", reason)
	}
	return m
}

func TestH4_DefaultRouteFreeLocalAndEcho(t *testing.T) {
	res := boot(t)
	m := submit(t, res.Kernel, nil)
	if m.ProviderID != "provider.example.echo" {
		t.Fatalf("provider %s", m.ProviderID)
	}
	// alphabetical: echo before steps
	if m.RuntimeID != "runtime.example.echo" {
		t.Fatalf("runtime %s", m.RuntimeID)
	}
}

func TestH4_ProviderSwapExcludeLocal(t *testing.T) {
	res := boot(t)
	m := submit(t, res.Kernel, map[string]string{
		"route.excludeProvider": "provider.example.echo",
		"route.preferLocal":     "false",
	})
	if m.ProviderID != "provider.example.budget" {
		t.Fatalf("expected budget provider after exclude local, got %s", m.ProviderID)
	}
	if m.State != host.StateSucceeded {
		t.Fatal(m.State)
	}
}

func TestH4_RuntimeSwapPreferSteps(t *testing.T) {
	res := boot(t)
	m := submit(t, res.Kernel, map[string]string{
		"route.preferRuntime": "runtime.example.steps",
	})
	if m.RuntimeID != "runtime.example.steps" {
		t.Fatalf("runtime %s", m.RuntimeID)
	}
	if m.ProviderID != "provider.example.echo" {
		t.Fatalf("provider should remain free-local, got %s", m.ProviderID)
	}
	// steps runtime produces multi-step output
	if m.Output == "" {
		t.Fatal("empty output")
	}
}

func TestH4_RuntimeSwapExcludeEcho(t *testing.T) {
	res := boot(t)
	m := submit(t, res.Kernel, map[string]string{
		"route.excludeRuntime": "runtime.example.echo",
	})
	if m.RuntimeID != "runtime.example.steps" {
		t.Fatalf("runtime %s", m.RuntimeID)
	}
}

func TestH4_FullMatrix(t *testing.T) {
	// All four combinations succeed without kernel edits — labels only.
	res := boot(t)
	cases := []struct {
		name     string
		labels   map[string]string
		wantProv types.PluginID
		wantRT   types.PluginID
	}{
		{"echo+echo", nil, "provider.example.echo", "runtime.example.echo"},
		{"echo+steps", map[string]string{"route.preferRuntime": "runtime.example.steps"}, "provider.example.echo", "runtime.example.steps"},
		{"budget+echo", map[string]string{
			"route.excludeProvider": "provider.example.echo",
			"route.preferLocal":     "false",
		}, "provider.example.budget", "runtime.example.echo"},
		{"budget+steps", map[string]string{
			"route.excludeProvider": "provider.example.echo",
			"route.preferLocal":     "false",
			"route.preferRuntime":   "runtime.example.steps",
		}, "provider.example.budget", "runtime.example.steps"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := submit(t, res.Kernel, tc.labels)
			if m.ProviderID != tc.wantProv || m.RuntimeID != tc.wantRT {
				t.Fatalf("got %s + %s want %s + %s", m.ProviderID, m.RuntimeID, tc.wantProv, tc.wantRT)
			}
		})
	}
}
