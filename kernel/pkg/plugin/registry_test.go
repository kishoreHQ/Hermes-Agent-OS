package plugin

import (
	"testing"

	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/types"
)

func TestRegisterList(t *testing.T) {
	r := NewMemoryRegistry()
	err := r.Register(Manifest{
		APIVersion: "hermes.plugin/v1",
		Kind:       KindProvider,
		Metadata:   Metadata{ID: "provider.example", Version: "1.0.0"},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	list := r.List(KindProvider)
	if len(list) != 1 {
		t.Fatalf("%d", len(list))
	}
	_, _, ok := r.Get(types.PluginID("provider.example"))
	if !ok {
		t.Fatal("missing")
	}
}
