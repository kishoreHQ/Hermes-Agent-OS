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

	// health cache avoids stalling routing on every mission when remote is slow/unreachable
	healthOK   bool
	healthErr  error
	healthAt   time.Time
	healthTTL  time.Duration
}

func NewProvider(m plugin.Manifest) (*Provider, error) {
	p := &Provider{
		manifest:  m,
		client:    &http.Client{Timeout: 120 * time.Second},
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
	// Cached + short-timeout probe so remote gateways (Kimchi, etc.) don't stall routing.
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
		// Unreachable is unhealthy for remote; still allow Complete to fail clearly
		err = fmt.Errorf("unreachable: %w", err)
		p.cacheHealth(false, err)
		return err
	}
	defer resp.Body.Close()
	// 2xx/3xx = up; 401/403 still mean the endpoint is reachable (key may be missing).
	// 404/5xx = unhealthy for routing.
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
	// Use manifest models for routing — never network-probe here.
	// Live discovery is GET ListModels / Host /providers/models (operator path).
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
	if p.Resolve != nil {
		// try empty handle skip; Resolve only with real handle from Complete path
	}
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
	// cache into descriptor for subsequent Describe
	p.desc.Models = out
	return out, nil
}

var _ provider.ModelCatalog = (*Provider)(nil)

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
