// Package provider defines the Provider Plugin contract (INV-01).
// A provider supplies models. It does not execute agent work.
package provider

import (
	"context"

	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/types"
)

// Provider is a model-inference plugin.
type Provider interface {
	ID() types.PluginID
	Describe(ctx context.Context) (Descriptor, error)
	Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error)
	Health(ctx context.Context) error
}

// ModelCatalog is optional: auto-discover models (e.g. GET /v1/models).
// When implemented, Hermes prefers live discovery over static manifest models.
type ModelCatalog interface {
	ListModels(ctx context.Context) ([]ModelInfo, error)
}

// DiscoverModels returns live models when supported, else descriptor.Models.
func DiscoverModels(ctx context.Context, p Provider) ([]ModelInfo, error) {
	if c, ok := p.(ModelCatalog); ok {
		models, err := c.ListModels(ctx)
		if err == nil && len(models) > 0 {
			return models, nil
		}
		if err != nil {
			// fall through to static
			_ = err
		}
	}
	d, err := p.Describe(ctx)
	if err != nil {
		return nil, err
	}
	return d.Models, nil
}

type Descriptor struct {
	ID           types.PluginID     `json:"id"`
	Capabilities []types.Capability `json:"capabilities"`
	Models       []ModelInfo        `json:"models"`
	Local        bool               `json:"local"`
	CostTier     types.CostTier     `json:"costTier"`
	// BaseURL is optional (e.g. openai-compat) for operator display.
	BaseURL string `json:"baseUrl,omitempty"`
}

type ModelInfo struct {
	ID           string             `json:"id"`
	Capabilities []types.Capability `json:"capabilities"`
	CostTier     types.CostTier     `json:"costTier"`
	// OwnedBy is optional discovery metadata (openai-compat).
	OwnedBy string `json:"ownedBy,omitempty"`
}

type CompletionRequest struct {
	Model            string
	Messages         []Message
	MaxTokens        int
	CredentialHandle string
	Correlation      map[string]string
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type CompletionResponse struct {
	ProviderID types.PluginID `json:"providerId"`
	ModelID    string         `json:"modelId"`
	Content    string         `json:"content"`
	TokensIn   int64          `json:"tokensIn"`
	TokensOut  int64          `json:"tokensOut"`
	CostUSD    float64        `json:"costUSD"`
}
