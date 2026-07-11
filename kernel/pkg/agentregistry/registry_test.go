package agentregistry

import "testing"

func TestRoles(t *testing.T) {
	r := New()
	if len(r.List()) < 2 {
		t.Fatal("seed")
	}
	if !r.HasRole("agent.default", "builder") {
		t.Fatal("role")
	}
	if r.HasRole("agent.default", "admin") {
		t.Fatal("unexpected")
	}
}
