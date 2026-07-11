// Package kernel is the Hermes Agent Runtime Kernel.
// Zero vendor names. Zero host product assumptions.
package kernel

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/a2a"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/adapters/echo"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/adapters/openaicompat"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/adapters/steps"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/agentregistry"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/artifact"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/capability"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/credentials"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/deck"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/deploy"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/docgen"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/eventbus"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/host"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/knowledge"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/mcpbridge"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/memorystore"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/planner"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/plugin"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/policy"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/provider"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/remediation"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/router"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/runtime"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/security"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/toolrouter"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/types"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/workflow"
)

// Kernel is the host-neutral core of Hermes Agent OS.
type Kernel struct {
	mu       sync.Mutex
	plugins  plugin.Registry
	bus      eventbus.Bus
	caps     *capability.Engine
	creds    credentials.Broker
	memory   memorystore.Store
	policy   policy.Policy
	missions map[types.MissionID]*host.Mission
	tools    *toolrouter.Router

	// Command Deck services (H3.1)
	Connections *deck.ConnectionsService
	Sessions    *deck.SessionsService
	Board       *deck.BoardService
	Routines    *deck.RoutinesService

	// Platform services (gap closures)
	Artifacts *artifact.Store
	Agents    *agentregistry.Registry
	Plans     *planner.Store
	Workflow  *workflow.Orchestrator
	Knowledge *knowledge.Graph
	MCP       *mcpbridge.Bridge
	A2A       *a2a.Registry
	Remediate *remediation.Engine
	Deploy    *deploy.Service
	Docs      *docgen.Generator

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
	Policy   *policy.Policy
	Tools    *toolrouter.Router
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
	pol := policy.Default()
	if opts.Policy != nil {
		pol = *opts.Policy
	}
	tools := opts.Tools
	if tools == nil {
		tools = toolrouter.New()
	}
	k := &Kernel{
		plugins:  reg,
		bus:      bus,
		caps:     capability.New(),
		creds:    creds,
		memory:   mem,
		policy:   pol,
		tools:    tools,
		missions: map[types.MissionID]*host.Mission{},
	}
	k.Connections = deck.NewConnections(reg)
	k.Sessions = deck.NewSessions(k)
	k.Board = deck.NewBoard()
	k.Routines = deck.NewRoutines(k)
	k.Artifacts = artifact.New()
	k.Agents = agentregistry.New()
	k.Plans = planner.New()
	k.Workflow = workflow.New(k.Plans, k)
	k.Knowledge = knowledge.New()
	k.MCP = mcpbridge.New(tools)
	k.A2A = a2a.New()
	k.A2A.SetRunner(k) // multi-agent: peer tasks become real missions
	k.Remediate = remediation.New()
	k.Deploy = deploy.New()
	k.Docs = docgen.New()
	k.refreshAdapters()
	return k
}

func (k *Kernel) Plugins() plugin.Registry  { return k.plugins }
func (k *Kernel) Bus() eventbus.Bus         { return k.bus }
func (k *Kernel) Creds() credentials.Broker { return k.creds }
func (k *Kernel) Memory() memorystore.Store { return k.memory }
func (k *Kernel) Policy() policy.Policy     { return k.policy }
func (k *Kernel) Tools() *toolrouter.Router { return k.tools }

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
			// Wire credential resolve for OpenAI-compatible providers
			if oc, ok := p.(*openaicompat.Provider); ok {
				oc.Resolve = func(ctx context.Context, handle string) (string, error) {
					sec, _, err := k.creds.Resolve(ctx, credentials.Handle(handle))
					return sec, err
				}
			}
			k.providers = append(k.providers, p)
		}
	}
	for _, m := range k.plugins.List(plugin.KindRuntime) {
		_, inst, ok := k.plugins.Get(m.Metadata.ID)
		if !ok {
			continue
		}
		if rt, ok := inst.(runtime.Runtime); ok {
			// Wire Completer so runtimes use the routed provider (never own vendors)
			k.wireCompleter(rt)
			k.runtimes = append(k.runtimes, rt)
		}
	}
}

func (k *Kernel) wireCompleter(rt runtime.Runtime) {
	switch r := rt.(type) {
	case *echo.Runtime:
		r.Complete = k.completeViaRoutedProvider
	case *steps.Runtime:
		r.Complete = k.completeViaRoutedProvider
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
	// Resolve agent mode (field or label security.mode)
	mode := m.Mode
	if mode == "" && m.Labels != nil {
		mode, _ = security.ParseMode(m.Labels["security.mode"])
	}
	if mode == "" {
		mode = k.policy.DefaultMode
		if mode == "" {
			mode = types.ModeFull
		}
	}
	if _, err := security.ParseMode(string(mode)); err != nil {
		return "", err
	}
	m.Mode = mode
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
			"mode": string(mode), "requiredCapabilities": capsStrings(req),
		},
	})

	// Merge structured selection fields into labels for routing
	labels := mergeRouteLabels(m.Labels, m)

	// Execute synchronously (async worker pool later)
	if err := k.executeMission(ctx, m.ID, req, labels, mode, providers, runtimes); err != nil {
		k.setMissionFailed(ctx, m.ID, err)
		// Still return id — host can inspect failed state
		return m.ID, nil
	}
	return m.ID, nil
}

func mergeRouteLabels(labels map[string]string, m host.Mission) map[string]string {
	out := map[string]string{}
	for k, v := range labels {
		out[k] = v
	}
	if m.PreferProvider != "" && out["route.preferProvider"] == "" {
		out["route.preferProvider"] = string(m.PreferProvider)
	}
	if m.RequireProvider != "" && out["route.requireProvider"] == "" {
		out["route.requireProvider"] = string(m.RequireProvider)
	}
	if m.PreferModel != "" && out["route.preferModel"] == "" {
		out["route.preferModel"] = m.PreferModel
	}
	if len(m.Providers) > 0 && out["route.providers"] == "" {
		parts := make([]string, len(m.Providers))
		for i, p := range m.Providers {
			parts[i] = string(p)
		}
		out["route.providers"] = strings.Join(parts, ",")
	}
	if m.Failover != nil {
		if *m.Failover {
			out["route.failover"] = "true"
		} else {
			out["route.failover"] = "false"
		}
	}
	return out
}

func routeOptionsFromLabels(labels map[string]string) router.Options {
	opts := router.Options{PreferLocal: true, PolicyID: "default", Failover: true}
	if labels == nil {
		return opts
	}
	if v, ok := labels["route.preferLocal"]; ok && (v == "false" || v == "0") {
		opts.PreferLocal = false
	}
	if v, ok := labels["route.failover"]; ok && (v == "false" || v == "0") {
		opts.Failover = false
	}
	if v := labels["route.preferProvider"]; v != "" {
		opts.PreferProvider = types.PluginID(v)
	}
	// aliases for operator UX
	if v := labels["route.provider"]; v != "" && opts.PreferProvider == "" {
		opts.PreferProvider = types.PluginID(v)
	}
	if v := labels["route.requireProvider"]; v != "" {
		opts.RequireProvider = types.PluginID(v)
	}
	if v := labels["route.preferRuntime"]; v != "" {
		opts.PreferRuntime = types.PluginID(v)
	}
	if v := labels["route.preferModel"]; v != "" {
		opts.PreferModel = v
	}
	if v := labels["route.model"]; v != "" && opts.PreferModel == "" {
		opts.PreferModel = v
	}
	if v := labels["route.excludeProvider"]; v != "" {
		opts.ExcludeProvider = map[types.PluginID]bool{types.PluginID(v): true}
	}
	if v := labels["route.excludeRuntime"]; v != "" {
		opts.ExcludeRuntime = map[types.PluginID]bool{types.PluginID(v): true}
	}
	// Comma-separated allowlist: route.providers=provider.a,provider.b
	if v := labels["route.providers"]; v != "" {
		opts.AllowProviders = map[types.PluginID]bool{}
		for _, part := range strings.Split(v, ",") {
			part = strings.TrimSpace(part)
			if part != "" {
				opts.AllowProviders[types.PluginID(part)] = true
			}
		}
	}
	if v := labels["route.policyId"]; v != "" {
		opts.PolicyID = v
	}
	return opts
}

func (k *Kernel) executeMission(
	ctx context.Context,
	id types.MissionID,
	req []types.Capability,
	labels map[string]string,
	mode types.AgentMode,
	providers []provider.Provider,
	runtimes []runtime.Runtime,
) error {
	r := router.New(providers, runtimes)
	opts := routeOptionsFromLabels(labels)
	if opts.PolicyID == "default" || opts.PolicyID == "" {
		opts.PolicyID = k.policy.ID
	}
	candidates, err := r.Candidates(ctx, req, opts)
	if err != nil {
		return err
	}
	decision := candidates[0]

	_ = k.bus.Publish(ctx, eventbus.Event{
		Type: "route.decided", MissionID: id,
		Data: map[string]any{
			"providerId":          string(decision.ProviderID),
			"runtimeId":           string(decision.RuntimeID),
			"modelId":             decision.ModelID,
			"costTier":            string(decision.CostTier),
			"reason":              decision.Reason,
			"required":            capsStrings(decision.Required),
			"policyId":            decision.PolicyID,
			"providersConsidered": decision.ProvidersConsidered,
			"runtimesConsidered":  decision.RuntimesConsidered,
			"failoverChain":       len(candidates),
		},
	})

	// Resolve runtime for sandbox policy (even if we later skip execute)
	var rt runtime.Runtime
	var rtDesc runtime.Descriptor
	for _, rtx := range runtimes {
		if rtx.ID() == decision.RuntimeID {
			rt = rtx
			rtDesc, _ = rtx.Describe(ctx)
			break
		}
	}
	if rt == nil {
		return fmt.Errorf("runtime %s not found", decision.RuntimeID)
	}
	if err := k.policy.CheckSandbox(rtDesc.SandboxTier); err != nil {
		return err
	}

	external := labels != nil && (labels["security.externalAction"] == "true" || labels["security.externalAction"] == "1")
	sec := security.EvaluateMode(mode, external)
	_ = k.bus.Publish(ctx, eventbus.Event{
		Type: "security.evaluated", MissionID: id,
		Data: map[string]any{
			"mode": string(sec.Mode), "allowExecute": sec.AllowExecute,
			"requireApproval": sec.RequireApproval, "reason": sec.Reason,
			"scopes": sec.Scopes, "externalAction": external,
		},
	})

	k.mu.Lock()
	if mission := k.missions[id]; mission != nil {
		mission.ProviderID = decision.ProviderID
		mission.RuntimeID = decision.RuntimeID
		mission.ModelID = decision.ModelID
		mission.RouteReason = decision.Reason
		mission.Mode = mode
		mission.SecurityNote = sec.Reason
	}
	goal := ""
	if mission := k.missions[id]; mission != nil {
		goal = mission.Goal
	}
	k.mu.Unlock()

	// Assist + external → await approval (no execute)
	if sec.RequireApproval {
		k.mu.Lock()
		if mission := k.missions[id]; mission != nil {
			mission.State = host.StateAwaitingApproval
			mission.Output = sec.Reason
			mission.UpdatedAt = time.Now().UTC()
		}
		k.mu.Unlock()
		_ = k.bus.Publish(ctx, eventbus.Event{
			Type: "mission.updated", MissionID: id,
			Data: map[string]any{"state": string(host.StateAwaitingApproval), "reason": sec.Reason},
		})
		return nil
	}

	// Observe → route + security journal only
	if !sec.AllowExecute {
		out := sec.Reason
		k.mu.Lock()
		if mission := k.missions[id]; mission != nil {
			mission.State = host.StateSucceeded
			mission.Output = out
			mission.UpdatedAt = time.Now().UTC()
		}
		k.mu.Unlock()
		_ = k.bus.Publish(ctx, eventbus.Event{
			Type: "mission.updated", MissionID: id,
			Data: map[string]any{"state": string(host.StateSucceeded), "output": out, "mode": string(mode)},
		})
		return nil
	}

	memHits, _ := k.memory.Search(ctx, memorystore.Query{MissionID: id, Limit: 20})
	global, _ := k.memory.Search(ctx, memorystore.Query{Kind: memorystore.KindSemantic, Limit: 5})
	memHits = append(memHits, global...)

	// Multi-provider failover: try each candidate until execute succeeds
	var result runtime.Result
	var lastErr error
	for i, cand := range candidates {
		decision = cand
		var handle credentials.Handle
		if h, ok := k.creds.FindByPlugin(ctx, decision.ProviderID); ok {
			handle = h
		} else {
			handle, lastErr = k.creds.Put(ctx, string(decision.ProviderID), "mission", decision.ProviderID, "hermes-demo-token")
			if lastErr != nil {
				continue
			}
		}
		_ = k.bus.Publish(ctx, eventbus.Event{
			Type: "credential.issued", MissionID: id,
			Data: map[string]any{"handle": string(handle), "scope": string(decision.ProviderID), "rank": i},
		})

		k.mu.Lock()
		if mission := k.missions[id]; mission != nil {
			mission.ProviderID = decision.ProviderID
			mission.RuntimeID = decision.RuntimeID
			mission.ModelID = decision.ModelID
			mission.RouteReason = decision.Reason
		}
		k.mu.Unlock()

		k.wireCompleter(rt)
		env := runtime.ContextEnvelope{
			Mission: map[string]any{
				"id": string(id), "goal": goal, "mode": string(mode),
				"requiredCapabilities": capsStrings(req),
			},
			Workspace: map[string]any{"profile": "host-neutral"},
			Memory:    memorystore.AsMaps(memHits),
			Credentials: []map[string]any{
				{"handle": string(handle), "scope": string(decision.ProviderID)},
			},
			Tools:  k.toolMaps(),
			Budget: map[string]any{"maxSteps": k.policy.MaxSteps, "maxCostUsd": k.policy.MaxCostUSD},
			Security: map[string]any{
				"sandbox": rtDesc.SandboxTier,
				"mode":    string(mode),
				"scopes":  sec.Scopes,
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
			Data: map[string]any{
				"runtimeId": string(decision.RuntimeID), "providerId": string(decision.ProviderID),
				"modelId": decision.ModelID, "sandbox": rtDesc.SandboxTier, "rank": i,
			},
		})

		result, lastErr = rt.Execute(ctx, env)
		if lastErr == nil && result.Status != "failed" {
			if i > 0 {
				_ = k.bus.Publish(ctx, eventbus.Event{
					Type: "provider.failover", MissionID: id,
					Data: map[string]any{
						"succeededProviderId": string(decision.ProviderID),
						"modelId":             decision.ModelID,
						"rank":                i,
					},
				})
			}
			lastErr = nil
			break
		}
		errMsg := result.Output
		if lastErr != nil {
			errMsg = lastErr.Error()
		} else {
			lastErr = fmt.Errorf("provider %s execute status=%s", decision.ProviderID, result.Status)
		}
		_ = k.bus.Publish(ctx, eventbus.Event{
			Type: "provider.failed", MissionID: id,
			Data: map[string]any{
				"providerId": string(decision.ProviderID), "modelId": decision.ModelID,
				"error": errMsg, "rank": i, "willFailover": i+1 < len(candidates),
			},
		})
	}
	if lastErr != nil {
		return lastErr
	}
	if err := k.policy.CheckSteps(result.StepsUsed); err != nil {
		return err
	}
	if err := k.policy.CheckBudget(result.CostUSD); err != nil {
		return err
	}

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

	// Content-addressed artifact of mission output (CG-ARTIFACT)
	var artDigest types.ArtifactDigest
	if result.Output != "" && k.Artifacts != nil {
		if meta, err := k.Artifacts.Put(ctx, []byte(result.Output), "text/plain", id, map[string]string{
			"kind": "mission-output",
		}); err == nil {
			artDigest = meta.Digest
			_ = k.bus.Publish(ctx, eventbus.Event{
				Type: "artifact.created", MissionID: id,
				Data: map[string]any{"digest": string(meta.Digest), "size": meta.Size},
			})
		}
	}
	// Knowledge graph node for mission (KG-GRAPH integration)
	if k.Knowledge != nil {
		n := k.Knowledge.UpsertNode(knowledge.Node{
			Type: "mission", Props: map[string]string{"id": string(id), "goal": goal},
		})
		_ = n
	}

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

	data := map[string]any{"state": string(state), "output": result.Output}
	if artDigest != "" {
		data["artifactDigest"] = string(artDigest)
	}
	_ = k.bus.Publish(ctx, eventbus.Event{
		Type: "mission.updated", MissionID: id, Data: data,
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

func (k *Kernel) toolMaps() []map[string]any {
	if k.tools == nil {
		return []map[string]any{{"id": "echo", "name": "echo"}}
	}
	list := k.tools.List()
	out := make([]map[string]any, 0, len(list))
	for _, t := range list {
		out = append(out, map[string]any{"id": t.ID, "name": t.Name, "description": t.Description})
	}
	return out
}

var _ host.Interface = (*Kernel)(nil)
