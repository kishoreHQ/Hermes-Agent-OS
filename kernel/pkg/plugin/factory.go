package plugin

import (
	"fmt"
	"sync"
)

// Factory constructs a live plugin instance from a loaded Manifest.
// Factories are registered by driver name (manifest label hermes.driver or metadata id).
type Factory func(m Manifest) (any, error)

// FactoryRegistry maps driver keys to constructors.
// Kernel never hardcodes vendors — only registers known in-tree drivers.
type FactoryRegistry struct {
	mu   sync.RWMutex
	byKey map[string]Factory
}

func NewFactoryRegistry() *FactoryRegistry {
	return &FactoryRegistry{byKey: map[string]Factory{}}
}

// Register binds a driver key (e.g. "echo-provider", "echo-runtime", "memory-ephemeral").
func (f *FactoryRegistry) Register(driver string, fn Factory) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.byKey[driver] = fn
}

// Create builds an instance for manifest using hermes.driver label, else metadata.id, else kind default.
func (f *FactoryRegistry) Create(m Manifest) (any, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	keys := []string{}
	if m.Labels != nil {
		if d := m.Labels["hermes.driver"]; d != "" {
			keys = append(keys, d)
		}
	}
	keys = append(keys, string(m.Metadata.ID), string(m.Kind))
	for _, k := range keys {
		if fn, ok := f.byKey[k]; ok {
			return fn(m)
		}
	}
	return nil, fmt.Errorf("no factory for plugin %s (kind=%s); set labels.hermes.driver", m.Metadata.ID, m.Kind)
}

// Has reports whether a driver key is registered.
func (f *FactoryRegistry) Has(driver string) bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	_, ok := f.byKey[driver]
	return ok
}
