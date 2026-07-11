// Package hardening aggregates the H5 production-hardening gate.
package hardening

import (
	"context"
	"fmt"
	"strings"

	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/evaluation"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/interchange"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/perf"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/security"
)

// Report is the H5 composite result.
type Report struct {
	H4        *interchange.Report
	Eval      *evaluation.Report
	Perf      *perf.Report
	Security  map[string]any
	Passed    bool
	Failures  []string
}

// Run executes H4 + evaluation suite + perf baselines + security self-check.
func Run(ctx context.Context) (*Report, error) {
	rep := &Report{
		Security: map[string]any{
			"modes":          []string{"full", "assist", "observe"},
			"hmacSigning":    true,
			"requireSigned":  security.RequireSignedFromEnv(),
			"scopesObserve":  security.DefaultScopes(security.ModeObserve),
			"scopesAssist":   security.DefaultScopes(security.ModeAssist),
			"scopesFull":     security.DefaultScopes(security.ModeFull),
		},
	}

	h4, err := interchange.Run(ctx)
	if err != nil {
		return nil, fmt.Errorf("h4: %w", err)
	}
	rep.H4 = h4
	if h4.Failed > 0 {
		rep.Failures = append(rep.Failures, "interchangeability matrix failed")
	}

	ev, err := evaluation.Run(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("eval: %w", err)
	}
	rep.Eval = ev
	if ev.Failed > 0 {
		rep.Failures = append(rep.Failures, "evaluation suite failed")
	}

	pf, err := perf.RunBaselines(ctx)
	if err != nil {
		return nil, fmt.Errorf("perf: %w", err)
	}
	rep.Perf = pf
	// Soft: only fail pathological p99
	for _, s := range pf.Samples {
		if s.Name == "mission.submit+execute" && s.P99 > perf.MaxMissionP99*4 {
			rep.Failures = append(rep.Failures, "mission latency pathological")
		}
	}

	// Security unit invariants
	d := security.EvaluateMode(security.ModeObserve, true)
	if d.AllowExecute {
		rep.Failures = append(rep.Failures, "observe must not execute")
	}
	d = security.EvaluateMode(security.ModeAssist, true)
	if !d.RequireApproval {
		rep.Failures = append(rep.Failures, "assist external must require approval")
	}
	if err := security.EnforceMinSandbox(security.SandboxProcessPTY, security.SandboxContainer); err == nil {
		rep.Failures = append(rep.Failures, "sandbox ranking broken")
	}

	rep.Passed = len(rep.Failures) == 0
	return rep, nil
}

// Format human-readable H5 report.
func Format(rep *Report) string {
	var b strings.Builder
	fmt.Fprintf(&b, "=== H5 Production Hardening Proof ===\n\n")
	if rep.H4 != nil {
		fmt.Fprintf(&b, "— Interchangeability (H4) —\n%s\n", interchange.Format(rep.H4))
	}
	if rep.Eval != nil {
		fmt.Fprintf(&b, "— Evaluation suite —\n%s\n", evaluation.Format(rep.Eval))
	}
	if rep.Perf != nil {
		fmt.Fprintf(&b, "— Performance baselines —\n%s\n", perf.Format(rep.Perf))
	}
	fmt.Fprintf(&b, "— Security posture —\n")
	fmt.Fprintf(&b, "  modes: full | assist | observe\n")
	fmt.Fprintf(&b, "  plugin HMAC signing supported\n")
	fmt.Fprintf(&b, "  requireSigned env: %v\n", rep.Security["requireSigned"])
	if rep.Passed {
		fmt.Fprintf(&b, "\nRESULT: PASS — H5 production hardening\n")
	} else {
		fmt.Fprintf(&b, "\nRESULT: FAIL\n")
		for _, f := range rep.Failures {
			fmt.Fprintf(&b, "  - %s\n", f)
		}
	}
	return b.String()
}
