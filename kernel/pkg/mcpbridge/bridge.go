// Package mcpbridge provides MCP-aligned tool descriptors and invoke bridge (AESP-0015 / INT-MCP).
// This is not a full MCP wire server; it exposes MCP-shaped tool lists backed by Hermes toolrouter.
package mcpbridge

import (
	"context"
	"fmt"

	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/toolrouter"
)

// ToolDesc is MCP-style tool metadata.
type ToolDesc struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	InputSchema map[string]any `json:"inputSchema,omitempty"`
}

// Bridge lists/invokes tools in MCP-shaped form.
type Bridge struct {
	Tools *toolrouter.Router
}

func New(tools *toolrouter.Router) *Bridge {
	return &Bridge{Tools: tools}
}

// ListTools returns MCP-style tool descriptors.
func (b *Bridge) ListTools() []ToolDesc {
	if b.Tools == nil {
		return nil
	}
	var out []ToolDesc
	for _, t := range b.Tools.List() {
		out = append(out, ToolDesc{
			Name: t.ID, Description: t.Description,
			InputSchema: map[string]any{"type": "object"},
		})
	}
	return out
}

// CallTool invokes via Hermes tool router (authz at host layer later).
func (b *Bridge) CallTool(ctx context.Context, name string, args map[string]any) (string, error) {
	if b.Tools == nil {
		return "", fmt.Errorf("no tools")
	}
	inv, err := b.Tools.Invoke(ctx, name, "", "", args)
	if err != nil {
		return "", err
	}
	return inv.Output, nil
}
