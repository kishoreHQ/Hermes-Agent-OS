// Package conformance maps Hermes Agent OS to AESP suite expectations (H1.1).
// Normative requirements live upstream in AESP; this package is an implementer claim + runtime checks.
package conformance

// Status of a catalog requirement for this product version.
type Status string

const (
	StatusImplemented Status = "implemented"
	StatusPartial     Status = "partial"
	StatusGap         Status = "gap-filed"
	StatusNA          Status = "n/a-profile"
)

// Item is one tracked AESP / invariant requirement.
type Item struct {
	ID       string `json:"id"`
	Spec     string `json:"spec"`
	Title    string `json:"title"`
	Status   Status `json:"status"`
	Module   string `json:"module"`
	Profile  string `json:"profile,omitempty"` // core-runtime | hermes-agent-os | host
	Notes    string `json:"notes,omitempty"`
	CheckID  string `json:"checkId,omitempty"` // runtime check id if executable
}

// Profile is a named suite claim (see AESP specification/CONFORMANCE.md).
type Profile struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	ItemIDs     []string `json:"itemIds"`
}

// ImplementationVersion for claim reports.
const (
	ImplementationVersion = "hermesd-host/0.6.0"
	SuiteVersion          = "hermes-conformance/1.1.0"
	ClaimProfile          = "aesp.profile.hermes-core"
)

// Catalog returns the Hermes product mapping of high-priority AESP MUSTs and INVs.
func Catalog() []Item {
	return []Item{
		// Core identity / messaging
		{ID: "CORE-WORKUNIT", Spec: "AESP-0001", Title: "WorkUnit/mission identity and admission", Status: StatusImplemented, Module: "pkg/kernel+pkg/host", Profile: "core-runtime", CheckID: "host.mission_admit"},
		{ID: "CORE-ROLES", Spec: "AESP-0002", Title: "Agent principal registry and roles", Status: StatusGap, Module: "", Profile: "core-runtime", Notes: "Agent registry product surface not yet first-class"},
		{ID: "CORE-EVENTS", Spec: "AESP-0003", Title: "Event envelope and bus with monotonic seq", Status: StatusImplemented, Module: "pkg/eventbus", Profile: "core-runtime", CheckID: "obs.event_seq"},

		// Cognition / memory
		{ID: "MEM-UNIFIED", Spec: "AESP-0004", Title: "Unified memory with trust labels", Status: StatusImplemented, Module: "pkg/memorystore", Profile: "hermes-agent-os", CheckID: "mem.trust_write"},
		{ID: "WF-ORCH", Spec: "AESP-0005", Title: "Orchestrated multi-agent workflow engine", Status: StatusPartial, Module: "pkg/kernel", Profile: "hermes-agent-os", Notes: "Single-mission execute path; multi-agent DAG deferred"},
		{ID: "KG-GRAPH", Spec: "AESP-0006", Title: "Knowledge graph upsert/query", Status: StatusGap, Module: "", Profile: "hermes-agent-os", Notes: "Plugin kind reserved; no graph store yet"},

		// Delivery / ops (gaps acknowledged)
		{ID: "CG-ARTIFACT", Spec: "AESP-0007", Title: "Content-addressed artifacts", Status: StatusGap, Module: "", Profile: "hermes-agent-os"},
		{ID: "DOC-GEN", Spec: "AESP-0008", Title: "Documentation generator pipeline", Status: StatusGap, Module: "", Profile: "build-ship"},
		{ID: "DEP-ROLLOUT", Spec: "AESP-0009", Title: "Deployment session orchestration", Status: StatusGap, Module: "", Profile: "build-ship"},
		{ID: "TEST-EVAL", Spec: "AESP-0010", Title: "Evaluation harness distinct from agent harness", Status: StatusImplemented, Module: "pkg/evaluation", Profile: "hermes-agent-os", CheckID: "eval.suite"},
		{ID: "OBS-EVENTS", Spec: "AESP-0011", Title: "Mission event journal / replay", Status: StatusImplemented, Module: "pkg/eventbus+pkg/kernel", Profile: "hermes-agent-os", CheckID: "obs.replay"},
		{ID: "REM-PLAYBOOK", Spec: "AESP-0012", Title: "Remediation playbook engine", Status: StatusGap, Module: "", Profile: "hermes-agent-os"},
		{ID: "SEC-POLICY", Spec: "AESP-0013", Title: "Policy fail-closed + security modes", Status: StatusPartial, Module: "pkg/policy+pkg/security", Profile: "core-runtime", CheckID: "sec.modes", Notes: "Modes/scopes/signing; full classification matrix deferred"},
		{ID: "HITL-NO-AUTO", Spec: "AESP-0014", Title: "HITL: assist external awaits approval (no auto-approve)", Status: StatusPartial, Module: "pkg/security+pkg/kernel", Profile: "mission-control", CheckID: "hitl.assist", Notes: "Awaiting-approval state; full HITL task API deferred"},
		{ID: "INT-PROVIDER", Spec: "AESP-0015", Title: "Provider registry capability advertisement", Status: StatusImplemented, Module: "pkg/plugin+pkg/provider", Profile: "hermes-agent-os", CheckID: "int.providers"},
		{ID: "INT-RUNTIME", Spec: "AESP-0015", Title: "Runtime registry discovery", Status: StatusImplemented, Module: "pkg/plugin+pkg/runtime", Profile: "hermes-agent-os", CheckID: "int.runtimes"},
		{ID: "INT-TOOLS", Spec: "AESP-0015", Title: "Unified tool router + invocation records", Status: StatusImplemented, Module: "pkg/toolrouter", Profile: "hermes-agent-os", CheckID: "int.tools"},
		{ID: "INT-PLAN", Spec: "AESP-0015", Title: "Versioned plan artifacts", Status: StatusGap, Module: "", Profile: "hermes-agent-os"},
		{ID: "INT-MCP", Spec: "AESP-0015", Title: "MCP-aligned tool server/client", Status: StatusGap, Module: "", Profile: "hermes-agent-os"},
		{ID: "INT-A2A", Spec: "AESP-0015", Title: "A2A peer registry", Status: StatusGap, Module: "", Profile: "hermes-agent-os"},

		// Hermes INV alignment (product principles, AESP-compatible)
		{ID: "INV-01", Spec: "INV", Title: "Provider ≠ Runtime separate plugins", Status: StatusImplemented, Module: "pkg/provider+pkg/runtime", Profile: "core-runtime", CheckID: "inv.provider_ne_runtime"},
		{ID: "INV-02", Spec: "INV", Title: "Everything is a plugin", Status: StatusImplemented, Module: "pkg/plugin", Profile: "core-runtime", CheckID: "inv.plugins"},
		{ID: "INV-03", Spec: "INV", Title: "Capability-based routing (never model-name primary key)", Status: StatusImplemented, Module: "pkg/capability+pkg/router", Profile: "core-runtime", CheckID: "inv.capability_route"},
		{ID: "INV-05", Spec: "INV", Title: "Shared context envelope (prompt is one field)", Status: StatusImplemented, Module: "pkg/runtime", Profile: "hermes-agent-os", CheckID: "inv.context_envelope"},
		{ID: "INV-06", Spec: "INV", Title: "Unified memory owned by Hermes", Status: StatusImplemented, Module: "pkg/memorystore", Profile: "hermes-agent-os", CheckID: "mem.trust_write"},
		{ID: "INV-07", Spec: "INV", Title: "Credential broker handles only", Status: StatusImplemented, Module: "pkg/credentials", Profile: "hermes-agent-os", CheckID: "inv.credentials"},
		{ID: "INV-10", Spec: "INV", Title: "Audit journal + replayable routing", Status: StatusImplemented, Module: "pkg/eventbus+pkg/kernel", Profile: "core-runtime", CheckID: "obs.replay"},
		{ID: "INV-11", Spec: "INV", Title: "Host-neutral interface (in-process + HTTP)", Status: StatusImplemented, Module: "pkg/host+pkg/httpapi", Profile: "host", CheckID: "host.api_health"},
	}
}

// Profiles returns named suite profiles Hermes can claim or target.
func Profiles() []Profile {
	return []Profile{
		{
			ID:          "aesp.profile.hermes-core",
			Title:       "Hermes Core Runtime (claimed)",
			Description: "Executable host + plugins + capability routing + memory + credentials + journal. Subset of hermes-agent-os.",
			ItemIDs: []string{
				"CORE-WORKUNIT", "CORE-EVENTS", "MEM-UNIFIED", "TEST-EVAL", "OBS-EVENTS",
				"SEC-POLICY", "HITL-NO-AUTO", "INT-PROVIDER", "INT-RUNTIME", "INT-TOOLS",
				"INV-01", "INV-02", "INV-03", "INV-05", "INV-06", "INV-07", "INV-10", "INV-11",
			},
		},
		{
			ID:          "aesp.profile.hermes-agent-os",
			Title:       "Hermes Agent OS full pilot (target)",
			Description: "Full AESP CONFORMANCE.md hermes-agent-os profile — includes gaps filed above.",
			ItemIDs: []string{
				"CORE-WORKUNIT", "CORE-ROLES", "CORE-EVENTS", "MEM-UNIFIED", "WF-ORCH", "KG-GRAPH",
				"TEST-EVAL", "OBS-EVENTS", "SEC-POLICY", "HITL-NO-AUTO",
				"INT-PROVIDER", "INT-RUNTIME", "INT-TOOLS", "INT-PLAN",
				"INV-01", "INV-02", "INV-03", "INV-05", "INV-06", "INV-07", "INV-10", "INV-11",
			},
		},
	}
}

// Note: INT-TOOLS closed in H3.1 via pkg/toolrouter.

// ItemByID finds a catalog item.
func ItemByID(id string) (Item, bool) {
	for _, it := range Catalog() {
		if it.ID == id {
			return it, true
		}
	}
	return Item{}, false
}
