// Package openaicompat is a real OpenAI-compatible HTTP provider plugin.
// Works with any OpenAI Chat Completions-compatible base URL (local or remote).
// Supports tools / tool_calls for the agent-loop runtime.
package openaicompat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/plugin"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/provider"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/types"
)

// SecretResolver returns API key material for a credential handle (kernel injects).
type SecretResolver func(ctx context.Context, handle string) (string, error)

// Provider talks to POST {baseURL}/v1/chat/completions (or baseURL already ending in /v1).
type Provider struct {
	manifest plugin.Manifest
	desc     provider.Descriptor
	baseURL  string
	apiKey   string // optional static (dev); prefer CredentialHandle + Resolve
	client   *http.Client
	Resolve  SecretResolver

	// health cache avoids stalling routing on every mission when remote is slow/unreachable
	healthOK  bool
	healthErr error
	healthAt  time.Time
	healthTTL time.Duration
}

func NewProvider(m plugin.Manifest) (*Provider, error) {
	p := &Provider{
		manifest:  m,
		client:    &http.Client{Timeout: 180 * time.Second},
		baseURL:   "http://127.0.0.1:11434/v1",
		healthTTL: 30 * time.Second,
	}
	spec := m.Spec
	if spec == nil {
		spec = map[string]any{}
	}
	if u, ok := spec["baseURL"].(string); ok && u != "" {
		p.baseURL = strings.TrimRight(u, "/")
	}
	if k, ok := spec["apiKey"].(string); ok {
		p.apiKey = k // discouraged in prod — prefer credentials
	}
	p.desc = descriptorFromSpec(m)
	return p, nil
}

func ProviderFactory(m plugin.Manifest) (any, error) {
	return NewProvider(m)
}

func (p *Provider) ID() types.PluginID { return p.manifest.Metadata.ID }

func (p *Provider) Health(ctx context.Context) error {
	if p.healthTTL > 0 && !p.healthAt.IsZero() && time.Since(p.healthAt) < p.healthTTL {
		if p.healthOK {
			return nil
		}
		return p.healthErr
	}
	url := p.modelsURL()
	hctx, cancel := context.WithTimeout(ctx, 800*time.Millisecond)
	defer cancel()
	req, err := http.NewRequestWithContext(hctx, http.MethodGet, url, nil)
	if err != nil {
		p.cacheHealth(false, err)
		return err
	}
	p.auth(req, "")
	resp, err := p.client.Do(req)
	if err != nil {
		err = fmt.Errorf("unreachable: %w", err)
		p.cacheHealth(false, err)
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 500 || resp.StatusCode == 404 {
		err = fmt.Errorf("http %d", resp.StatusCode)
		p.cacheHealth(false, err)
		return err
	}
	p.cacheHealth(true, nil)
	return nil
}

func (p *Provider) cacheHealth(ok bool, err error) {
	p.healthOK = ok
	p.healthErr = err
	p.healthAt = time.Now()
}

func (p *Provider) Describe(ctx context.Context) (provider.Descriptor, error) {
	d := p.desc
	d.BaseURL = p.baseURL
	return d, nil
}

// ListModels auto-discovers models via GET {base}/models (OpenAI-compatible).
func (p *Provider) ListModels(ctx context.Context) ([]provider.ModelInfo, error) {
	url := p.modelsURL()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	p.auth(req, p.apiKey)
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("list models http %d: %s", resp.StatusCode, string(raw))
	}
	var body struct {
		Data []struct {
			ID      string `json:"id"`
			OwnedBy string `json:"owned_by"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		return nil, err
	}
	out := make([]provider.ModelInfo, 0, len(body.Data))
	for _, m := range body.Data {
		if m.ID == "" {
			continue
		}
		out = append(out, provider.ModelInfo{
			ID: m.ID, OwnedBy: m.OwnedBy,
			Capabilities: p.desc.Capabilities, CostTier: p.desc.CostTier,
		})
	}
	if len(out) == 0 {
		return p.desc.Models, nil
	}
	p.desc.Models = out
	return out, nil
}

var _ provider.ModelCatalog = (*Provider)(nil)

type chatReq struct {
	Model      string           `json:"model"`
	Messages   []map[string]any `json:"messages"`
	Tools      []map[string]any `json:"tools,omitempty"`
	ToolChoice any              `json:"tool_choice,omitempty"`
	MaxTokens  int              `json:"max_tokens,omitempty"`
}

type chatResp struct {
	Choices []struct {
		FinishReason string `json:"finish_reason"`
		Message      struct {
			Content   string `json:"content"`
			ToolCalls []struct {
				ID       string `json:"id"`
				Type     string `json:"type"`
				Function struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int64 `json:"prompt_tokens"`
		CompletionTokens int64 `json:"completion_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (p *Provider) Complete(ctx context.Context, req provider.CompletionRequest) (provider.CompletionResponse, error) {
	model := req.Model
	if model == "" && len(p.desc.Models) > 0 {
		model = p.desc.Models[0].ID
	}
	if model == "" {
		model = "default"
	}
	msgs := make([]map[string]any, 0, len(req.Messages))
	for _, m := range req.Messages {
		msg := map[string]any{"role": m.Role}
		if m.Content != "" || len(m.ToolCalls) == 0 {
			msg["content"] = m.Content
		}
		if m.Name != "" {
			msg["name"] = m.Name
		}
		if m.ToolCallID != "" {
			msg["tool_call_id"] = m.ToolCallID
		}
		if len(m.ToolCalls) > 0 {
			tcs := make([]map[string]any, 0, len(m.ToolCalls))
			for _, tc := range m.ToolCalls {
				tcs = append(tcs, map[string]any{
					"id":   tc.ID,
					"type": "function",
					"function": map[string]any{
						"name":      tc.Function.Name,
						"arguments": tc.Function.Arguments,
					},
				})
			}
			msg["tool_calls"] = tcs
			if m.Content == "" {
				msg["content"] = nil
			}
		}
		msgs = append(msgs, msg)
	}
	if len(msgs) == 0 {
		msgs = []map[string]any{{"role": "user", "content": ""}}
	}

	cr := chatReq{Model: model, Messages: msgs}
	if req.MaxTokens > 0 {
		cr.MaxTokens = req.MaxTokens
	}
	if len(req.Tools) > 0 {
		tools := make([]map[string]any, 0, len(req.Tools))
		for _, t := range req.Tools {
			fn := map[string]any{
				"name": t.Function.Name,
			}
			if t.Function.Description != "" {
				fn["description"] = t.Function.Description
			}
			if t.Function.Parameters != nil {
				fn["parameters"] = t.Function.Parameters
			} else {
				fn["parameters"] = map[string]any{"type": "object", "properties": map[string]any{}}
			}
			tools = append(tools, map[string]any{"type": "function", "function": fn})
		}
		cr.Tools = tools
		choice := req.ToolChoice
		if choice == "" {
			choice = "auto"
		}
		if choice != "none" {
			cr.ToolChoice = choice
		}
	}

	body, _ := json.Marshal(cr)
	url := p.chatURL()
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return provider.CompletionResponse{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	key := p.apiKey
	if req.CredentialHandle != "" && p.Resolve != nil {
		if k, err := p.Resolve(ctx, req.CredentialHandle); err == nil && k != "" {
			key = k
		}
	}
	p.auth(httpReq, key)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return provider.CompletionResponse{}, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var out chatResp
	_ = json.Unmarshal(raw, &out)
	if resp.StatusCode >= 300 {
		msg := string(raw)
		if out.Error != nil && out.Error.Message != "" {
			msg = out.Error.Message
		}
		return provider.CompletionResponse{}, fmt.Errorf("openai-compat http %d: %s", resp.StatusCode, msg)
	}
	content := ""
	var toolCalls []provider.ToolCall
	finish := ""
	if len(out.Choices) > 0 {
		content = out.Choices[0].Message.Content
		finish = out.Choices[0].FinishReason
		for _, tc := range out.Choices[0].Message.ToolCalls {
			call := provider.ToolCall{ID: tc.ID, Type: tc.Type}
			call.Function.Name = tc.Function.Name
			call.Function.Arguments = tc.Function.Arguments
			toolCalls = append(toolCalls, call)
		}
	}
	return provider.CompletionResponse{
		ProviderID:   p.ID(),
		ModelID:      model,
		Content:      content,
		ToolCalls:    toolCalls,
		FinishReason: finish,
		TokensIn:     out.Usage.PromptTokens,
		TokensOut:    out.Usage.CompletionTokens,
		CostUSD:      0,
	}, nil
}

func (p *Provider) auth(req *http.Request, key string) {
	if key == "" {
		key = p.apiKey
	}
	if key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
}

func (p *Provider) chatURL() string {
	base := p.baseURL
	if strings.HasSuffix(base, "/v1") {
		return base + "/chat/completions"
	}
	if strings.Contains(base, "/v1/") {
		return strings.TrimRight(base, "/") + "/chat/completions"
	}
	return base + "/v1/chat/completions"
}

func (p *Provider) modelsURL() string {
	base := p.baseURL
	if strings.HasSuffix(base, "/v1") {
		return base + "/models"
	}
	return base + "/v1/models"
}

func descriptorFromSpec(m plugin.Manifest) provider.Descriptor {
	d := provider.Descriptor{
		ID: m.Metadata.ID, Local: true, CostTier: types.TierFreeLocal,
		Capabilities: []types.Capability{"coding", "tools"},
	}
	spec := m.Spec
	if spec == nil {
		return d
	}
	if v, ok := spec["local"].(bool); ok {
		d.Local = v
	}
	if v, ok := spec["costTier"].(string); ok {
		d.CostTier = types.CostTier(v)
	}
	if caps, ok := spec["capabilities"].([]any); ok {
		d.Capabilities = nil
		for _, c := range caps {
			if s, ok := c.(string); ok {
				d.Capabilities = append(d.Capabilities, types.Capability(s))
			}
		}
	}
	if models, ok := spec["models"].([]any); ok {
		for _, raw := range models {
			mm, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			id, _ := mm["id"].(string)
			if id == "" {
				continue
			}
			d.Models = append(d.Models, provider.ModelInfo{
				ID: id, Capabilities: d.Capabilities, CostTier: d.CostTier,
			})
		}
	}
	if len(d.Models) == 0 {
		d.Models = []provider.ModelInfo{{ID: "default", Capabilities: d.Capabilities, CostTier: d.CostTier}}
	}
	return d
}

var _ provider.Provider = (*Provider)(nil)
