// Package policy holds enforceable platform policies (H5).
package policy

import (
	"fmt"

	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/security"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/types"
)

// Policy is a named constraint set applied at mission submit/execute.
type Policy struct {
	ID              string          `json:"id"`
	DefaultMode     types.AgentMode `json:"defaultMode"`
	MaxSteps        int             `json:"maxSteps"`
	MaxCostUSD      float64         `json:"maxCostUsd"`
	MinSandboxTier  string          `json:"minSandboxTier,omitempty"`
	PreferLocal     bool            `json:"preferLocal"`
	RequireSigned   bool            `json:"requireSignedPlugins,omitempty"`
}

// Default returns a production-minded baseline.
func Default() Policy {
	return Policy{
		ID:             "policy.default",
		DefaultMode:    types.ModeFull,
		MaxSteps:       50,
		MaxCostUSD:     5.0,
		MinSandboxTier: "", // no min by default (dev-friendly)
		PreferLocal:    true,
	}
}

// Strict returns a tighter baseline for production demos.
func Strict() Policy {
	return Policy{
		ID:             "policy.strict",
		DefaultMode:    types.ModeAssist,
		MaxSteps:       20,
		MaxCostUSD:     1.0,
		MinSandboxTier: security.SandboxProcessPTY,
		PreferLocal:    true,
	}
}

// CheckBudget fails if cost exceeds policy.
func (p Policy) CheckBudget(costUSD float64) error {
	if p.MaxCostUSD > 0 && costUSD > p.MaxCostUSD {
		return fmt.Errorf("policy %s: cost %.4f exceeds max %.4f", p.ID, costUSD, p.MaxCostUSD)
	}
	return nil
}

// CheckSteps fails if steps exceed policy.
func (p Policy) CheckSteps(steps int) error {
	if p.MaxSteps > 0 && steps > p.MaxSteps {
		return fmt.Errorf("policy %s: steps %d exceeds max %d", p.ID, steps, p.MaxSteps)
	}
	return nil
}

// CheckSandbox delegates to security.
func (p Policy) CheckSandbox(runtimeTier string) error {
	return security.EnforceMinSandbox(runtimeTier, p.MinSandboxTier)
}
