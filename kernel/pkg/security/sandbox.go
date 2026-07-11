package security

import (
	"fmt"
	"strings"
)

// Sandbox tier ordering: higher = more isolation.
const (
	SandboxProcessPTY = "process-pty"
	SandboxContainer  = "container"
	SandboxMicroVM    = "micro-vm"
)

var sandboxRank = map[string]int{
	SandboxProcessPTY: 1,
	SandboxContainer:  2,
	SandboxMicroVM:    3,
}

// RankSandbox returns isolation rank (0 unknown).
func RankSandbox(tier string) int {
	return sandboxRank[strings.ToLower(strings.TrimSpace(tier))]
}

// EnforceMinSandbox fails if runtime tier is weaker than required minimum.
func EnforceMinSandbox(runtimeTier, minRequired string) error {
	if minRequired == "" {
		return nil
	}
	have := RankSandbox(runtimeTier)
	need := RankSandbox(minRequired)
	if need == 0 {
		return fmt.Errorf("unknown min sandbox tier %q", minRequired)
	}
	if have == 0 {
		return fmt.Errorf("runtime sandbox tier %q unknown", runtimeTier)
	}
	if have < need {
		return fmt.Errorf("sandbox policy: runtime %q weaker than required min %q", runtimeTier, minRequired)
	}
	return nil
}
