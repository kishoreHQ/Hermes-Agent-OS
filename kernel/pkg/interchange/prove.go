// Package interchange provides the H4 interchangeability proof harness.
package interchange

import (
	"context"
	"fmt"
	"strings"

	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/bootstrap"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/host"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/plugin"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/types"
)

// Case is one swap scenario.
type Case struct {
	Name            string
	Labels          map[string]string
	ExpectProvider  types.PluginID
	ExpectRuntime   types.PluginID
}

// Result of one case.
type CaseResult struct {
	Case     Case
	OK       bool
	Mission  host.Mission
	Error    string
	RouteEvt map[string]any
}

// Report is the full H4 proof output.
type Report struct {
	Providers int
	Runtimes  int
	Cases     []CaseResult
	Passed    int
	Failed    int
}

// DefaultCases covers the provider×runtime matrix (default runtime = agent-loop).
func DefaultCases() []Case {
	return []Case{
		{
			Name: "default free-local + agent-loop runtime",
			ExpectProvider: "provider.example.echo", ExpectRuntime: "runtime.agent.loop",
		},
		{
			Name: "prefer steps runtime",
			Labels: map[string]string{"route.preferRuntime": "runtime.example.steps"},
			ExpectProvider: "provider.example.echo", ExpectRuntime: "runtime.example.steps",
		},
		{
			Name: "prefer echo runtime (legacy harness)",
			Labels: map[string]string{"route.preferRuntime": "runtime.example.echo"},
			ExpectProvider: "provider.example.echo", ExpectRuntime: "runtime.example.echo",
		},
		{
			Name: "exclude free-local → budget provider + agent-loop",
			Labels: map[string]string{
				"route.excludeProvider": "provider.example.echo",
				"route.preferLocal":     "false",
			},
			ExpectProvider: "provider.example.budget", ExpectRuntime: "runtime.agent.loop",
		},
		{
			Name: "budget provider + steps runtime",
			Labels: map[string]string{
				"route.excludeProvider": "provider.example.echo",
				"route.preferLocal":     "false",
				"route.preferRuntime":   "runtime.example.steps",
			},
			ExpectProvider: "provider.example.budget", ExpectRuntime: "runtime.example.steps",
		},
	}
}

// Run executes the H4 proof against a seeded bootstrap kernel.
func Run(ctx context.Context) (*Report, error) {
	res, err := bootstrap.New(bootstrap.Options{SeedBuiltins: true, PluginRoots: []string{"/no-disk"}})
	if err != nil {
		return nil, err
	}
	// Prefer disk plugins when available for real prove-h4 from repo root
	if disk, err2 := bootstrap.New(bootstrap.Options{SeedBuiltins: true}); err2 == nil {
		if len(disk.Registry.List(plugin.KindRuntime)) >= 2 {
			res = disk
		}
	}

	rep := &Report{
		Providers: len(res.Registry.List(plugin.KindProvider)),
		Runtimes:  len(res.Registry.List(plugin.KindRuntime)),
	}
	if rep.Providers < 2 || rep.Runtimes < 2 {
		return rep, fmt.Errorf("H4 requires ≥2 providers and ≥2 runtimes (got p=%d r=%d)", rep.Providers, rep.Runtimes)
	}

	for _, c := range DefaultCases() {
		cr := CaseResult{Case: c}
		id, err := res.Kernel.SubmitMission(ctx, host.Mission{
			Goal:         "H4 prove: " + c.Name,
			RequiredCaps: []types.Capability{"coding", "tools"},
			Labels:       c.Labels,
		})
		if err != nil {
			cr.Error = err.Error()
			rep.Cases = append(rep.Cases, cr)
			rep.Failed++
			continue
		}
		m, err := res.Kernel.GetMission(ctx, id)
		if err != nil {
			cr.Error = err.Error()
			rep.Cases = append(rep.Cases, cr)
			rep.Failed++
			continue
		}
		cr.Mission = m
		evs, _ := res.Kernel.Replay(ctx, id)
		for _, e := range evs {
			if e.Type == "route.decided" {
				cr.RouteEvt = e.Data
				break
			}
		}
		if m.State != host.StateSucceeded {
			cr.Error = "state=" + string(m.State) + " output=" + m.Output
		} else if m.ProviderID != c.ExpectProvider || m.RuntimeID != c.ExpectRuntime {
			cr.Error = fmt.Sprintf("got %s+%s want %s+%s", m.ProviderID, m.RuntimeID, c.ExpectProvider, c.ExpectRuntime)
		} else if cr.RouteEvt == nil {
			cr.Error = "missing route.decided"
		} else {
			cr.OK = true
			rep.Passed++
		}
		if !cr.OK {
			rep.Failed++
		}
		rep.Cases = append(rep.Cases, cr)
	}
	return rep, nil
}

// Format human-readable report.
func Format(rep *Report) string {
	var b strings.Builder
	fmt.Fprintf(&b, "H4 Interchangeability Proof\n")
	fmt.Fprintf(&b, "  providers: %d  runtimes: %d\n", rep.Providers, rep.Runtimes)
	fmt.Fprintf(&b, "  passed: %d  failed: %d\n", rep.Passed, rep.Failed)
	for _, c := range rep.Cases {
		mark := "FAIL"
		if c.OK {
			mark = "PASS"
		}
		fmt.Fprintf(&b, "  [%s] %s\n", mark, c.Case.Name)
		if c.OK {
			fmt.Fprintf(&b, "         provider=%s runtime=%s reason=%v\n",
				c.Mission.ProviderID, c.Mission.RuntimeID, c.RouteEvt["reason"])
		} else {
			fmt.Fprintf(&b, "         %s\n", c.Error)
		}
	}
	if rep.Failed == 0 && rep.Passed > 0 {
		fmt.Fprintf(&b, "RESULT: PASS — plugins interchangeable without kernel edit\n")
	} else {
		fmt.Fprintf(&b, "RESULT: FAIL\n")
	}
	return b.String()
}
