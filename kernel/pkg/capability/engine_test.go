package capability

import (
	"testing"

	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/types"
)

func TestNormalize_RejectsModelNames(t *testing.T) {
	e := New()
	out := e.Normalize([]types.Capability{"coding", "gpt-4", "claude", "tools", "claude-3-5-sonnet-20241022"})
	if len(out) != 2 {
		t.Fatalf("got %v", out)
	}
}

func TestIsModelNameAntiPattern(t *testing.T) {
	cases := []struct {
		s    string
		want bool
	}{
		{"coding", false},
		{"tools", false},
		{"gpt-4", true},
		{"claude-3-opus", true},
		{"gemini-pro", true},
	}
	for _, tc := range cases {
		if got := IsModelNameAntiPattern(tc.s); got != tc.want {
			t.Errorf("%q: got %v want %v", tc.s, got, tc.want)
		}
	}
}

func TestCompatible(t *testing.T) {
	if !Compatible([]types.Capability{"a", "b"}, []types.Capability{"a"}) {
		t.Fatal("expected compatible")
	}
	if Compatible([]types.Capability{"a"}, []types.Capability{"a", "b"}) {
		t.Fatal("expected incompatible")
	}
}
