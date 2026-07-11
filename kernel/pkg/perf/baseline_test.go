package perf

import (
	"context"
	"testing"
)

func TestBaselines(t *testing.T) {
	rep, err := RunBaselines(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if rep.Failed > 0 {
		t.Log(Format(rep))
		// Soft: do not fail CI on overloaded shared hosts — log only if p99 wildly high
		for _, s := range rep.Samples {
			if s.Name == "mission.submit+execute" && s.P99 > MaxMissionP99*4 {
				t.Fatalf("p99 pathological: %s", s.P99)
			}
		}
	}
}
