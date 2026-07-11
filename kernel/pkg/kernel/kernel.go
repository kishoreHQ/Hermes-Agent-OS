// Package kernel is the Hermes Agent Runtime Kernel.
// Zero vendor names. Zero host product assumptions.
package kernel

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/capability"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/eventbus"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/host"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/plugin"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/types"
)

// Kernel is the host-neutral core of Hermes Agent OS.
type Kernel struct {
	mu       sync.Mutex
	plugins  plugin.Registry
	bus      eventbus.Bus
	caps     *capability.Engine
	missions map[types.MissionID]*host.Mission
}

// Options configures Kernel construction.
type Options struct {
	Registry plugin.Registry
	Bus      eventbus.Bus
}

func New(reg plugin.Registry) *Kernel {
	return NewWithOptions(Options{Registry: reg})
}

func NewWithOptions(opts Options) *Kernel {
	reg := opts.Registry
	if reg == nil {
		reg = plugin.NewMemoryRegistry()
	}
	bus := opts.Bus
	if bus == nil {
		bus = eventbus.NewMemoryBus()
	}
	return &Kernel{
		plugins:  reg,
		bus:      bus,
		caps:     capability.New(),
		missions: map[types.MissionID]*host.Mission{},
	}
}

func (k *Kernel) Plugins() plugin.Registry { return k.plugins }
func (k *Kernel) Bus() eventbus.Bus        { return k.bus }

func (k *Kernel) SubmitMission(ctx context.Context, m host.Mission) (types.MissionID, error) {
	req := k.caps.Normalize(m.RequiredCaps)
	if len(req) == 0 {
		return "", fmt.Errorf("requiredCapabilities required (capability routing, never model names)")
	}
	now := time.Now().UTC()
	if m.ID == "" {
		m.ID = types.MissionID("mis_" + now.Format("150405.000000000"))
	}
	if m.Name == "" {
		m.Name = m.Goal
	}
	if m.Goal == "" {
		return "", fmt.Errorf("goal required")
	}
	m.RequiredCaps = req
	m.State = host.StateRunning
	m.CreatedAt = now
	m.UpdatedAt = now

	k.mu.Lock()
	if _, exists := k.missions[m.ID]; exists {
		k.mu.Unlock()
		return "", fmt.Errorf("mission id already exists")
	}
	cp := m
	k.missions[m.ID] = &cp
	k.mu.Unlock()

	_ = k.bus.Publish(ctx, eventbus.Event{
		Type: "mission.created", MissionID: m.ID,
		Data: map[string]any{
			"goal": m.Goal, "name": m.Name, "state": string(m.State),
			"requiredCapabilities": capsStrings(req),
		},
	})
	_ = k.bus.Publish(ctx, eventbus.Event{
		Type: "mission.updated", MissionID: m.ID,
		Data: map[string]any{"state": string(host.StateRunning)},
	})
	return m.ID, nil
}

func (k *Kernel) CancelMission(ctx context.Context, id types.MissionID, reason string) error {
	k.mu.Lock()
	m, ok := k.missions[id]
	if !ok {
		k.mu.Unlock()
		return fmt.Errorf("unknown mission")
	}
	m.State = host.StateCancelled
	m.CancelReason = reason
	m.UpdatedAt = time.Now().UTC()
	k.mu.Unlock()

	_ = k.bus.Publish(ctx, eventbus.Event{
		Type: "mission.updated", MissionID: id,
		Data: map[string]any{"state": string(host.StateCancelled), "reason": reason},
	})
	return nil
}

func (k *Kernel) GetMission(ctx context.Context, id types.MissionID) (host.Mission, error) {
	k.mu.Lock()
	defer k.mu.Unlock()
	m, ok := k.missions[id]
	if !ok {
		return host.Mission{}, fmt.Errorf("unknown mission")
	}
	return *m, nil
}

func (k *Kernel) ListMissions(ctx context.Context, stateFilter string) ([]host.Mission, error) {
	k.mu.Lock()
	defer k.mu.Unlock()
	out := make([]host.Mission, 0, len(k.missions))
	for _, m := range k.missions {
		if stateFilter != "" && string(m.State) != stateFilter {
			continue
		}
		out = append(out, *m)
	}
	return out, nil
}

func (k *Kernel) SubscribeEvents(ctx context.Context, id types.MissionID) (<-chan host.Event, error) {
	filter := string(id)
	raw, err := k.bus.Subscribe(ctx, filter)
	if err != nil {
		return nil, err
	}
	out := make(chan host.Event, 128)
	go func() {
		defer close(out)
		for e := range raw {
			select {
			case <-ctx.Done():
				return
			case out <- toHostEvent(e):
			}
		}
	}()
	return out, nil
}

func (k *Kernel) EventsSince(ctx context.Context, since int64, missionFilter string) ([]host.Event, error) {
	raw, err := k.bus.Since(ctx, since)
	if err != nil {
		return nil, err
	}
	var out []host.Event
	for _, e := range raw {
		if missionFilter != "" && string(e.MissionID) != missionFilter {
			continue
		}
		out = append(out, toHostEvent(e))
	}
	return out, nil
}

func (k *Kernel) Replay(ctx context.Context, id types.MissionID) ([]host.Event, error) {
	raw, err := k.bus.Replay(ctx, id)
	if err != nil {
		return nil, err
	}
	out := make([]host.Event, 0, len(raw))
	for _, e := range raw {
		out = append(out, toHostEvent(e))
	}
	return out, nil
}

func (k *Kernel) Health(ctx context.Context) error { return nil }

func toHostEvent(e eventbus.Event) host.Event {
	return host.Event{
		Seq: e.Seq, Type: e.Type, MissionID: e.MissionID, TS: e.Time, Data: e.Data,
	}
}

func capsStrings(in []types.Capability) []string {
	out := make([]string, len(in))
	for i, c := range in {
		out[i] = string(c)
	}
	return out
}

var _ host.Interface = (*Kernel)(nil)
