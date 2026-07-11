package memorystore

import (
	"context"
	"testing"

	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/types"
)

func TestWriteSearch(t *testing.T) {
	s := New()
	ctx := context.Background()
	e, err := s.Write(ctx, Entry{
		Kind: KindEpisodic, MissionID: "m1", Content: "learned routing prefers free-local",
		Trust: types.TrustAgent,
	})
	if err != nil || e.ID == "" {
		t.Fatal(err)
	}
	hits, err := s.Search(ctx, Query{MissionID: "m1", Text: "routing"})
	if err != nil || len(hits) != 1 {
		t.Fatalf("%v %d", err, len(hits))
	}
	if hits[0].Trust != types.TrustAgent {
		t.Fatalf("%s", hits[0].Trust)
	}
}
