// Package capability resolves intents into declarative capabilities (INV-03).
// Never routes by vendor model-name strings.
package capability

import (
	"strings"
	"unicode"

	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/types"
)

type Engine struct{}

func New() *Engine { return &Engine{} }

// knownModelNameAntiPatterns are exact strings that must never be treated as capabilities.
var knownModelNameAntiPatterns = map[string]struct{}{
	"gpt-4": {}, "gpt-4o": {}, "gpt-4-turbo": {}, "gpt-3.5-turbo": {},
	"claude": {}, "claude-3": {}, "claude-3.5": {}, "claude-opus": {}, "claude-sonnet": {},
	"gemini": {}, "gemini-pro": {}, "gemini-flash": {},
	"o1": {}, "o1-mini": {}, "o3": {}, "o3-mini": {},
}

// IsModelNameAntiPattern reports whether s looks like a vendor model id used as a capability.
func IsModelNameAntiPattern(s string) bool {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return false
	}
	if _, ok := knownModelNameAntiPatterns[s]; ok {
		return true
	}
	// Heuristic: versioned model ids (e.g. claude-3-5-sonnet-20241022, gpt-4o-mini)
	if strings.HasPrefix(s, "gpt-") || strings.HasPrefix(s, "claude-") || strings.HasPrefix(s, "gemini-") {
		return true
	}
	// Reject strings that are mostly model-version shaped (contain year-like 20xx suffix)
	if len(s) > 12 && strings.Contains(s, "-20") {
		for _, r := range s[len(s)-4:] {
			if !unicode.IsDigit(r) {
				return false
			}
		}
		return true
	}
	return false
}

func (e *Engine) Normalize(required []types.Capability) []types.Capability {
	seen := map[types.Capability]bool{}
	var out []types.Capability
	for _, c := range required {
		if c == "" || seen[c] {
			continue
		}
		if IsModelNameAntiPattern(string(c)) {
			continue
		}
		seen[c] = true
		out = append(out, c)
	}
	return out
}

func Compatible(have, need []types.Capability) bool {
	set := map[types.Capability]bool{}
	for _, c := range have {
		set[c] = true
	}
	for _, n := range need {
		if !set[n] {
			return false
		}
	}
	return true
}
