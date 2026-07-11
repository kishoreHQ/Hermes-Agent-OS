package a2a

import "testing"

func TestOffer(t *testing.T) {
	r := New()
	if len(r.List()) < 1 {
		t.Fatal("peers")
	}
	task, err := r.OfferTask("peer.local.reviewer", "review PR")
	if err != nil || task.Status != "done" {
		t.Fatal(err, task)
	}
}
