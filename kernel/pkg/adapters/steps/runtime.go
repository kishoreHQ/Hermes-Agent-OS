// Package steps is a second example runtime (H4 interchangeability).
// Distinct harness behavior from echo — multi-step, still vendor-neutral.
package steps

import (
	"context"
	"fmt"
	"strings"

	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/plugin"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/provider"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/runtime"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/types"
)

// Completer is injected by the kernel (same contract as echo).
type Completer func(ctx context.Context, req provider.CompletionRequest) (provider.CompletionResponse, error)

// Runtime executes a two-step plan: plan → act via Completer.
type Runtime struct {
	manifest plugin.Manifest
	desc     runtime.Descriptor
	Complete Completer
}

func NewRuntime(m plugin.Manifest) (*Runtime, error) {
	d := runtime.Descriptor{
		ID:          m.Metadata.ID,
		Version:     m.Metadata.Version,
		SandboxTier: "container",
	}
	if m.Spec != nil {
		if s, ok := m.Spec["sandboxTier"].(string); ok {
			d.SandboxTier = s
		}
		d.CapabilitiesIn = stringSliceCaps(m.Spec["capabilitiesIn"])
		d.CapabilitiesOut = stringSliceCaps(m.Spec["capabilitiesOut"])
	}
	if len(d.CapabilitiesIn) == 0 {
		d.CapabilitiesIn = []types.Capability{"coding", "tools"}
	}
	if len(d.CapabilitiesOut) == 0 {
		d.CapabilitiesOut = []types.Capability{"artifacts", "plan"}
	}
	return &Runtime{manifest: m, desc: d}, nil
}

func RuntimeFactory(m plugin.Manifest) (any, error) {
	return NewRuntime(m)
}

func (r *Runtime) ID() types.PluginID { return r.manifest.Metadata.ID }

func (r *Runtime) Health(ctx context.Context) error { return nil }

func (r *Runtime) Describe(ctx context.Context) (runtime.Descriptor, error) {
	return r.desc, nil
}

func (r *Runtime) Execute(ctx context.Context, env runtime.ContextEnvelope) (runtime.Result, error) {
	prompt := env.Prompt
	if prompt == "" {
		if g, ok := env.Mission["goal"].(string); ok {
			prompt = g
		}
	}
	model := ""
	if env.Correlation != nil {
		model = env.Correlation["modelId"]
	}
	handle := ""
	for _, c := range env.Credentials {
		if h, ok := c["handle"].(string); ok {
			handle = h
			break
		}
	}

	var tokens int64
	var cost float64
	var parts []string

	// Step 1 — plan
	plan := fmt.Sprintf("plan: accomplish %q in bounded steps", strings.TrimSpace(prompt))
	if r.Complete != nil {
		resp, err := r.Complete(ctx, provider.CompletionRequest{
			Model: model,
			Messages: []provider.Message{
				{Role: "system", Content: "You are a Hermes multi-step planner. Reply with one short plan line."},
				{Role: "user", Content: "Plan: " + prompt},
			},
			MaxTokens:        256,
			CredentialHandle: handle,
			Correlation:      env.Correlation,
		})
		if err != nil {
			return runtime.Result{Status: "failed", Output: err.Error(), StepsUsed: 1}, err
		}
		plan = resp.Content
		tokens += resp.TokensIn + resp.TokensOut
		cost += resp.CostUSD
	}
	parts = append(parts, "[step1/plan] "+plan)

	// Step 2 — act
	act := fmt.Sprintf("act: execute plan for %q", strings.TrimSpace(prompt))
	if r.Complete != nil {
		resp, err := r.Complete(ctx, provider.CompletionRequest{
			Model: model,
			Messages: []provider.Message{
				{Role: "system", Content: "You are a Hermes multi-step actor. Execute the plan."},
				{Role: "user", Content: "Plan was: " + plan + "\nGoal: " + prompt},
			},
			MaxTokens:        512,
			CredentialHandle: handle,
			Correlation:      env.Correlation,
		})
		if err != nil {
			return runtime.Result{Status: "failed", Output: strings.Join(parts, "\n") + "\n" + err.Error(), StepsUsed: 2}, err
		}
		act = resp.Content
		tokens += resp.TokensIn + resp.TokensOut
		cost += resp.CostUSD
	}
	parts = append(parts, "[step2/act] "+act)
	parts = append(parts, fmt.Sprintf("[runtime:%s steps=2]", r.ID()))

	return runtime.Result{
		Status:     "succeeded",
		Output:     strings.Join(parts, "\n"),
		StepsUsed:  2,
		TokensUsed: tokens,
		CostUSD:    cost,
	}, nil
}

func stringSliceCaps(v any) []types.Capability {
	var out []types.Capability
	switch t := v.(type) {
	case []any:
		for _, x := range t {
			if s, ok := x.(string); ok && s != "" {
				out = append(out, types.Capability(s))
			}
		}
	case []string:
		for _, s := range t {
			out = append(out, types.Capability(s))
		}
	}
	return out
}

var _ runtime.Runtime = (*Runtime)(nil)
