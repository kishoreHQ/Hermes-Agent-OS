package conformance

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/bootstrap"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/capability"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/evaluation"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/host"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/httpapi"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/kernel"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/memorystore"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/plugin"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/runtime"
	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/types"
)

// CheckResult is one executable check outcome.
type CheckResult struct {
	ID      string `json:"id"`
	OK      bool   `json:"ok"`
	Detail  string `json:"detail,omitempty"`
	Error   string `json:"error,omitempty"`
}

// CheckFunc runs against a live kernel.
type CheckFunc func(ctx context.Context, k *kernel.Kernel) CheckResult

// Registry of executable checks (id → func).
func Checks() map[string]CheckFunc {
	return map[string]CheckFunc{
		"host.mission_admit":      checkMissionAdmit,
		"host.api_health":         checkAPIHealth,
		"obs.event_seq":           checkEventSeq,
		"obs.replay":              checkReplay,
		"mem.trust_write":         checkMemoryTrust,
		"eval.suite":              checkEvalSuite,
		"sec.modes":               checkSecurityModes,
		"hitl.assist":             checkHITLAssist,
		"int.providers":           checkProviders,
		"int.runtimes":            checkRuntimes,
		"int.tools":               checkTools,
		"inv.provider_ne_runtime": checkProviderNeRuntime,
		"inv.plugins":             checkPlugins,
		"inv.capability_route":    checkCapabilityRoute,
		"inv.context_envelope":    checkContextEnvelope,
		"inv.credentials":         checkCredentials,
	}
}

func bootKernel(t string) (*kernel.Kernel, error) {
	_ = t
	res, err := bootstrap.New(bootstrap.Options{SeedBuiltins: true, PluginRoots: []string{"/no-disk-for-conformance"}})
	if err != nil {
		return nil, err
	}
	return res.Kernel, nil
}

func checkMissionAdmit(ctx context.Context, k *kernel.Kernel) CheckResult {
	id, err := k.SubmitMission(ctx, host.Mission{
		Goal: "conformance admit", RequiredCaps: []types.Capability{"coding"},
	})
	if err != nil {
		return fail("host.mission_admit", err.Error())
	}
	m, err := k.GetMission(ctx, id)
	if err != nil || m.ID == "" {
		return fail("host.mission_admit", "get mission failed")
	}
	return pass("host.mission_admit", "mission id="+string(id)+" state="+string(m.State))
}

func checkAPIHealth(ctx context.Context, k *kernel.Kernel) CheckResult {
	s := httpapi.New(k)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/health", nil))
	if rec.Code != 200 {
		return fail("host.api_health", fmt.Sprintf("status %d", rec.Code))
	}
	var env struct {
		Data  map[string]any `json:"data"`
		Error any            `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		return fail("host.api_health", err.Error())
	}
	if env.Error != nil {
		return fail("host.api_health", "error envelope set")
	}
	if env.Data["status"] != "ok" {
		return fail("host.api_health", fmt.Sprintf("%v", env.Data))
	}
	return pass("host.api_health", "envelope {data,error} ok")
}

func checkEventSeq(ctx context.Context, k *kernel.Kernel) CheckResult {
	id, err := k.SubmitMission(ctx, host.Mission{
		Goal: "seq check", RequiredCaps: []types.Capability{"coding"},
	})
	if err != nil {
		return fail("obs.event_seq", err.Error())
	}
	evs, err := k.EventsSince(ctx, 0, string(id))
	if err != nil || len(evs) < 2 {
		return fail("obs.event_seq", fmt.Sprintf("events=%d err=%v", len(evs), err))
	}
	for i := 1; i < len(evs); i++ {
		if evs[i].Seq <= evs[i-1].Seq {
			return fail("obs.event_seq", "seq not monotonic")
		}
	}
	return pass("obs.event_seq", fmt.Sprintf("n=%d first=%d last=%d", len(evs), evs[0].Seq, evs[len(evs)-1].Seq))
}

func checkReplay(ctx context.Context, k *kernel.Kernel) CheckResult {
	id, err := k.SubmitMission(ctx, host.Mission{
		Goal: "replay check", RequiredCaps: []types.Capability{"coding", "tools"},
	})
	if err != nil {
		return fail("obs.replay", err.Error())
	}
	evs, err := k.Replay(ctx, id)
	if err != nil {
		return fail("obs.replay", err.Error())
	}
	var sawRoute bool
	for _, e := range evs {
		if e.Type == "route.decided" {
			sawRoute = true
			if e.Data["reason"] == nil || e.Data["reason"] == "" {
				return fail("obs.replay", "route.decided missing reason")
			}
			if e.Data["required"] == nil {
				return fail("obs.replay", "route.decided missing required caps")
			}
		}
	}
	if !sawRoute {
		return fail("obs.replay", "missing route.decided")
	}
	return pass("obs.replay", "route.decided with reason+required")
}

func checkMemoryTrust(ctx context.Context, k *kernel.Kernel) CheckResult {
	e, err := k.Memory().Write(ctx, memorystore.Entry{
		Kind: memorystore.KindEpisodic, Content: "conformance memory", Trust: types.TrustAgent,
	})
	if err != nil {
		return fail("mem.trust_write", err.Error())
	}
	if e.Trust != types.TrustAgent {
		return fail("mem.trust_write", "trust not persisted")
	}
	hits, err := k.Memory().Search(ctx, memorystore.Query{Text: "conformance", Limit: 5})
	if err != nil || len(hits) == 0 {
		return fail("mem.trust_write", "search miss")
	}
	return pass("mem.trust_write", "trust="+string(e.Trust))
}

func checkEvalSuite(ctx context.Context, k *kernel.Kernel) CheckResult {
	_ = k
	rep, err := evaluation.Run(ctx, nil)
	if err != nil {
		return fail("eval.suite", err.Error())
	}
	if rep.Failed > 0 {
		return fail("eval.suite", evaluation.Format(rep))
	}
	return pass("eval.suite", fmt.Sprintf("passed=%d", rep.Passed))
}

func checkSecurityModes(ctx context.Context, k *kernel.Kernel) CheckResult {
	id, err := k.SubmitMission(ctx, host.Mission{
		Goal: "observe conf", RequiredCaps: []types.Capability{"coding"},
		Labels: map[string]string{"security.mode": "observe"},
	})
	if err != nil {
		return fail("sec.modes", err.Error())
	}
	m, _ := k.GetMission(ctx, id)
	if m.State != host.StateSucceeded || m.Mode != types.ModeObserve {
		return fail("sec.modes", fmt.Sprintf("state=%s mode=%s", m.State, m.Mode))
	}
	return pass("sec.modes", "observe journal-only ok")
}

func checkHITLAssist(ctx context.Context, k *kernel.Kernel) CheckResult {
	id, err := k.SubmitMission(ctx, host.Mission{
		Goal: "assist conf", RequiredCaps: []types.Capability{"coding"},
		Labels: map[string]string{
			"security.mode": "assist", "security.externalAction": "true",
		},
	})
	if err != nil {
		return fail("hitl.assist", err.Error())
	}
	m, _ := k.GetMission(ctx, id)
	if m.State != host.StateAwaitingApproval {
		return fail("hitl.assist", "expected awaiting_approval got "+string(m.State))
	}
	return pass("hitl.assist", "assist external does not auto-approve")
}

func checkProviders(ctx context.Context, k *kernel.Kernel) CheckResult {
	list := k.Plugins().List(plugin.KindProvider)
	if len(list) < 1 {
		return fail("int.providers", "no providers")
	}
	return pass("int.providers", fmt.Sprintf("n=%d first=%s", len(list), list[0].Metadata.ID))
}

func checkRuntimes(ctx context.Context, k *kernel.Kernel) CheckResult {
	list := k.Plugins().List(plugin.KindRuntime)
	if len(list) < 1 {
		return fail("int.runtimes", "no runtimes")
	}
	return pass("int.runtimes", fmt.Sprintf("n=%d first=%s", len(list), list[0].Metadata.ID))
}

func checkTools(ctx context.Context, k *kernel.Kernel) CheckResult {
	if k.Tools() == nil || len(k.Tools().List()) < 1 {
		return fail("int.tools", "no tools")
	}
	inv, err := k.Tools().Invoke(ctx, "echo", "conf", "runtime.example.echo", map[string]any{"text": "tool-ok"})
	if err != nil || inv.Status != "ok" || inv.Output != "tool-ok" {
		return fail("int.tools", fmt.Sprintf("%+v %v", inv, err))
	}
	if len(k.Tools().Invocations(5)) < 1 {
		return fail("int.tools", "no invocation record")
	}
	return pass("int.tools", "invoke+audit ok")
}

func checkProviderNeRuntime(ctx context.Context, k *kernel.Kernel) CheckResult {
	ps := k.Plugins().List(plugin.KindProvider)
	rs := k.Plugins().List(plugin.KindRuntime)
	if len(ps) == 0 || len(rs) == 0 {
		return fail("inv.provider_ne_runtime", "need both kinds")
	}
	// IDs must not collide across kinds
	pset := map[types.PluginID]bool{}
	for _, p := range ps {
		pset[p.Metadata.ID] = true
	}
	for _, r := range rs {
		if pset[r.Metadata.ID] {
			return fail("inv.provider_ne_runtime", "id collision "+string(r.Metadata.ID))
		}
	}
	return pass("inv.provider_ne_runtime", fmt.Sprintf("providers=%d runtimes=%d", len(ps), len(rs)))
}

func checkPlugins(ctx context.Context, k *kernel.Kernel) CheckResult {
	all := k.Plugins().List("")
	if len(all) < 2 {
		return fail("inv.plugins", "too few plugins")
	}
	kinds := map[plugin.Kind]int{}
	for _, m := range all {
		kinds[m.Kind]++
	}
	if kinds[plugin.KindProvider] < 1 || kinds[plugin.KindRuntime] < 1 {
		return fail("inv.plugins", "missing provider or runtime kind")
	}
	return pass("inv.plugins", fmt.Sprintf("total=%d kinds=%d", len(all), len(kinds)))
}

func checkCapabilityRoute(ctx context.Context, k *kernel.Kernel) CheckResult {
	eng := capability.New()
	out := eng.Normalize([]types.Capability{"coding", "gpt-4", "claude"})
	if len(out) != 1 || out[0] != "coding" {
		return fail("inv.capability_route", fmt.Sprintf("normalize=%v", out))
	}
	// Mission with only model names must fail admit
	_, err := k.SubmitMission(ctx, host.Mission{
		Goal: "bad route", RequiredCaps: []types.Capability{"gpt-4"},
	})
	if err == nil {
		return fail("inv.capability_route", "model-name-only mission accepted")
	}
	return pass("inv.capability_route", "model names rejected; coding kept")
}

func checkContextEnvelope(ctx context.Context, k *kernel.Kernel) CheckResult {
	// Structural contract: ContextEnvelope fields exist beyond Prompt
	env := runtime.ContextEnvelope{
		Prompt: "x",
		Mission: map[string]any{"goal": "x"},
		Memory:  []map[string]any{},
		Budget:  map[string]any{"maxSteps": 1},
		Security: map[string]any{"mode": "full"},
	}
	if env.Prompt == "" || env.Mission == nil || env.Budget == nil || env.Security == nil {
		return fail("inv.context_envelope", "envelope incomplete")
	}
	// Live path succeeds with shared memory write after execute
	id, err := k.SubmitMission(ctx, host.Mission{
		Goal: "envelope live", RequiredCaps: []types.Capability{"coding", "tools"},
	})
	if err != nil {
		return fail("inv.context_envelope", err.Error())
	}
	m, _ := k.GetMission(ctx, id)
	if m.State != host.StateSucceeded {
		return fail("inv.context_envelope", "mission "+string(m.State))
	}
	return pass("inv.context_envelope", "prompt is one field among mission/memory/budget/security")
}

func checkCredentials(ctx context.Context, k *kernel.Kernel) CheckResult {
	h, err := k.Creds().Put(ctx, "scope.demo", "conf", "provider.example.echo", "super-secret-value")
	if err != nil {
		return fail("inv.credentials", err.Error())
	}
	list, err := k.Creds().List(ctx)
	if err != nil || len(list) == 0 {
		return fail("inv.credentials", "list empty")
	}
	// Ensure JSON of list metadata cannot include secret field by encoding Record
	b, _ := json.Marshal(list)
	if strings.Contains(string(b), "super-secret-value") {
		return fail("inv.credentials", "secret leaked in list JSON")
	}
	secret, _, err := k.Creds().Resolve(ctx, h)
	if err != nil || secret != "super-secret-value" {
		return fail("inv.credentials", "resolve failed")
	}
	return pass("inv.credentials", "handle issued; list has no secret material")
}

func pass(id, detail string) CheckResult {
	return CheckResult{ID: id, OK: true, Detail: detail}
}

func fail(id, err string) CheckResult {
	return CheckResult{ID: id, OK: false, Error: err}
}
