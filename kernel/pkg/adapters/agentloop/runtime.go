// Package agentloop is the multi-turn tool-calling agent runtime.
// Provider supplies models; this runtime executes the agent loop (INV-01).
package agentloop

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/plugin"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/provider"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/runtime"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/toolrouter"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/types"
)

// Completer is injected by the kernel (never vendor-owned).
type Completer func(ctx context.Context, req provider.CompletionRequest) (provider.CompletionResponse, error)

// ToolInvoker runs a Hermes tool by id.
type ToolInvoker func(ctx context.Context, toolID string, mission types.MissionID, input map[string]any) (string, error)

// EventHook optional progress (kernel may publish to bus).
type EventHook func(ctx context.Context, typ string, data map[string]any)

// Runtime is a tool-calling agent harness.
type Runtime struct {
	manifest plugin.Manifest
	desc     runtime.Descriptor
	Complete Completer
	Invoke   ToolInvoker
	OnEvent  EventHook
	// MaxSteps overrides envelope budget when > 0.
	MaxSteps int
}

func NewRuntime(m plugin.Manifest) (*Runtime, error) {
	d := runtime.Descriptor{
		ID: m.Metadata.ID, Version: m.Metadata.Version, SandboxTier: "process-pty",
		CapabilitiesIn:  []types.Capability{"coding", "tools", "reasoning"},
		CapabilitiesOut: []types.Capability{"artifacts", "plan", "tools"},
	}
	if m.Spec != nil {
		if s, ok := m.Spec["sandboxTier"].(string); ok {
			d.SandboxTier = s
		}
		if caps := stringSliceCaps(m.Spec["capabilitiesIn"]); len(caps) > 0 {
			d.CapabilitiesIn = caps
		}
		if caps := stringSliceCaps(m.Spec["capabilitiesOut"]); len(caps) > 0 {
			d.CapabilitiesOut = caps
		}
	}
	return &Runtime{manifest: m, desc: d, MaxSteps: 12}, nil
}

func RuntimeFactory(m plugin.Manifest) (any, error) { return NewRuntime(m) }

func (r *Runtime) ID() types.PluginID                               { return r.manifest.Metadata.ID }
func (r *Runtime) Health(ctx context.Context) error                 { return nil }
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
	missionID := types.MissionID("")
	if env.Correlation != nil {
		missionID = types.MissionID(env.Correlation["missionId"])
	}

	maxSteps := r.MaxSteps
	if env.Budget != nil {
		if v, ok := env.Budget["maxSteps"].(int); ok && v > 0 {
			maxSteps = v
		}
		if v, ok := env.Budget["maxSteps"].(float64); ok && v > 0 {
			maxSteps = int(v)
		}
	}

	tools := toolsFromEnvelope(env.Tools)
	system := buildSystemPrompt(env)

	msgs := []provider.Message{
		{Role: "system", Content: system},
	}
	// Inject compact memory
	if len(env.Memory) > 0 {
		var b strings.Builder
		b.WriteString("Relevant memory:\n")
		for i, m := range env.Memory {
			if i >= 8 {
				break
			}
			if c, ok := m["content"].(string); ok && c != "" {
				b.WriteString("- ")
				b.WriteString(truncate(c, 400))
				b.WriteString("\n")
			}
		}
		msgs = append(msgs, provider.Message{Role: "system", Content: b.String()})
	}
	// Skills
	if env.Preferences != nil {
		if sk, ok := env.Preferences["skills"].(string); ok && sk != "" {
			msgs = append(msgs, provider.Message{Role: "system", Content: "Skills:\n" + truncate(sk, 4000)})
		}
	}
	msgs = append(msgs, provider.Message{Role: "user", Content: prompt})

	if r.Complete == nil {
		return runtime.Result{Status: "failed", Output: "no completer wired", StepsUsed: 0}, fmt.Errorf("no completer")
	}

	var tokens int64
	var cost float64
	var transcript []string
	steps := 0

	for steps < maxSteps {
		steps++
		if r.OnEvent != nil {
			r.OnEvent(ctx, "agent.step", map[string]any{"step": steps, "maxSteps": maxSteps})
		}
		// Compact context if huge
		msgs = compactMessages(msgs, 24)

		resp, err := r.Complete(ctx, provider.CompletionRequest{
			Model:            model,
			Messages:         msgs,
			Tools:            tools,
			ToolChoice:       "auto",
			MaxTokens:        2048,
			CredentialHandle: handle,
			Correlation:      env.Correlation,
		})
		if err != nil {
			// Fallback: retry once without tools (some models reject tools)
			if len(tools) > 0 && strings.Contains(err.Error(), "tool") {
				resp, err = r.Complete(ctx, provider.CompletionRequest{
					Model: model, Messages: msgs, MaxTokens: 2048,
					CredentialHandle: handle, Correlation: env.Correlation,
				})
			}
			if err != nil {
				return runtime.Result{
					Status: "failed", Output: strings.Join(transcript, "\n") + "\nerror: " + err.Error(),
					StepsUsed: steps, TokensUsed: tokens, CostUSD: cost,
				}, err
			}
		}
		tokens += resp.TokensIn + resp.TokensOut
		cost += resp.CostUSD

		if len(resp.ToolCalls) == 0 {
			out := strings.TrimSpace(resp.Content)
			if out == "" {
				out = "(empty model response)"
			}
			transcript = append(transcript, out)
			if r.OnEvent != nil {
				r.OnEvent(ctx, "agent.final", map[string]any{"steps": steps, "chars": len(out)})
			}
			return runtime.Result{
				Status: "succeeded", Output: strings.Join(transcript, "\n\n"),
				StepsUsed: steps, TokensUsed: tokens, CostUSD: cost,
			}, nil
		}

		// Append assistant message with tool_calls
		asst := provider.Message{Role: "assistant", Content: resp.Content, ToolCalls: resp.ToolCalls}
		msgs = append(msgs, asst)

		for _, tc := range resp.ToolCalls {
			name := tc.Function.Name
			argsRaw := tc.Function.Arguments
			var args map[string]any
			if argsRaw != "" {
				_ = json.Unmarshal([]byte(argsRaw), &args)
			}
			if args == nil {
				args = map[string]any{}
			}
			if r.OnEvent != nil {
				r.OnEvent(ctx, "agent.tool", map[string]any{"tool": name, "step": steps})
			}
			result := ""
			if r.Invoke != nil {
				out, err := r.Invoke(ctx, name, missionID, args)
				if err != nil {
					result = "error: " + err.Error()
				} else {
					result = out
				}
			} else {
				result = "error: no tool invoker"
			}
			result = truncate(result, 12000)
			transcript = append(transcript, fmt.Sprintf("[tool:%s] %s", name, truncate(result, 500)))
			msgs = append(msgs, provider.Message{
				Role: "tool", Content: result, ToolCallID: tc.ID, Name: name,
			})
		}
	}

	// Max steps — ask for final summary without tools
	msgs = append(msgs, provider.Message{
		Role: "user", Content: "Stop calling tools. Provide your final answer based on the tool results so far.",
	})
	resp, err := r.Complete(ctx, provider.CompletionRequest{
		Model: model, Messages: msgs, MaxTokens: 1024,
		CredentialHandle: handle, Correlation: env.Correlation,
	})
	if err != nil {
		return runtime.Result{
			Status: "succeeded", Output: strings.Join(transcript, "\n\n") + "\n(max steps; model summary failed)",
			StepsUsed: steps, TokensUsed: tokens, CostUSD: cost,
		}, nil
	}
	tokens += resp.TokensIn + resp.TokensOut
	final := strings.TrimSpace(resp.Content)
	if final != "" {
		transcript = append(transcript, final)
	}
	return runtime.Result{
		Status: "succeeded", Output: strings.Join(transcript, "\n\n"),
		StepsUsed: steps, TokensUsed: tokens, CostUSD: cost,
	}, nil
}

func toolsFromEnvelope(raw []map[string]any) []provider.ToolSpec {
	var out []provider.ToolSpec
	for _, t := range raw {
		id, _ := t["id"].(string)
		if id == "" {
			id, _ = t["name"].(string)
		}
		if id == "" {
			continue
		}
		desc, _ := t["description"].(string)
		params, _ := t["parameters"].(map[string]any)
		if params == nil {
			params = map[string]any{"type": "object", "properties": map[string]any{}}
		}
		out = append(out, provider.ToolSpec{
			Type: "function",
			Function: provider.ToolFunction{
				Name: id, Description: desc, Parameters: params,
			},
		})
	}
	return out
}

func buildSystemPrompt(env runtime.ContextEnvelope) string {
	var b strings.Builder
	b.WriteString("You are Hermes Agent OS — a capable tool-using agent.\n")
	b.WriteString("Use tools when they help. Prefer concrete actions over speculation.\n")
	b.WriteString("When done, answer clearly without further tool calls.\n")
	if env.Workspace != nil {
		if root, ok := env.Workspace["root"].(string); ok && root != "" {
			b.WriteString("Workspace root: ")
			b.WriteString(root)
			b.WriteString("\n")
		}
	}
	if len(env.Tools) > 0 {
		b.WriteString("Available tools: ")
		names := make([]string, 0, len(env.Tools))
		for _, t := range env.Tools {
			if id, ok := t["id"].(string); ok {
				names = append(names, id)
			}
		}
		b.WriteString(strings.Join(names, ", "))
		b.WriteString("\n")
	}
	return b.String()
}

func compactMessages(msgs []provider.Message, keepRecent int) []provider.Message {
	if len(msgs) <= keepRecent+2 {
		return msgs
	}
	// Keep first system messages + last N
	var head []provider.Message
	for i, m := range msgs {
		if m.Role == "system" && i < 4 {
			head = append(head, m)
			continue
		}
		break
	}
	start := len(msgs) - keepRecent
	if start < len(head) {
		start = len(head)
	}
	out := append([]provider.Message{}, head...)
	out = append(out, msgs[start:]...)
	return out
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
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

// Ensure toolrouter import used when building schemas externally.
var _ = toolrouter.Tool{}

var _ runtime.Runtime = (*Runtime)(nil)
