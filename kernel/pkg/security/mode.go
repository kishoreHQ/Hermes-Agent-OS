// Package security enforces agent modes, scopes, and sandbox policy (H5).
package security

import (
	"fmt"
	"strings"

	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/types"
)

// Mode is an alias for clarity in security APIs.
type Mode = types.AgentMode

const (
	ModeFull    = types.ModeFull
	ModeAssist  = types.ModeAssist
	ModeObserve = types.ModeObserve
)

// ParseMode parses host/label input into an AgentMode.
func ParseMode(s string) (types.AgentMode, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "full":
		return ModeFull, nil
	case "assist":
		return ModeAssist, nil
	case "observe":
		return ModeObserve, nil
	default:
		return "", fmt.Errorf("unknown agent mode %q (full|assist|observe)", s)
	}
}

// Decision is the security gate outcome before execution.
type Decision struct {
	Mode            types.AgentMode `json:"mode"`
	AllowExecute    bool            `json:"allowExecute"`
	RequireApproval bool            `json:"requireApproval"`
	Reason          string          `json:"reason"`
	// Scopes granted to the mission (handles only; never secrets).
	Scopes []string `json:"scopes,omitempty"`
}

// EvaluateMode applies Full / Assist / Observe rules.
// externalAction is true when the mission intends side effects beyond pure inference.
func EvaluateMode(mode types.AgentMode, externalAction bool) Decision {
	d := Decision{Mode: mode, Scopes: DefaultScopes(mode)}
	switch mode {
	case ModeObserve:
		// Journal / route only — no runtime execution of tools/side effects.
		d.AllowExecute = false
		d.RequireApproval = false
		d.Reason = "observe: journal routing without execution"
	case ModeAssist:
		if externalAction {
			d.AllowExecute = false
			d.RequireApproval = true
			d.Reason = "assist: external action requires human approval"
		} else {
			d.AllowExecute = true
			d.Reason = "assist: inference-only path allowed"
		}
	default: // Full
		d.AllowExecute = true
		d.Reason = "full: autonomous execution allowed"
	}
	return d
}

// DefaultScopes returns credential/tool scopes for a mode.
func DefaultScopes(mode types.AgentMode) []string {
	switch mode {
	case ModeObserve:
		return []string{"memory:read", "registry:read", "events:read"}
	case ModeAssist:
		return []string{"memory:read", "memory:write", "registry:read", "provider:complete", "events:read"}
	default:
		return []string{"memory:read", "memory:write", "registry:read", "provider:complete", "runtime:execute", "events:read"}
	}
}

// ScopeAllows reports whether a required scope is present.
func ScopeAllows(granted []string, need string) bool {
	for _, g := range granted {
		if g == need || g == "*" {
			return true
		}
	}
	return false
}
