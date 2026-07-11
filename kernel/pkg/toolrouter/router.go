// Package toolrouter is Hermes unified tools surface (INV-08 / AESP-0015 INT-TOOLS).
// Hermes defines tools; runtimes consume them. Invocations are recorded for audit.
package toolrouter

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/types"
)

// Tool is a Hermes-defined tool descriptor.
type Tool struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Enabled     bool              `json:"enabled"`
	Labels      map[string]string `json:"labels,omitempty"`
	// Parameters is JSON Schema for the tool arguments (OpenAI tools format).
	Parameters map[string]any `json:"parameters,omitempty"`
}

// Invocation is an audit record of a tool call.
type Invocation struct {
	ID         string            `json:"id"`
	ToolID     string            `json:"toolId"`
	MissionID  types.MissionID   `json:"missionId,omitempty"`
	RuntimeID  types.PluginID    `json:"runtimeId,omitempty"`
	Input      map[string]any    `json:"input,omitempty"`
	Output     string            `json:"output,omitempty"`
	Status     string            `json:"status"` // ok|error|denied
	Error      string            `json:"error,omitempty"`
	At         time.Time         `json:"at"`
	Provenance map[string]string `json:"provenance,omitempty"`
}

// Handler executes a tool.
type Handler func(ctx context.Context, input map[string]any) (string, error)

// Router holds tool registry + invocation log.
type Router struct {
	mu    sync.Mutex
	tools map[string]Tool
	h     map[string]Handler
	log   []Invocation
	seq   int64
}

func New() *Router {
	r := &Router{tools: map[string]Tool{}, h: map[string]Handler{}}
	// Built-in platform tools (vendor-neutral)
	_ = r.Register(Tool{
		ID: "echo", Name: "echo", Description: "Echo input text", Enabled: true,
		Parameters: map[string]any{"type": "object", "properties": map[string]any{"text": map[string]any{"type": "string"}}},
	}, func(ctx context.Context, input map[string]any) (string, error) {
		if t, ok := input["text"].(string); ok {
			return t, nil
		}
		return fmt.Sprintf("%v", input), nil
	})
	_ = r.Register(Tool{
		ID: "time.now", Name: "time.now", Description: "UTC timestamp", Enabled: true,
		Parameters: map[string]any{"type": "object", "properties": map[string]any{}},
	}, func(ctx context.Context, input map[string]any) (string, error) {
		return time.Now().UTC().Format(time.RFC3339Nano), nil
	})
	return r
}

func (r *Router) Register(t Tool, h Handler) error {
	if t.ID == "" {
		return fmt.Errorf("tool id required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[t.ID] = t
	if h != nil {
		r.h[t.ID] = h
	}
	return nil
}

// Unregister removes a tool (MCP disconnect).
func (r *Router) Unregister(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.tools, id)
	delete(r.h, id)
	return nil
}

// Get returns a tool by id.
func (r *Router) Get(id string) (Tool, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	t, ok := r.tools[id]
	return t, ok
}

func (r *Router) List() []Tool {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]Tool, 0, len(r.tools))
	for _, t := range r.tools {
		out = append(out, t)
	}
	return out
}

// Invoke runs a tool and records the invocation (AESP tool invocation records).
func (r *Router) Invoke(ctx context.Context, toolID string, mission types.MissionID, runtime types.PluginID, input map[string]any) (Invocation, error) {
	r.mu.Lock()
	t, ok := r.tools[toolID]
	h := r.h[toolID]
	r.mu.Unlock()
	inv := Invocation{
		ToolID: toolID, MissionID: mission, RuntimeID: runtime, Input: input,
		At: time.Now().UTC(), Status: "ok",
	}
	r.mu.Lock()
	r.seq++
	inv.ID = fmt.Sprintf("inv_%d", r.seq)
	r.mu.Unlock()

	if !ok || !t.Enabled {
		inv.Status = "denied"
		inv.Error = "tool not found or disabled"
		r.append(inv)
		return inv, fmt.Errorf("%s", inv.Error)
	}
	if h == nil {
		inv.Status = "error"
		inv.Error = "no handler"
		r.append(inv)
		return inv, fmt.Errorf("no handler")
	}
	out, err := h(ctx, input)
	if err != nil {
		inv.Status = "error"
		inv.Error = err.Error()
		r.append(inv)
		return inv, err
	}
	inv.Output = out
	r.append(inv)
	return inv, nil
}

func (r *Router) append(inv Invocation) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.log = append(r.log, inv)
	if len(r.log) > 500 {
		r.log = r.log[len(r.log)-500:]
	}
}

func (r *Router) Invocations(limit int) []Invocation {
	r.mu.Lock()
	defer r.mu.Unlock()
	if limit <= 0 || limit > len(r.log) {
		limit = len(r.log)
	}
	start := len(r.log) - limit
	out := make([]Invocation, limit)
	copy(out, r.log[start:])
	return out
}
