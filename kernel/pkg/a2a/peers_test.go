package a2a

import (
	"context"
	"testing"

	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/host"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/types"
)

type fakeRunner struct{}

func (f *fakeRunner) SubmitMission(ctx context.Context, m host.Mission) (types.MissionID, error) {
	return "mis_a2a", nil
}
func (f *fakeRunner) GetMission(ctx context.Context, id types.MissionID) (host.Mission, error) {
	return host.Mission{ID: id, State: host.StateSucceeded, Output: "peer-ok"}, nil
}

func TestOffer(t *testing.T) {
	r := New()
	if len(r.List()) < 2 {
		t.Fatal("peers")
	}
	task, err := r.OfferTask("peer.local.reviewer", "review PR")
	if err != nil || task.Status != "done" {
		t.Fatal(err, task)
	}
	r.SetRunner(&fakeRunner{})
	task2, err := r.OfferTaskCtx(context.Background(), "peer.local.builder", "build it")
	if err != nil || task2.MissionID == "" || task2.Result != "peer-ok" {
		t.Fatalf("%+v %v", task2, err)
	}
}
