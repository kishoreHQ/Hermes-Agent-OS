// Package echo provides vendor-neutral example provider and runtime adapters (H2).
// No external model vendor — deterministic for tests and demos.
package echo

import (
	"context"
	"fmt"
	"strings"

	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/plugin"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/provider"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/types"
)

// Provider is an in-process echo inference adapter.
type Provider struct {
	manifest plugin.Manifest
	desc     provider.Descriptor
}

func NewProvider(m plugin.Manifest) (*Provider, error) {
	desc := descriptorFromSpec(m)
	return &Provider{manifest: m, desc: desc}, nil
}

func ProviderFactory(m plugin.Manifest) (any, error) {
	return NewProvider(m)
}

func (p *Provider) ID() types.PluginID { return p.manifest.Metadata.ID }

func (p *Provider) Health(ctx context.Context) error { return nil }

func (p *Provider) Describe(ctx context.Context) (provider.Descriptor, error) {
	return p.desc, nil
}

// ListModels returns static models (echo has no remote discovery).
func (p *Provider) ListModels(ctx context.Context) ([]provider.ModelInfo, error) {
	return p.desc.Models, nil
}

var _ provider.ModelCatalog = (*Provider)(nil)

func (p *Provider) Complete(ctx context.Context, req provider.CompletionRequest) (provider.CompletionResponse, error) {
	model := req.Model
	if model == "" && len(p.desc.Models) > 0 {
		model = p.desc.Models[0].ID
	}
	var last string
	for _, m := range req.Messages {
		if m.Role == "user" || m.Role == "system" {
			last = m.Content
		}
	}
	content := fmt.Sprintf("[echo:%s] %s", model, strings.TrimSpace(last))
	inTok := int64(len(last) / 4)
	outTok := int64(len(content) / 4)
	if inTok < 1 {
		inTok = 1
	}
	if outTok < 1 {
		outTok = 1
	}
	return provider.CompletionResponse{
		ProviderID: p.ID(),
		ModelID:    model,
		Content:    content,
		TokensIn:   inTok,
		TokensOut:  outTok,
		CostUSD:    0,
	}, nil
}

func descriptorFromSpec(m plugin.Manifest) provider.Descriptor {
	d := provider.Descriptor{
		ID:       m.Metadata.ID,
		Local:    true,
		CostTier: types.TierFreeLocal,
	}
	spec := m.Spec
	if spec == nil {
		spec = map[string]any{}
	}
	if v, ok := spec["local"].(bool); ok {
		d.Local = v
	}
	if v, ok := spec["costTier"].(string); ok {
		d.CostTier = types.CostTier(v)
	}
	d.Capabilities = stringSliceCaps(spec["capabilities"])
	if models, ok := spec["models"].([]any); ok {
		for _, raw := range models {
			mm, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			mi := provider.ModelInfo{
				ID:       strVal(mm["id"]),
				CostTier: d.CostTier,
			}
			if t, ok := mm["costTier"].(string); ok {
				mi.CostTier = types.CostTier(t)
			}
			mi.Capabilities = stringSliceCaps(mm["capabilities"])
			if mi.ID != "" {
				d.Models = append(d.Models, mi)
			}
		}
	}
	if len(d.Models) == 0 {
		d.Models = []provider.ModelInfo{{
			ID: "echo-1", Capabilities: d.Capabilities, CostTier: d.CostTier,
		}}
	}
	if len(d.Capabilities) == 0 {
		d.Capabilities = []types.Capability{"coding", "tools"}
	}
	return d
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

func strVal(v any) string {
	s, _ := v.(string)
	return s
}

var _ provider.Provider = (*Provider)(nil)
