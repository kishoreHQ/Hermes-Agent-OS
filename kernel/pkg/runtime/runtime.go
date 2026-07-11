// Package runtime defines the Runtime Plugin contract (INV-01, INV-09).
// A runtime executes work. It is not a model provider.
package runtime

import (
	"context"

	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/types"
)

// Runtime is an agent harness plugin.
type Runtime interface {
	ID() types.PluginID
	Describe(ctx context.Context) (Descriptor, error)
	Execute(ctx context.Context, env ContextEnvelope) (Result, error)
	Health(ctx context.Context) error
}

type Descriptor struct {
	ID              types.PluginID     `json:"id"`
	Version         string             `json:"version"`
	CapabilitiesIn  []types.Capability `json:"capabilitiesIn"`
	CapabilitiesOut []types.Capability `json:"capabilitiesOut"`
	SandboxTier     string             `json:"sandboxTier"` // micro-vm|container|process-pty
}

// ContextEnvelope is the unified context (INV-05). Prompt is one field.
type ContextEnvelope struct {
	Workspace   map[string]any       `json:"workspace"`
	Mission     map[string]any       `json:"mission"`
	Memory      []map[string]any     `json:"memory"`
	Knowledge   []map[string]any     `json:"knowledge"`
	Artifacts   []types.ArtifactDigest `json:"artifacts"`
	Policies    []map[string]any     `json:"policies"`
	Credentials []map[string]any     `json:"credentials"` // handles only
	Tools       []map[string]any     `json:"tools"`
	Budget      map[string]any       `json:"budget"`
	Security    map[string]any       `json:"security"`
	Preferences map[string]any       `json:"preferences,omitempty"`
	Prompt      string               `json:"prompt"`
	Correlation map[string]string    `json:"correlation"`
}

type Result struct {
	Status     string                 `json:"status"`
	Output     string                 `json:"output,omitempty"`
	Artifacts  []types.ArtifactDigest `json:"artifacts,omitempty"`
	StepsUsed  int                    `json:"stepsUsed"`
	TokensUsed int64                  `json:"tokensUsed"`
	CostUSD    float64                `json:"costUSD"`
}
