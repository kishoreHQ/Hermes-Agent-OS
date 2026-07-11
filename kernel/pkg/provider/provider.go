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

type Descriptor struct {
	ID           types.PluginID     `json:"id"`
	Capabilities []types.Capability `json:"capabilities"`
	Models       []ModelInfo        `json:"models"`
	Local        bool               `json:"local"`
	CostTier     types.CostTier     `json:"costTier"`
}

type ModelInfo struct {
	ID           string             `json:"id"`
	Capabilities []types.Capability `json:"capabilities"`
	CostTier     types.CostTier     `json:"costTier"`
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
