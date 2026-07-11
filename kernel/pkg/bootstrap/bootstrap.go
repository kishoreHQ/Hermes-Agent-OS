// Package bootstrap wires default factories and loads plugins from disk.
package bootstrap

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/adapters/echo"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/adapters/openaicompat"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/adapters/steps"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/credentials"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/kernel"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/memorystore"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/plugin"
)

func memoryFactory(m plugin.Manifest) (any, error) {
	// Kernel owns the Store; this plugin is a discoverable capability marker.
	return map[string]any{"backend": "kernel-memorystore", "id": string(m.Metadata.ID)}, nil
}

// DefaultFactories registers in-tree drivers (vendor-neutral examples only).
func DefaultFactories() *plugin.FactoryRegistry {
	f := plugin.NewFactoryRegistry()
	f.Register("echo-provider", echo.ProviderFactory)
	f.Register("echo-runtime", echo.RuntimeFactory)
	f.Register("steps-runtime", steps.RuntimeFactory)
	f.Register("openai-compat", openaicompat.ProviderFactory)
	f.Register("memory-ephemeral", memoryFactory)
	// Id aliases
	f.Register("provider.example.echo", echo.ProviderFactory)
	f.Register("runtime.example.echo", echo.RuntimeFactory)
	f.Register("provider.example.budget", echo.ProviderFactory)
	f.Register("runtime.example.steps", steps.RuntimeFactory)
	f.Register("provider.openai.compat", openaicompat.ProviderFactory)
	f.Register("provider.kimchi", openaicompat.ProviderFactory)
	f.Register("memory.example.ephemeral", memoryFactory)
	return f
}

// Result of bootstrap.
type Result struct {
	Kernel       *kernel.Kernel
	Registry     plugin.Registry
	Loaded       int
	LoadWarnings string
}

// Options for New.
type Options struct {
	PluginRoots []string
	// SeedBuiltins registers example plugins in-memory when disk load finds none.
	SeedBuiltins bool
}

// New builds a kernel with plugins loaded from disk (or builtins).
func New(opts Options) (*Result, error) {
	reg := plugin.NewMemoryRegistry()
	factories := DefaultFactories()
	loader := plugin.NewLoader(factories)

	roots := opts.PluginRoots
	if len(roots) == 0 {
		roots = plugin.FindPluginRoots()
	}

	loaded := 0
	var warn string
	for _, root := range roots {
		n, _, err := loader.LoadTree(root, reg)
		loaded += n
		if err != nil {
			warn = err.Error()
		}
	}

	if loaded == 0 && opts.SeedBuiltins {
		if err := seedBuiltins(reg, factories); err != nil {
			return nil, err
		}
		loaded = len(reg.List(""))
	}

	creds := credentials.NewMemoryBroker()
	// Optional live provider keys from environment (never logged)
	if key := os.Getenv("HERMES_OPENAI_API_KEY"); key != "" {
		_, _ = creds.Put(context.Background(), "provider.openai.compat", "env", "provider.openai.compat", key)
	}
	// Kimchi Inference — https://docs.kimchi.dev/docs/cursor · key from app.kimchi.dev/settings
	if key := firstEnv("KIMCHI_API_KEY", "HERMES_KIMCHI_API_KEY"); key != "" {
		_, _ = creds.Put(context.Background(), "provider.kimchi", "env", "provider.kimchi", key)
	}

	k := kernel.NewWithOptions(kernel.Options{
		Registry: reg,
		Creds:    creds,
		Memory:   memorystore.New(),
	})

	// Rebind openai-compat baseURL from env if present
	if base := os.Getenv("HERMES_OPENAI_BASE_URL"); base != "" {
		if _, inst, ok := reg.Get("provider.openai.compat"); ok {
			if p, ok := inst.(*openaicompat.Provider); ok {
				// Recreate with updated base — simpler: re-register
				_ = p
			}
		}
		// Prefer re-registering plugin with env baseURL
		m := plugin.Manifest{
			APIVersion: "hermes.plugin/v1",
			Kind:       plugin.KindProvider,
			Metadata:   plugin.Metadata{ID: "provider.openai.compat", Version: "0.1.0", Name: "OpenAI-Compatible HTTP"},
			Spec: map[string]any{
				"baseURL": base, "local": false, "costTier": "standard",
				"capabilities": []any{"coding", "tools"},
				"models":       []any{map[string]any{"id": envOr("HERMES_OPENAI_MODEL", "default")}},
			},
			Labels: map[string]string{"hermes.driver": "openai-compat"},
		}
		if inst, err := factories.Create(m); err == nil {
			_ = reg.Register(m, inst)
			k.RefreshAdapters()
		}
	}

	// Ensure Kimchi provider is registered (disk plugin or env-forced) with correct base URL.
	// https://llm.kimchi.dev/openai/v1 — same endpoint Cursor/OpenCode use.
	ensureKimchiProvider(reg, factories, k)

	return &Result{
		Kernel:       k,
		Registry:     reg,
		Loaded:       loaded,
		LoadWarnings: warn,
	}, nil
}

// ensureKimchiProvider registers provider.kimchi when an API key is present, or
// refreshes base URL/models if the disk plugin (plugins/providers/kimchi) already loaded.
func ensureKimchiProvider(reg plugin.Registry, factories *plugin.FactoryRegistry, k *kernel.Kernel) {
	base := envOr("HERMES_KIMCHI_BASE_URL", "https://llm.kimchi.dev/openai/v1")
	model := envOr("HERMES_KIMCHI_MODEL", "kimi-k2.6")
	_, _, exists := reg.Get("provider.kimchi")
	keyPresent := firstEnv("KIMCHI_API_KEY", "HERMES_KIMCHI_API_KEY") != ""
	// Only force-register from env when key is set; disk plugin may already exist.
	if !exists && !keyPresent {
		return
	}
	if exists && !keyPresent {
		// Disk plugin already registered — leave as-is (no re-register churn).
		return
	}
	m := plugin.Manifest{
		APIVersion: "hermes.plugin/v1",
		Kind:       plugin.KindProvider,
		Metadata:   plugin.Metadata{ID: "provider.kimchi", Version: "0.1.0", Name: "Kimchi Inference"},
		Spec: map[string]any{
			"baseURL": base, "local": false, "costTier": "budget",
			"capabilities": []any{"coding", "tools"},
			"models": []any{
				map[string]any{"id": model, "capabilities": []any{"coding", "tools"}, "costTier": "budget"},
				map[string]any{"id": "minimax-m3", "capabilities": []any{"coding", "tools"}, "costTier": "budget"},
				map[string]any{"id": "nemotron-3-ultra-fp4", "capabilities": []any{"coding", "tools"}, "costTier": "budget"},
			},
		},
		Labels: map[string]string{
			"hermes.driver": "openai-compat",
			"hermes.vendor": "kimchi",
			"hermes.docs":   "https://docs.kimchi.dev/docs/cursor",
		},
	}
	if inst, err := factories.Create(m); err == nil {
		_ = reg.Register(m, inst)
		k.RefreshAdapters()
	}
}

func firstEnv(keys ...string) string {
	for _, k := range keys {
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			return v
		}
	}
	return ""
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func seedBuiltins(reg plugin.Registry, f *plugin.FactoryRegistry) error {
	manifests := []plugin.Manifest{
		{
			APIVersion: "hermes.plugin/v1",
			Kind:       plugin.KindProvider,
			Metadata:   plugin.Metadata{ID: "provider.example.echo", Version: "0.0.1", Name: "Example Echo Provider"},
			Spec: map[string]any{
				"capabilities": []any{"coding", "tools"},
				"local":        true,
				"costTier":     "free-local",
				"models": []any{
					map[string]any{"id": "echo-1", "capabilities": []any{"coding", "tools"}, "costTier": "free-local"},
				},
			},
			Labels: map[string]string{"hermes.driver": "echo-provider", "hermes.example": "true"},
		},
		{
			APIVersion: "hermes.plugin/v1",
			Kind:       plugin.KindProvider,
			Metadata:   plugin.Metadata{ID: "provider.example.budget", Version: "0.0.1", Name: "Example Budget Provider"},
			Spec: map[string]any{
				"capabilities": []any{"coding", "tools"},
				"local":        false,
				"costTier":     "budget",
				"models": []any{
					map[string]any{"id": "budget-1", "capabilities": []any{"coding", "tools"}, "costTier": "budget"},
				},
			},
			Labels: map[string]string{"hermes.driver": "echo-provider", "hermes.example": "true"},
		},
		{
			APIVersion: "hermes.plugin/v1",
			Kind:       plugin.KindRuntime,
			Metadata:   plugin.Metadata{ID: "runtime.example.echo", Version: "0.0.1", Name: "Example Echo Runtime"},
			Spec: map[string]any{
				"sandboxTier":     "process-pty",
				"capabilitiesIn":  []any{"coding", "tools"},
				"capabilitiesOut": []any{"artifacts"},
			},
			Labels: map[string]string{"hermes.driver": "echo-runtime", "hermes.example": "true"},
		},
		{
			APIVersion: "hermes.plugin/v1",
			Kind:       plugin.KindRuntime,
			Metadata:   plugin.Metadata{ID: "runtime.example.steps", Version: "0.0.1", Name: "Example Steps Runtime"},
			Spec: map[string]any{
				"sandboxTier":     "container",
				"capabilitiesIn":  []any{"coding", "tools"},
				"capabilitiesOut": []any{"artifacts", "plan"},
			},
			Labels: map[string]string{"hermes.driver": "steps-runtime", "hermes.example": "true"},
		},
		{
			APIVersion: "hermes.plugin/v1",
			Kind:       plugin.KindMemory,
			Metadata:   plugin.Metadata{ID: "memory.example.ephemeral", Version: "0.0.1", Name: "Ephemeral Memory"},
			Spec:       map[string]any{"backend": "memory"},
			Labels:     map[string]string{"hermes.driver": "memory-ephemeral", "hermes.example": "true"},
		},
	}
	for _, m := range manifests {
		inst, err := f.Create(m)
		if err != nil {
			return fmt.Errorf("seed %s: %w", m.Metadata.ID, err)
		}
		if err := reg.Register(m, inst); err != nil {
			return err
		}
	}
	return nil
}
