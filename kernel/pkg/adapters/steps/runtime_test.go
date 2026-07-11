package steps

import (
	"context"
	"testing"

	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/plugin"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/provider"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/runtime"
)

func TestStepsRuntimeTwoSteps(t *testing.T) {
	rt, err := NewRuntime(plugin.Manifest{
		Metadata: plugin.Metadata{ID: "runtime.example.steps", Version: "0.0.1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	rt.Complete = func(ctx context.Context, req provider.CompletionRequest) (provider.CompletionResponse, error) {
		calls++
		return provider.CompletionResponse{
			ProviderID: "p", ModelID: "m", Content: "ok", TokensIn: 1, TokensOut: 1,
		}, nil
	}
	res, err := rt.Execute(context.Background(), runtime.ContextEnvelope{
		Prompt: "interchangeability",
		Correlation: map[string]string{
			"providerId": "p", "modelId": "m",
		},
	})
	if err != nil || res.Status != "succeeded" {
		t.Fatalf("%+v %v", res, err)
	}
	if res.StepsUsed != 2 || calls != 2 {
		t.Fatalf("steps=%d calls=%d", res.StepsUsed, calls)
	}
	if res.Output == "" {
		t.Fatal("empty output")
	}
}
