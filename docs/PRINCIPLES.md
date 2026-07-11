# Hermes Agent OS — Architectural Principles

These principles are **mandatory**. Violations require an ADR that is challenged and approved before merge.

---

## INV-01 — Provider ≠ Runtime

**Providers** expose models (Anthropic, OpenAI, Google, xAI, Groq, Ollama, OpenRouter, …).  
**Runtimes** execute work (Claude Code, Codex CLI, Gemini CLI, OpenCode, Continue, Aider, OpenHands, Cline, …).

| | Provider | Runtime |
|--|----------|---------|
| Supplies | Models, completion APIs | Agent harness, tool loop, session I/O |
| Does not | Run multi-step agent work | Own model credentials or model catalog as product truth |
| Plugin kind | `provider` | `runtime` |

Agents and Mission Control **never** hardcode either. Routing selects both through the capability path.

---

## INV-02 — Everything is a Plugin

No integration is special-cased in the kernel.

| Kind | Examples |
|------|----------|
| `provider` | OpenAI-compatible, Anthropic Messages, Ollama |
| `runtime` | Named CLIs, PTY harness, sandbox-agent host |
| `tool` | Shell, browser, repo tools (Hermes-defined surface) |
| `channel` | Telegram, Slack, email |
| `memory` | Episodic store, vector, graph adapters |
| `knowledge` | Corpus indexers, retrieval plugins |
| `workflow` | DAG / temporal-style orchestrators |
| `policy` | Cost, data residency, HITL rules |
| `security` | Sandbox, signature, secret scopes |
| `evaluation` | Scorers, golden traces |
| `storage` | Artifact backends |
| `credential` | Brokers (handles only to runtimes) |

Discovery is dynamic (manifest + registry). Kernel never imports a vendor package by name as a permanent dependency of core.

---

## INV-03 — Capability-Based Routing

Routing is **never** by vendor or model-name string as the primary key.

```
Intent
  → Capabilities
  → Policy
  → Budget
  → Security
  → Availability
  → Provider
  → Model
  → Runtime
  → Execution
  → Validation
  → Memory
  → Artifacts
```

Every decision must be **replayable** (journaled reason, scores, tier, policy id).

Reject anti-patterns: treating `"gpt-4"`, `"claude"`, `"gemini"` as capabilities.

---

## INV-04 — Vendor Neutrality

- Hermes must not depend on a specific model vendor for correctness.  
- Adding a provider or runtime = **plugin only**. Kernel unchanged.  
- Product docs may list examples; code contracts must not.

---

## INV-05 — Shared Context Envelope

Every runtime receives a unified context (prompt is one field among many):

- Mission · Workspace · Policies · Knowledge · Memory  
- Artifacts · Credential **handles** · Tool registry  
- Security context · Budgets · User preferences · Correlation ids  

Runtimes do not invent private global state that bypasses Hermes memory.

---

## INV-06 — Unified Memory

Memory belongs to **Hermes**, not vendors.

Includes: episodic, semantic, procedural, knowledge graph, artifacts, evaluations, trust labels, provenance.

All runtimes read/write through Hermes memory plugins.

---

## INV-07 — Unified Credentials

One credential broker. Handles only cross the runtime boundary.  
No vendor-specific secret stores as the system of record.

---

## INV-08 — Unified Tools

Hermes defines the tool surface. Runtimes **consume** it (adapters translate).  
Runtimes do not own the global tool schema.

---

## INV-09 — Runtime Registry

Runtimes register capabilities, sandbox tier, health, and version.  
Fleet and routing use the registry — never hardcoded CLI names in kernel logic.

---

## INV-10 — Audit & Replay

- Event journal with monotonic `seq` per stream  
- Routing decisions with reason codes  
- Policy evaluations  
- Trust labels on memory writes  
- Deterministic replay of decisions given the same journal + config snapshot  

---

## INV-11 — Host-Neutral Kernel

Mission Control is a **host**. So are CLIs, bots, and other UIs.

Kernel exposes a Host Interface (`/api/v1` + events).  
Kernel never assumes a single product shell, theme, or channel.

Profiles (illustrative): P1 headless · P2 desktop operator · P3 multi-host.

---

## Engineering discipline

Before implementing:

1. Challenge architecture (coupling, lock-in, missing abstractions).  
2. Prefer ADR over silent drift.  
3. Follow: Architecture Review → Dependencies → Plan → Implement → Unit → Integration → Conformance → Security → Performance → Docs → Peer review → Refactor → Merge.  
4. Independent reviewer personas must approve (Principal Architect, Platform, Security, Performance, DX, Docs, Protocol).  

Do not skip stages for “just a small change” that touches invariants.
