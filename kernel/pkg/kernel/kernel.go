// Package kernel is the Hermes Agent Runtime Kernel.
// Zero vendor names. Zero host product assumptions.
package kernel

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/adapters/echo"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/capability"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/credentials"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/eventbus"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/host"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/memorystore"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/plugin"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/provider"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/router"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/runtime"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/types"
)

// Kernel is the host-neutral core of Hermes Agent OS.
type Kernel struct {
	mu       sync.Mutex
	plugins  plugin.Registry
	bus      eventbus.Bus
	caps     *capability.Engine
	creds    credentials.Broker
	memory   memorystore.Store
	missions map[types.MissionID]*host.Mission

	// Live instances collected from registry (refreshed on demand)
	providers []provider.Provider
	runtimes  []runtime.Runtime
}

// Options configures Kernel construction.
type Options struct {
	Registry plugin.Registry
	Bus      eventbus.Bus
	Creds    credentials.Broker
	Memory   memorystore.Store
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
	creds := opts.Creds
	if creds == nil {
		creds = credentials.NewMemoryBroker()
	}
	mem := opts.Memory
	if mem == nil {
		mem = memorystore.New()
	}
	k := &Kernel{
		plugins:  reg,
		bus:      bus,
		caps:     capability.New(),
		creds:    creds,
		memory:   mem,
		missions: map[types.MissionID]*host.Mission{},
	}
	k.refreshAdapters()
	return k
}

func (k *Kernel) Plugins() plugin.Registry      { return k.plugins }
func (k *Kernel) Bus() eventbus.Bus             { return k.bus }
func (k *Kernel) Creds() credentials.Broker     { return k.creds }
func (k *Kernel) Memory() memorystore.Store     { return k.memory }

// RefreshAdapters re-reads provider/runtime instances from the plugin registry.
func (k *Kernel) RefreshAdapters() {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.refreshAdaptersLocked()
}

func (k *Kernel) refreshAdapters() {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.refreshAdaptersLocked()
}

func (k *Kernel) refreshAdaptersLocked() {
	k.providers = nil
	k.runtimes = nil
	for _, m := range k.plugins.List(plugin.KindProvider) {
		_, inst, ok := k.plugins.Get(m.Metadata.ID)
		if !ok {
			continue
		}
		if p, ok := inst.(provider.Provider); ok {
			k.providers = append(k.providers, p)
		}
	}
	for _, m := range k.plugins.List(plugin.KindRuntime) {
		_, inst, ok := k.plugins.Get(m.Metadata.ID)
		if !ok {
			continue
		}
		if rt, ok := inst.(runtime.Runtime); ok {
			// Wire Completer for echo runtimes so they use the routed provider
			if er, ok := rt.(*echo.Runtime); ok {
				er.Complete = k.completeViaRoutedProvider
			}
			k.runtimes = append(k.runtimes, rt)
		}
	}
}

// completeViaRoutedProvider looks up provider by correlation or first healthy.
func (k *Kernel) completeViaRoutedProvider(ctx context.Context, req provider.CompletionRequest) (provider.CompletionResponse, error) {
	k.mu.Lock()
	providers := append([]provider.Provider{}, k.providers...)
	k.mu.Unlock()

	var pid types.PluginID
	if req.Correlation != nil {
		pid = types.PluginID(req.Correlation["providerId"])
	}
	for _, p := range providers {
		if pid != "" && p.ID() != pid {
			continue
		}
		if err := p.Health(ctx); err != nil {
			continue
		}
		return p.Complete(ctx, req)
	}
	return provider.CompletionResponse{}, fmt.Errorf("no provider available for completion")
}

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
	// Ensure adapters reflect latest registry
	k.refreshAdaptersLocked()
	cp := m
	k.missions[m.ID] = &cp
	providers := append([]provider.Provider{}, k.providers...)
	runtimes := append([]runtime.Runtime{}, k.runtimes...)
	k.mu.Unlock()

	_ = k.bus.Publish(ctx, eventbus.Event{
		Type: "mission.created", MissionID: m.ID,
		Data: map[string]any{
			"goal": m.Goal, "name": m.Name, "state": string(m.State),
			"requiredCapabilities": capsStrings(req),
		},
	})

	// Execute synchronously for H2 (async worker pool later)
	if err := k.executeMission(ctx, m.ID, req, providers, runtimes); err != nil {
		k.setMissionFailed(ctx, m.ID, err)
		// Still return id — host can inspect failed state
		return m.ID, nil
	}
	return m.ID, nil
}

func (k *Kernel) executeMission(
	ctx context.Context,
	id types.MissionID,
	req []types.Capability,
	providers []provider.Provider,
	runtimes []runtime.Runtime,
) error {
	r := router.New(providers, runtimes)
	decision, err := r.Route(ctx, req, true)
	if err != nil {
		return err
	}

	_ = k.bus.Publish(ctx, eventbus.Event{
		Type: "route.decided", MissionID: id,
		Data: map[string]any{
			"providerId": string(decision.ProviderID),
			"runtimeId":  string(decision.RuntimeID),
			"modelId":    decision.ModelID,
			"costTier":   string(decision.CostTier),
			"reason":     decision.Reason,
			"required":   capsStrings(decision.Required),
		},
	})

	// Credential handle only (INV-07) — demo platform token for echo path
	handle, err := k.creds.Put(ctx, string(decision.ProviderID), "mission", decision.ProviderID, "hermes-demo-token")
	if err != nil {
		return err
	}
	_ = k.bus.Publish(ctx, eventbus.Event{
		Type: "credential.issued", MissionID: id,
		Data: map[string]any{"handle": string(handle), "scope": string(decision.ProviderID)},
	})

	// Shared memory read (INV-06)
	memHits, _ := k.memory.Search(ctx, memorystore.Query{MissionID: id, Limit: 20})
	// Also include recent platform semantic memory
	global, _ := k.memory.Search(ctx, memorystore.Query{Kind: memorystore.KindSemantic, Limit: 5})
	memHits = append(memHits, global...)

	var rt runtime.Runtime
	for _, rtx := range runtimes {
		if rtx.ID() == decision.RuntimeID {
			rt = rtx
			break
		}
	}
	if rt == nil {
		return fmt.Errorf("runtime %s not found", decision.RuntimeID)
	}

	// Re-wire completer on echo runtime (instance may be shared)
	if er, ok := rt.(*echo.Runtime); ok {
		er.Complete = k.completeViaRoutedProvider
	}

	k.mu.Lock()
	mission := k.missions[id]
	goal := ""
	if mission != nil {
		goal = mission.Goal
		mission.ProviderID = decision.ProviderID
		mission.RuntimeID = decision.RuntimeID
		mission.ModelID = decision.ModelID
		mission.RouteReason = decision.Reason
	}
	k.mu.Unlock()

	env := runtime.ContextEnvelope{
		Mission: map[string]any{
			"id": string(id), "goal": goal,
			"requiredCapabilities": capsStrings(req),
		},
		Workspace: map[string]any{"profile": "host-neutral"},
		Memory:    memorystore.AsMaps(memHits),
		Credentials: []map[string]any{
			{"handle": string(handle), "scope": string(decision.ProviderID)},
		},
		Tools:  []map[string]any{{"id": "echo", "name": "echo"}},
		Budget: map[string]any{"maxSteps": 10},
		Security: map[string]any{
			"sandbox": "process-pty",
		},
		Prompt: goal,
		Correlation: map[string]string{
			"missionId":  string(id),
			"providerId": string(decision.ProviderID),
			"runtimeId":  string(decision.RuntimeID),
			"modelId":    decision.ModelID,
		},
	}

	_ = k.bus.Publish(ctx, eventbus.Event{
		Type: "runtime.started", MissionID: id,
		Data: map[string]any{"runtimeId": string(decision.RuntimeID)},
	})

	result, err := rt.Execute(ctx, env)
	if err != nil {
		return err
	}

	// Write episodic memory (shared, trust-labeled)
	_, _ = k.memory.Write(ctx, memorystore.Entry{
		Kind:      memorystore.KindEpisodic,
		MissionID: id,
		Content:   fmt.Sprintf("mission %s: %s", id, result.Output),
		Trust:     types.TrustAgent,
		Provenance: map[string]string{
			"providerId": string(decision.ProviderID),
			"runtimeId":  string(decision.RuntimeID),
			"modelId":    decision.ModelID,
		},
	})

	_ = k.bus.Publish(ctx, eventbus.Event{
		Type: "runtime.completed", MissionID: id,
		Data: map[string]any{
			"status": result.Status, "steps": result.StepsUsed,
			"tokens": result.TokensUsed, "costUsd": result.CostUSD,
		},
	})
	_ = k.bus.Publish(ctx, eventbus.Event{
		Type: "memory.written", MissionID: id,
		Data: map[string]any{"kind": "episodic", "trust": string(types.TrustAgent)},
	})

	state := host.StateSucceeded
	if result.Status == "failed" {
		state = host.StateFailed
	}

	k.mu.Lock()
	if mission := k.missions[id]; mission != nil {
		mission.State = state
		mission.Output = result.Output
		mission.CostUSD = result.CostUSD
		mission.UpdatedAt = time.Now().UTC()
	}
	k.mu.Unlock()

	_ = k.bus.Publish(ctx, eventbus.Event{
		Type: "mission.updated", MissionID: id,
		Data: map[string]any{"state": string(state), "output": result.Output},
	})
	return nil
}

func (k *Kernel) setMissionFailed(ctx context.Context, id types.MissionID, err error) {
	k.mu.Lock()
	if m := k.missions[id]; m != nil {
		m.State = host.StateFailed
		m.Output = err.Error()
		m.UpdatedAt = time.Now().UTC()
	}
	k.mu.Unlock()
	_ = k.bus.Publish(ctx, eventbus.Event{
		Type: "mission.updated", MissionID: id,
		Data: map[string]any{"state": string(host.StateFailed), "error": err.Error()},
	})
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

func (k *Kernel) Health(ctx context.Context) error {
	k.mu.Lock()
	np, nr := len(k.providers), len(k.runtimes)
	k.mu.Unlock()
	if np == 0 || nr == 0 {
		// Still healthy process; degraded fleet
		return nil
	}
	return nil
}

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
