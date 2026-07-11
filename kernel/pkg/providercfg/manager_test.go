package providercfg

import (
	"context"
	"testing"

	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/credentials"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/plugin"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/types"
)

func TestCreateDeleteFromTemplate(t *testing.T) {
	reg := plugin.NewMemoryRegistry()
	creds := credentials.NewMemoryBroker()
	var refreshed int
	m := NewManager(reg, creds, func() { refreshed++ })

	cfg, err := m.Create(context.Background(), CreateRequest{
		FromTemplate: "ollama",
		Name:         "My Ollama",
	})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.BaseURL == "" || !cfg.Managed || cfg.Driver != "openai-compat" {
		t.Fatalf("%+v", cfg)
	}
	if _, _, ok := reg.Get(types.PluginID(cfg.ID)); !ok {
		t.Fatal("not registered")
	}
	if refreshed < 1 {
		t.Fatal("refresh not called")
	}

	// cannot delete unmanaged
	m.SyncFromRegistry()
	// seed unmanaged
	_ = reg.Register(plugin.Manifest{
		APIVersion: "hermes.plugin/v1", Kind: plugin.KindProvider,
		Metadata: plugin.Metadata{ID: "provider.example.echo", Version: "1", Name: "Echo"},
	}, nil)
	m.SyncFromRegistry()
	if err := m.Delete("provider.example.echo"); err == nil {
		t.Fatal("should refuse unmanaged")
	}

	if err := m.Delete(types.PluginID(cfg.ID)); err != nil {
		t.Fatal(err)
	}
	if _, _, ok := reg.Get(types.PluginID(cfg.ID)); ok {
		t.Fatal("still registered")
	}
}

func TestTemplatesPopular(t *testing.T) {
	ts := Templates()
	if len(ts) < 10 {
		t.Fatal(len(ts))
	}
	if _, ok := TemplateByID("openai"); !ok {
		t.Fatal("openai template")
	}
	if _, ok := TemplateByID("openrouter"); !ok {
		t.Fatal("openrouter")
	}
	k, ok := TemplateByID("kimchi")
	if !ok {
		t.Fatal("kimchi template")
	}
	if k.BaseURL != "https://llm.kimchi.dev/openai/v1" {
		t.Fatalf("kimchi baseURL %q", k.BaseURL)
	}
	if k.DefaultModel != "kimi-k2.6" {
		t.Fatalf("kimchi model %q", k.DefaultModel)
	}
}
