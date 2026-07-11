package planner

import "testing"

func TestCreateVersion(t *testing.T) {
	s := New()
	p, err := s.Create("m1", "ship feature", nil)
	if err != nil || p.Version != 1 || len(p.Steps) != 3 {
		t.Fatalf("%+v %v", p, err)
	}
	p2, err := s.BumpVersion(p.ID, nil)
	if err != nil || p2.Version != 2 {
		t.Fatal(err, p2)
	}
}
