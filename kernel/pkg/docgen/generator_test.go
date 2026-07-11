package docgen

import "testing"

func TestGenerate(t *testing.T) {
	g := New()
	d, err := g.Generate(GenerateReq{Title: "Runbook", Goal: "deploy safely", Kind: "runbook"})
	if err != nil || !stringsContains(d.Body, "deploy safely") {
		t.Fatal(err, d)
	}
}

func stringsContains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		(len(s) > 0 && (func() bool {
			for i := 0; i+len(sub) <= len(s); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		})()))
}
