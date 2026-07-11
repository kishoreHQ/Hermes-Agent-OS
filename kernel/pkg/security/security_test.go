package security

import (
	"testing"

	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/types"
)

func TestEvaluateMode(t *testing.T) {
	d := EvaluateMode(ModeObserve, true)
	if d.AllowExecute || d.RequireApproval {
		t.Fatalf("%+v", d)
	}
	d = EvaluateMode(ModeAssist, true)
	if d.AllowExecute || !d.RequireApproval {
		t.Fatalf("%+v", d)
	}
	d = EvaluateMode(ModeAssist, false)
	if !d.AllowExecute {
		t.Fatalf("%+v", d)
	}
	d = EvaluateMode(ModeFull, true)
	if !d.AllowExecute {
		t.Fatalf("%+v", d)
	}
}

func TestSandbox(t *testing.T) {
	if err := EnforceMinSandbox(SandboxContainer, SandboxProcessPTY); err != nil {
		t.Fatal(err)
	}
	if err := EnforceMinSandbox(SandboxProcessPTY, SandboxContainer); err == nil {
		t.Fatal("expected fail")
	}
}

func TestSignVerify(t *testing.T) {
	key := []byte("test-secret-key")
	sig := SignHMAC("hermes.plugin/v1", "provider", "provider.example.echo", "0.0.1", key)
	if err := VerifyHMAC("hermes.plugin/v1", "provider", "provider.example.echo", "0.0.1", sig, key); err != nil {
		t.Fatal(err)
	}
	if err := VerifyHMAC("hermes.plugin/v1", "provider", "provider.example.echo", "0.0.1", "deadbeef", key); err == nil {
		t.Fatal("expected fail")
	}
	_ = types.ModeFull
}

func TestParseMode(t *testing.T) {
	m, err := ParseMode("ASSIST")
	if err != nil || m != ModeAssist {
		t.Fatal(m, err)
	}
}
