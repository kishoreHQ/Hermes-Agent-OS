package credentials

import (
	"context"
	"testing"
)

func TestPutResolveListRevoke(t *testing.T) {
	b := NewMemoryBroker()
	ctx := context.Background()
	h, err := b.Put(ctx, "provider.example.echo", "demo", "provider.example.echo", "s3cr3t")
	if err != nil {
		t.Fatal(err)
	}
	secret, rec, err := b.Resolve(ctx, h)
	if err != nil || secret != "s3cr3t" || rec.Label != "demo" {
		t.Fatalf("%v %q %+v", err, secret, rec)
	}
	list, _ := b.List(ctx)
	if len(list) != 1 || list[0].Handle != h {
		t.Fatalf("%+v", list)
	}
	// List must not embed secrets — Record has no Secret field; resolve is separate.
	if err := b.Revoke(ctx, h); err != nil {
		t.Fatal(err)
	}
	if _, _, err := b.Resolve(ctx, h); err == nil {
		t.Fatal("expected revoke")
	}
}
