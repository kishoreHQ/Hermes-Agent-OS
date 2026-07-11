// Package openaicompat is a real OpenAI-compatible HTTP provider plugin.
// Works with any OpenAI Chat Completions-compatible base URL (local or remote).
// No vendor hardcoding — baseURL + model + credential handle are config.
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
}

func NewProvider(m plugin.Manifest) (*Provider, error) {
	p := &Provider{
		manifest: m,
		client:   &http.Client{Timeout: 120 * time.Second},
		baseURL:  "http://127.0.0.1:11434/v1",
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
	// Lightweight: GET models if available; ignore failures for offline-first demos
	url := p.modelsURL()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	p.auth(req, "")
	resp, err := p.client.Do(req)
	if err != nil {
		// Unreachable is unhealthy for remote; still allow Complete to fail clearly
		return fmt.Errorf("unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 500 {
		return fmt.Errorf("http %d", resp.StatusCode)
	}
	return nil
}

func (p *Provider) Describe(ctx context.Context) (provider.Descriptor, error) {
	return p.desc, nil
}

type chatReq struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResp struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
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
	msgs := make([]chatMessage, 0, len(req.Messages))
	for _, m := range req.Messages {
		msgs = append(msgs, chatMessage{Role: m.Role, Content: m.Content})
	}
	if len(msgs) == 0 {
		msgs = []chatMessage{{Role: "user", Content: ""}}
	}
	body, _ := json.Marshal(chatReq{Model: model, Messages: msgs})
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
	var cr chatResp
	_ = json.Unmarshal(raw, &cr)
	if resp.StatusCode >= 300 {
		msg := string(raw)
		if cr.Error != nil && cr.Error.Message != "" {
			msg = cr.Error.Message
		}
		return provider.CompletionResponse{}, fmt.Errorf("openai-compat http %d: %s", resp.StatusCode, msg)
	}
	content := ""
	if len(cr.Choices) > 0 {
		content = cr.Choices[0].Message.Content
	}
	return provider.CompletionResponse{
		ProviderID: p.ID(),
		ModelID:    model,
		Content:    content,
		TokensIn:   cr.Usage.PromptTokens,
		TokensOut:  cr.Usage.CompletionTokens,
		CostUSD:    0,
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
