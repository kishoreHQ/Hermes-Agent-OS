package echo

import (
	"context"
	"fmt"

	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/plugin"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/provider"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/runtime"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/types"
)

// Completer is injected by the kernel so the runtime can call the routed provider
// without knowing vendor SDKs (runtime ≠ provider ownership).
type Completer func(ctx context.Context, req provider.CompletionRequest) (provider.CompletionResponse, error)

// Runtime is a minimal agent harness that runs one inference step via Completer.
type Runtime struct {
	manifest  plugin.Manifest
	desc      runtime.Descriptor
	Complete  Completer // optional; if nil, returns deterministic local output
}

func NewRuntime(m plugin.Manifest) (*Runtime, error) {
	d := runtime.Descriptor{
		ID:          m.Metadata.ID,
		Version:     m.Metadata.Version,
		SandboxTier: "process-pty",
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
		d.CapabilitiesOut = []types.Capability{"artifacts"}
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

	var content string
	var tokens int64
	var cost float64
	if r.Complete != nil {
		resp, err := r.Complete(ctx, provider.CompletionRequest{
			Model: model,
			Messages: []provider.Message{
				{Role: "system", Content: "You are a Hermes example runtime. Be concise."},
				{Role: "user", Content: prompt},
			},
			MaxTokens:        512,
			CredentialHandle: handle,
			Correlation:      env.Correlation,
		})
		if err != nil {
			return runtime.Result{Status: "failed", Output: err.Error()}, err
		}
		content = resp.Content
		tokens = resp.TokensIn + resp.TokensOut
		cost = resp.CostUSD
	} else {
		content = fmt.Sprintf("[runtime:%s] completed: %s", r.ID(), prompt)
		tokens = 8
	}

	// Surface shared memory that was provided (read path)
	memN := len(env.Memory)
	out := content
	if memN > 0 {
		out = fmt.Sprintf("%s\n(memory_entries=%d)", content, memN)
	}

	return runtime.Result{
		Status:     "succeeded",
		Output:     out,
		StepsUsed:  1,
		TokensUsed: tokens,
		CostUSD:    cost,
	}, nil
}

var _ runtime.Runtime = (*Runtime)(nil)
