package plugin

import (
	"fmt"
	"sync"

	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/types"
)

// MemoryRegistry is an in-process Registry implementation.
type MemoryRegistry struct {
	mu   sync.RWMutex
	byID map[types.PluginID]entry
}

type entry struct {
	m Manifest
	v any
}

func NewMemoryRegistry() *MemoryRegistry {
	return &MemoryRegistry{byID: map[types.PluginID]entry{}}
}

func (r *MemoryRegistry) Register(m Manifest, instance any) error {
	if m.Metadata.ID == "" {
		return fmt.Errorf("plugin id required")
	}
	if m.Kind == "" {
		return fmt.Errorf("plugin kind required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.byID[m.Metadata.ID] = entry{m: m, v: instance}
	return nil
}

func (r *MemoryRegistry) Get(id types.PluginID) (Manifest, any, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	e, ok := r.byID[id]
	return e.m, e.v, ok
}

func (r *MemoryRegistry) List(kind Kind) []Manifest {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []Manifest
	for _, e := range r.byID {
		if kind == "" || e.m.Kind == kind {
			out = append(out, e.m)
		}
	}
	return out
}
