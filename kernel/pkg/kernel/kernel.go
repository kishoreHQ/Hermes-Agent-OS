// Package kernel is the Hermes Agent Runtime Kernel.
// Zero vendor names. Zero host product assumptions.
package kernel

import (
	"context"
	"fmt"
	"sync"

	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/host"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/plugin"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/types"
)

// Kernel is the host-neutral core of Hermes Agent OS.
type Kernel struct {
	mu       sync.Mutex
	plugins  plugin.Registry
	missions map[types.MissionID]*host.Mission
	seq      int64
}

func New(reg plugin.Registry) *Kernel {
	if reg == nil {
		reg = plugin.NewMemoryRegistry()
	}
	return &Kernel{
		plugins:  reg,
		missions: map[types.MissionID]*host.Mission{},
	}
}

func (k *Kernel) Plugins() plugin.Registry { return k.plugins }

func (k *Kernel) SubmitMission(ctx context.Context, m host.Mission) (types.MissionID, error) {
	if m.ID == "" {
		return "", fmt.Errorf("mission id required")
	}
	if len(m.RequiredCaps) == 0 {
		return "", fmt.Errorf("requiredCapabilities required (capability routing, never model names)")
	}
	k.mu.Lock()
	defer k.mu.Unlock()
	cp := m
	k.missions[m.ID] = &cp
	return m.ID, nil
}

func (k *Kernel) CancelMission(ctx context.Context, id types.MissionID, reason string) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	if _, ok := k.missions[id]; !ok {
		return fmt.Errorf("unknown mission")
	}
	return nil
}

func (k *Kernel) SubscribeEvents(ctx context.Context, id types.MissionID) (<-chan host.Event, error) {
	ch := make(chan host.Event)
	close(ch)
	return ch, nil
}

func (k *Kernel) Health(ctx context.Context) error { return nil }

var _ host.Interface = (*Kernel)(nil)
