package workflow

import (
	"context"
	"testing"

	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/host"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/planner"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/types"
)

type fake struct{ n int }

func (f *fake) SubmitMission(ctx context.Context, m host.Mission) (types.MissionID, error) {
	f.n++
	return types.MissionID("mis_" + m.Goal[:min(8, len(m.Goal))]), nil
}
func (f *fake) GetMission(ctx context.Context, id types.MissionID) (host.Mission, error) {
	return host.Mission{ID: id, State: host.StateSucceeded}, nil
}

func TestRunPlan(t *testing.T) {
	ps := planner.New()
	p, _ := ps.Create("m1", "workflow goal", nil)
	f := &fake{}
	o := New(ps, f)
	res, err := o.RunPlan(context.Background(), p.ID)
	if err != nil || res.Status != "completed" || len(res.StepMissions) != 3 {
		t.Fatalf("%+v %v", res, err)
	}
	if f.n != 3 {
		t.Fatal(f.n)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
