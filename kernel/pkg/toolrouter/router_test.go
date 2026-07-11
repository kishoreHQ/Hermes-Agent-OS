package toolrouter

import (
	"context"
	"testing"
)

func TestInvokeEcho(t *testing.T) {
	r := New()
	inv, err := r.Invoke(context.Background(), "echo", "m1", "runtime.example.echo", map[string]any{"text": "hi"})
	if err != nil || inv.Status != "ok" || inv.Output != "hi" {
		t.Fatalf("%+v %v", inv, err)
	}
	if len(r.Invocations(10)) != 1 {
		t.Fatal("log")
	}
	if len(r.List()) < 3 {
		t.Fatal("builtins")
	}
}

func TestDenyUnknown(t *testing.T) {
	r := New()
	inv, err := r.Invoke(context.Background(), "nope", "", "", nil)
	if err == nil || inv.Status != "denied" {
		t.Fatalf("%+v", inv)
	}
}
