package echo

import (
	"context"
	"testing"

	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/plugin"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/provider"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/runtime"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/types"
)

func TestProviderComplete(t *testing.T) {
	p, err := NewProvider(plugin.Manifest{
		Metadata: plugin.Metadata{ID: "provider.example.echo", Version: "0.0.1"},
		Spec: map[string]any{
			"capabilities": []any{"coding", "tools"},
			"local":        true,
			"costTier":     "free-local",
			"models":       []any{map[string]any{"id": "echo-1"}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	resp, err := p.Complete(context.Background(), provider.CompletionRequest{
		Model:    "echo-1",
		Messages: []provider.Message{{Role: "user", Content: "hello mission"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Content == "" || resp.ProviderID != "provider.example.echo" {
		t.Fatalf("%+v", resp)
	}
}

func TestRuntimeExecuteWithCompleter(t *testing.T) {
	rt, err := NewRuntime(plugin.Manifest{
		Metadata: plugin.Metadata{ID: "runtime.example.echo", Version: "0.0.1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	rt.Complete = func(ctx context.Context, req provider.CompletionRequest) (provider.CompletionResponse, error) {
		return provider.CompletionResponse{
			ProviderID: "provider.example.echo", ModelID: "echo-1",
			Content: "ok", TokensIn: 1, TokensOut: 1,
		}, nil
	}
	res, err := rt.Execute(context.Background(), runtime.ContextEnvelope{
		Prompt: "build it",
		Mission: map[string]any{"goal": "build it"},
		Correlation: map[string]string{
			"providerId": "provider.example.echo",
			"modelId":    "echo-1",
		},
		Memory: []map[string]any{{"id": "m1", "content": "prior"}},
	})
	if err != nil || res.Status != "succeeded" {
		t.Fatalf("%+v %v", res, err)
	}
	if res.StepsUsed != 1 {
		t.Fatal(res.StepsUsed)
	}
	_ = types.Capability("coding")
}
