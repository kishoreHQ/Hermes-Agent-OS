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

// ToolSpec is an OpenAI-compatible function tool offered to the model.
type ToolSpec struct {
	Type     string         `json:"type"` // "function"
	Function ToolFunction   `json:"function"`
}

type ToolFunction struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters,omitempty"`
}

// ToolCall is a model-requested tool invocation (OpenAI shape).
type ToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"` // "function"
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"` // JSON string
	} `json:"function"`
}

type CompletionRequest struct {
	Model            string
	Messages         []Message
	Tools            []ToolSpec
	ToolChoice       string // auto|none|required
	MaxTokens        int
	CredentialHandle string
	Correlation      map[string]string
}

type Message struct {
	Role       string     `json:"role"` // system|user|assistant|tool
	Content    string     `json:"content"`
	Name       string     `json:"name,omitempty"`       // tool name for role=tool
	ToolCallID string     `json:"toolCallId,omitempty"` // for role=tool
	ToolCalls  []ToolCall `json:"toolCalls,omitempty"`  // for role=assistant
}

type CompletionResponse struct {
	ProviderID types.PluginID `json:"providerId"`
	ModelID    string         `json:"modelId"`
	Content    string         `json:"content"`
	ToolCalls  []ToolCall     `json:"toolCalls,omitempty"`
	FinishReason string       `json:"finishReason,omitempty"`
	TokensIn   int64          `json:"tokensIn"`
	TokensOut  int64          `json:"tokensOut"`
	CostUSD    float64        `json:"costUSD"`
}
