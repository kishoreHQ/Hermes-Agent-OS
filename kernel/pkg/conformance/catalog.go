// Package conformance maps Hermes Agent OS to AESP suite expectations (H1.1+).
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
	ID      string `json:"id"`
	Spec    string `json:"spec"`
	Title   string `json:"title"`
	Status  Status `json:"status"`
	Module  string `json:"module"`
	Profile string `json:"profile,omitempty"`
	Notes   string `json:"notes,omitempty"`
	CheckID string `json:"checkId,omitempty"`
}

// Profile is a named suite claim (see AESP specification/CONFORMANCE.md).
type Profile struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	ItemIDs     []string `json:"itemIds"`
}

const (
	ImplementationVersion = "hermesd-host/0.7.0"
	SuiteVersion          = "hermes-conformance/1.2.0"
	ClaimProfile          = "aesp.profile.hermes-core"
)

// Catalog returns the Hermes product mapping of high-priority AESP MUSTs and INVs.
func Catalog() []Item {
	return []Item{
		{ID: "CORE-WORKUNIT", Spec: "AESP-0001", Title: "WorkUnit/mission identity and admission", Status: StatusImplemented, Module: "pkg/kernel+pkg/host", Profile: "core-runtime", CheckID: "host.mission_admit"},
		{ID: "CORE-ROLES", Spec: "AESP-0002", Title: "Agent principal registry and roles", Status: StatusImplemented, Module: "pkg/agentregistry", Profile: "core-runtime", CheckID: "core.roles"},
		{ID: "CORE-EVENTS", Spec: "AESP-0003", Title: "Event envelope and bus with monotonic seq", Status: StatusImplemented, Module: "pkg/eventbus", Profile: "core-runtime", CheckID: "obs.event_seq"},

		{ID: "MEM-UNIFIED", Spec: "AESP-0004", Title: "Unified memory with trust labels", Status: StatusImplemented, Module: "pkg/memorystore", Profile: "hermes-agent-os", CheckID: "mem.trust_write"},
		{ID: "WF-ORCH", Spec: "AESP-0005", Title: "Orchestrated multi-agent workflow engine", Status: StatusImplemented, Module: "pkg/workflow+pkg/planner", Profile: "hermes-agent-os", CheckID: "wf.orch"},
		{ID: "KG-GRAPH", Spec: "AESP-0006", Title: "Knowledge graph upsert/query", Status: StatusImplemented, Module: "pkg/knowledge", Profile: "hermes-agent-os", CheckID: "kg.graph"},

		{ID: "CG-ARTIFACT", Spec: "AESP-0007", Title: "Content-addressed artifacts", Status: StatusImplemented, Module: "pkg/artifact", Profile: "hermes-agent-os", CheckID: "cg.artifact"},
		{ID: "DOC-GEN", Spec: "AESP-0008", Title: "Documentation generator pipeline", Status: StatusImplemented, Module: "pkg/docgen", Profile: "build-ship", CheckID: "doc.gen"},
		{ID: "DEP-ROLLOUT", Spec: "AESP-0009", Title: "Deployment session orchestration", Status: StatusImplemented, Module: "pkg/deploy", Profile: "build-ship", CheckID: "dep.rollout"},
		{ID: "TEST-EVAL", Spec: "AESP-0010", Title: "Evaluation harness distinct from agent harness", Status: StatusImplemented, Module: "pkg/evaluation", Profile: "hermes-agent-os", CheckID: "eval.suite"},
		{ID: "OBS-EVENTS", Spec: "AESP-0011", Title: "Mission event journal / replay", Status: StatusImplemented, Module: "pkg/eventbus+pkg/kernel", Profile: "hermes-agent-os", CheckID: "obs.replay"},
		{ID: "REM-PLAYBOOK", Spec: "AESP-0012", Title: "Remediation playbook engine", Status: StatusImplemented, Module: "pkg/remediation", Profile: "hermes-agent-os", CheckID: "rem.playbook"},
		{ID: "SEC-POLICY", Spec: "AESP-0013", Title: "Policy fail-closed + security modes", Status: StatusImplemented, Module: "pkg/policy+pkg/security", Profile: "core-runtime", CheckID: "sec.modes"},
		{ID: "HITL-NO-AUTO", Spec: "AESP-0014", Title: "HITL: assist external awaits approval (no auto-approve)", Status: StatusImplemented, Module: "pkg/security+pkg/kernel", Profile: "mission-control", CheckID: "hitl.assist"},
		{ID: "INT-PROVIDER", Spec: "AESP-0015", Title: "Provider registry capability advertisement", Status: StatusImplemented, Module: "pkg/plugin+pkg/provider", Profile: "hermes-agent-os", CheckID: "int.providers"},
		{ID: "INT-RUNTIME", Spec: "AESP-0015", Title: "Runtime registry discovery", Status: StatusImplemented, Module: "pkg/plugin+pkg/runtime", Profile: "hermes-agent-os", CheckID: "int.runtimes"},
		{ID: "INT-TOOLS", Spec: "AESP-0015", Title: "Unified tool router + invocation records", Status: StatusImplemented, Module: "pkg/toolrouter", Profile: "hermes-agent-os", CheckID: "int.tools"},
		{ID: "INT-PLAN", Spec: "AESP-0015", Title: "Versioned plan artifacts", Status: StatusImplemented, Module: "pkg/planner", Profile: "hermes-agent-os", CheckID: "int.plan"},
		{ID: "INT-MCP", Spec: "AESP-0015", Title: "MCP-aligned tool server/client", Status: StatusImplemented, Module: "pkg/mcpbridge", Profile: "hermes-agent-os", CheckID: "int.mcp"},
		{ID: "INT-A2A", Spec: "AESP-0015", Title: "A2A peer registry", Status: StatusImplemented, Module: "pkg/a2a", Profile: "hermes-agent-os", CheckID: "int.a2a"},

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

// Profiles returns named suite profiles Hermes can claim.
func Profiles() []Profile {
	all := []string{
		"CORE-WORKUNIT", "CORE-ROLES", "CORE-EVENTS", "MEM-UNIFIED", "WF-ORCH", "KG-GRAPH",
		"CG-ARTIFACT", "DOC-GEN", "DEP-ROLLOUT", "TEST-EVAL", "OBS-EVENTS", "REM-PLAYBOOK",
		"SEC-POLICY", "HITL-NO-AUTO",
		"INT-PROVIDER", "INT-RUNTIME", "INT-TOOLS", "INT-PLAN", "INT-MCP", "INT-A2A",
		"INV-01", "INV-02", "INV-03", "INV-05", "INV-06", "INV-07", "INV-10", "INV-11",
	}
	return []Profile{
		{
			ID:          "aesp.profile.hermes-core",
			Title:       "Hermes Core Runtime (claimed)",
			Description: "Host + plugins + routing + memory + credentials + journal + tools + roles + security.",
			ItemIDs: []string{
				"CORE-WORKUNIT", "CORE-ROLES", "CORE-EVENTS", "MEM-UNIFIED", "TEST-EVAL", "OBS-EVENTS",
				"SEC-POLICY", "HITL-NO-AUTO", "INT-PROVIDER", "INT-RUNTIME", "INT-TOOLS",
				"INV-01", "INV-02", "INV-03", "INV-05", "INV-06", "INV-07", "INV-10", "INV-11",
			},
		},
		{
			ID:          "aesp.profile.hermes-agent-os",
			Title:       "Hermes Agent OS pilot profile",
			Description: "Full catalog claim when all items implemented + checks green.",
			ItemIDs:     all,
		},
	}
}

// ItemByID finds a catalog item.
func ItemByID(id string) (Item, bool) {
	for _, it := range Catalog() {
		if it.ID == id {
			return it, true
		}
	}
	return Item{}, false
}
