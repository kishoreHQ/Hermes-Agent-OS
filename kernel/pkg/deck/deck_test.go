package deck

import (
	"context"
	"testing"

	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/host"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/types"
)

type fakeRunner struct {
	n int
}

func (f *fakeRunner) SubmitMission(ctx context.Context, m host.Mission) (types.MissionID, error) {
	f.n++
	return types.MissionID("mis_fake"), nil
}
func (f *fakeRunner) GetMission(ctx context.Context, id types.MissionID) (host.Mission, error) {
	return host.Mission{ID: id, State: host.StateSucceeded, Output: "ok"}, nil
}

func TestBoardAndSessions(t *testing.T) {
	b := NewBoard()
	if len(b.ListBoards()) != 1 {
		t.Fatal("board")
	}
	task, err := b.CreateTask(CreateTaskRequest{Title: "x", Column: "backlog"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = b.ClaimTask(task.ID, "agent.a")
	if err != nil {
		t.Fatal(err)
	}

	r := &fakeRunner{}
	sess := NewSessions(r)
	s, err := sess.Create(context.Background(), CreateSessionRequest{RuntimeID: "runtime.example.echo"})
	if err != nil {
		t.Fatal(err)
	}
	out, err := sess.Message(context.Background(), s.ID, "hello")
	if err != nil || out.LastMission == "" {
		t.Fatalf("%+v %v", out, err)
	}
	rt := NewRoutines(r)
	list := rt.List()
	if len(list) < 1 {
		t.Fatal("routines")
	}
	fired, err := rt.Fire(context.Background(), list[0].ID)
	if err != nil || fired.LastMission == "" {
		t.Fatalf("%+v %v", fired, err)
	}
}
