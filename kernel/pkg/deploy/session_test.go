package deploy

import "testing"

func TestRollout(t *testing.T) {
	s := New()
	sess, err := s.Create(CreateReq{Artifact: "sha256:abc", Gates: []string{"eval-pass"}})
	if err != nil {
		t.Fatal(err)
	}
	sess, _ = s.Advance(sess.ID, false)
	if sess.Status != "running" {
		t.Fatal(sess.Status)
	}
	sess, _ = s.Advance(sess.ID, false)
	if sess.Status != "gated" {
		t.Fatal(sess.Status)
	}
	sess, err = s.Advance(sess.ID, true)
	if err != nil || sess.Status != "succeeded" {
		t.Fatal(err, sess)
	}
}
