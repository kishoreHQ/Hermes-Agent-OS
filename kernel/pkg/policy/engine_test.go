package policy

import "testing"

func TestBudgetSteps(t *testing.T) {
	p := Default()
	if err := p.CheckBudget(0.01); err != nil {
		t.Fatal(err)
	}
	if err := p.CheckBudget(99); err == nil {
		t.Fatal("expected budget fail")
	}
	if err := p.CheckSteps(10); err != nil {
		t.Fatal(err)
	}
	if err := p.CheckSteps(999); err == nil {
		t.Fatal("expected steps fail")
	}
}
