package artifact

import (
	"context"
	"testing"
)

func TestPutGetIdempotent(t *testing.T) {
	s := New()
	ctx := context.Background()
	m1, err := s.Put(ctx, []byte("hello"), "text/plain", "m1", nil)
	if err != nil {
		t.Fatal(err)
	}
	m2, err := s.Put(ctx, []byte("hello"), "text/plain", "m2", nil)
	if err != nil || m1.Digest != m2.Digest {
		t.Fatalf("%v %v", m1, m2)
	}
	data, meta, err := s.Get(ctx, m1.Digest)
	if err != nil || string(data) != "hello" || meta.Size != 5 {
		t.Fatal(err, data, meta)
	}
}
