package openaicompat

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/plugin"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/provider"
)

func TestCompleteAgainstMock(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{{"message": map[string]any{"content": "hello-from-compat"}}},
			"usage":   map[string]any{"prompt_tokens": 3, "completion_tokens": 5},
		})
	}))
	defer srv.Close()

	p, err := NewProvider(plugin.Manifest{
		Metadata: plugin.Metadata{ID: "provider.openai.compat", Version: "0.1.0"},
		Spec: map[string]any{
			"baseURL": srv.URL + "/v1",
			"models":  []any{map[string]any{"id": "test-model"}},
			"local":   true,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	resp, err := p.Complete(context.Background(), provider.CompletionRequest{
		Model:    "test-model",
		Messages: []provider.Message{{Role: "user", Content: "hi"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Content != "hello-from-compat" || resp.TokensOut != 5 {
		t.Fatalf("%+v", resp)
	}
}
