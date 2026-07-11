package eventbus

import (
	"context"
	"testing"
	"time"

	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/types"
)

func TestPublishMonotonicSeq(t *testing.T) {
	b := NewMemoryBus()
	ctx := context.Background()
	_ = b.Publish(ctx, Event{Type: "a", MissionID: "m1"})
	_ = b.Publish(ctx, Event{Type: "b", MissionID: "m1"})
	evs, err := b.Since(ctx, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(evs) != 2 || evs[0].Seq != 1 || evs[1].Seq != 2 {
		t.Fatalf("%+v", evs)
	}
}

func TestSinceAndReplay(t *testing.T) {
	b := NewMemoryBus()
	ctx := context.Background()
	_ = b.Publish(ctx, Event{Type: "x", MissionID: "m1"})
	_ = b.Publish(ctx, Event{Type: "y", MissionID: "m2"})
	_ = b.Publish(ctx, Event{Type: "z", MissionID: "m1"})

	since, _ := b.Since(ctx, 1)
	if len(since) != 2 {
		t.Fatalf("since: %d", len(since))
	}
	rep, _ := b.Replay(ctx, types.MissionID("m1"))
	if len(rep) != 2 {
		t.Fatalf("replay: %d", len(rep))
	}
}

func TestSubscribe(t *testing.T) {
	b := NewMemoryBus()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch, err := b.Subscribe(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	_ = b.Publish(context.Background(), Event{Type: "live", MissionID: "m"})
	select {
	case e := <-ch:
		if e.Type != "live" || e.Seq != 1 {
			t.Fatalf("%+v", e)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}
}
