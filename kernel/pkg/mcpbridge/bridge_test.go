package mcpbridge

import (
	"context"
	"testing"

	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/toolrouter"
)

func TestListCall(t *testing.T) {
	b := New(toolrouter.New())
	if len(b.ListTools()) < 1 {
		t.Fatal("empty")
	}
	out, err := b.CallTool(context.Background(), "echo", map[string]any{"text": "mcp"})
	if err != nil || out != "mcp" {
		t.Fatal(err, out)
	}
}
