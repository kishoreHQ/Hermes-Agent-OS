// Package evaluation is the product evaluation harness (H5).
package evaluation

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/bootstrap"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/host"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/types"
)

// Case is a golden mission evaluation.
type Case struct {
	ID           string
	Goal         string
	Capabilities []types.Capability
	Labels       map[string]string
	// ExpectState if non-empty
	ExpectState host.MissionState
	// ExpectProvider substring match optional
	ExpectProvider string
	// ExpectRouteEvent requires route.decided
	ExpectRouteEvent bool
	// ExpectMode in labels path
	ExpectMode string
}

// CaseResult is one evaluation outcome.
type CaseResult struct {
	Case     Case
	OK       bool
	Duration time.Duration
	Mission  host.Mission
	Error    string
}

// Report aggregates cases.
type Report struct {
	Suite   string
	Cases   []CaseResult
	Passed  int
	Failed  int
	Elapsed time.Duration
}

// DefaultSuite covers core platform behaviors for H5.
func DefaultSuite() []Case {
	return []Case{
		{
			ID: "eval.happy-path", Goal: "eval happy path",
			Capabilities:     []types.Capability{"coding", "tools"},
			ExpectState:      host.StateSucceeded,
			ExpectProvider:   "provider.example.echo",
			ExpectRouteEvent: true,
		},
		{
			ID: "eval.observe-mode", Goal: "eval observe",
			Capabilities: []types.Capability{"coding"},
			Labels:       map[string]string{"security.mode": "observe"},
			ExpectState:  host.StateSucceeded, // observe succeeds without runtime execute
			ExpectMode:   "observe",
		},
		{
			ID: "eval.assist-external", Goal: "eval assist external",
			Capabilities: []types.Capability{"coding"},
			Labels: map[string]string{
				"security.mode":           "assist",
				"security.externalAction": "true",
			},
			ExpectState: host.StateAwaitingApproval,
		},
		{
			ID: "eval.steps-runtime", Goal: "eval steps",
			Capabilities:   []types.Capability{"coding", "tools"},
			Labels:         map[string]string{"route.preferRuntime": "runtime.example.steps"},
			ExpectState:    host.StateSucceeded,
			ExpectProvider: "provider.example.echo",
		},
	}
}

// Run executes the suite against a bootstrapped kernel.
func Run(ctx context.Context, cases []Case) (*Report, error) {
	if len(cases) == 0 {
		cases = DefaultSuite()
	}
	res, err := bootstrap.New(bootstrap.Options{SeedBuiltins: true})
	if err != nil {
		return nil, err
	}
	start := time.Now()
	rep := &Report{Suite: "hermes-h5-default"}
	for _, c := range cases {
		cr := runCase(ctx, res.Kernel, c)
		if cr.OK {
			rep.Passed++
		} else {
			rep.Failed++
		}
		rep.Cases = append(rep.Cases, cr)
	}
	rep.Elapsed = time.Since(start)
	return rep, nil
}

func runCase(ctx context.Context, k interface {
	SubmitMission(ctx context.Context, m host.Mission) (types.MissionID, error)
	GetMission(ctx context.Context, id types.MissionID) (host.Mission, error)
	Replay(ctx context.Context, id types.MissionID) ([]host.Event, error)
}, c Case) CaseResult {
	cr := CaseResult{Case: c}
	t0 := time.Now()
	id, err := k.SubmitMission(ctx, host.Mission{
		Goal: c.Goal, RequiredCaps: c.Capabilities, Labels: c.Labels,
	})
	cr.Duration = time.Since(t0)
	if err != nil {
		cr.Error = err.Error()
		return cr
	}
	m, err := k.GetMission(ctx, id)
	if err != nil {
		cr.Error = err.Error()
		return cr
	}
	cr.Mission = m
	if c.ExpectState != "" && m.State != c.ExpectState {
		cr.Error = fmt.Sprintf("state got %s want %s", m.State, c.ExpectState)
		return cr
	}
	if c.ExpectProvider != "" && !strings.Contains(string(m.ProviderID), c.ExpectProvider) && m.State == host.StateSucceeded {
		// observe may not set provider if short-circuited before route — still check route event
		if m.ProviderID != "" && string(m.ProviderID) != c.ExpectProvider {
			cr.Error = fmt.Sprintf("provider %s want %s", m.ProviderID, c.ExpectProvider)
			return cr
		}
	}
	if c.ExpectRouteEvent || c.ExpectMode != "" {
		evs, _ := k.Replay(ctx, id)
		var sawRoute, sawMode bool
		for _, e := range evs {
			if e.Type == "route.decided" {
				sawRoute = true
			}
			if e.Type == "security.evaluated" {
				sawMode = true
				if c.ExpectMode != "" {
					if mode, _ := e.Data["mode"].(string); mode != c.ExpectMode {
						cr.Error = fmt.Sprintf("mode %s want %s", mode, c.ExpectMode)
						return cr
					}
				}
			}
		}
		if c.ExpectRouteEvent && !sawRoute {
			// observe mode still routes then stops
			cr.Error = "missing route.decided"
			return cr
		}
		if c.ExpectMode != "" && !sawMode {
			cr.Error = "missing security.evaluated"
			return cr
		}
	}
	cr.OK = true
	return cr
}

// Format report.
func Format(rep *Report) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Evaluation suite: %s\n", rep.Suite)
	fmt.Fprintf(&b, "  passed: %d  failed: %d  elapsed: %s\n", rep.Passed, rep.Failed, rep.Elapsed.Round(time.Millisecond))
	for _, c := range rep.Cases {
		mark := "FAIL"
		if c.OK {
			mark = "PASS"
		}
		fmt.Fprintf(&b, "  [%s] %s (%s)\n", mark, c.Case.ID, c.Duration.Round(time.Microsecond))
		if !c.OK {
			fmt.Fprintf(&b, "         %s\n", c.Error)
		}
	}
	if rep.Failed == 0 && rep.Passed > 0 {
		fmt.Fprintf(&b, "RESULT: PASS\n")
	} else {
		fmt.Fprintf(&b, "RESULT: FAIL\n")
	}
	return b.String()
}
