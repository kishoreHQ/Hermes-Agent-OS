package remediation

import "testing"

func TestRunAndFreeze(t *testing.T) {
	e := New()
	run, err := e.Run("pb.provider.restart", false)
	if err != nil || run.Status != "ok" {
		t.Fatal(err, run)
	}
	_, err = e.Run("pb.provider.restart", true)
	if err == nil {
		t.Fatal("expected freeze deny")
	}
}
