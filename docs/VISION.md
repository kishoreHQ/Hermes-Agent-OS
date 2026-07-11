# Hermes Agent OS — Vision

**Hermes is a vendor-neutral AI Operating System** powered by the AESP protocol suite.

It is not a chatbot, workflow builder, coding harness, or dashboard bolted onto a single model vendor.  
It is middleware: a runtime orchestration platform that makes agents, models, tools, memory, knowledge, and policy first-class, interchangeable, and auditable.

---

## What Hermes is

| Layer | Role |
|-------|------|
| **Mission Control** | Operator UI and host clients (one of many possible hosts) |
| **Agent Runtime Kernel** | Host-neutral core: missions, events, routing, plugins |
| **Planning / Execution / Memory / Knowledge** | Platform services above capability routing |
| **Capability · Provider · Runtime routers** | Intent → capabilities → policy → budget → security → provider → model → runtime |
| **Tool · Plugin · Security layers** | Dynamically discovered; never hardcoded vendors |

Hermes owns:

- **Runtime orchestration** — multi-agent workflows with shared context  
- **Memory & knowledge** — episodic, semantic, procedural, graph, artifacts, provenance  
- **Policy & security** — enforceable, replayable, trust-labeled  
- **Provider & runtime abstraction** — plugins only; kernel has zero vendor names  
- **Mission Control** — production operator experience independent of any model vendor  

---

## What Hermes is not

- Another chat product  
- Another no-code workflow toy  
- Another IDE coding assistant  
- Another vendor SDK wrapper  
- The AESP protocol itself  

**AESP** is the protocol standard (vendor-neutral contract).  
**AESP-Examples** are canonical illustrations only.  
**AESP-Reference-Implementation** is protocol compliance + historical product monorepo (upstream/compliance track).  
**Hermes-Agent-OS** is the product platform built *on* AESP.

---

## North-star outcomes

The program is complete only when:

1. Multi-agent workflows execute end-to-end under Hermes.  
2. Providers are interchangeable without kernel changes.  
3. Runtimes are interchangeable without kernel changes.  
4. New vendors require **only** a plugin (manifest + adapter).  
5. Mission Control operates without knowledge of specific vendors.  
6. AESP conformance remains green.  
7. Security boundaries are enforced (credentials, trust, policy, sandbox tiers).  
8. Memory and knowledge are shared across all runtimes.  
9. Routing and execution decisions are deterministic under replay.  
10. The platform is production-ready for long-term maintainability.

---

## Design stance

Think like the teams behind Kubernetes, VS Code, Temporal, OpenTelemetry, or OpenAPI:

- **Abstractions first** — providers ≠ runtimes; capabilities ≠ model names  
- **Extension over fork** — everything is a plugin  
- **Contracts over convenience** — AESP remains pure; Hermes implements  
- **Replay over vibes** — every routing and policy decision is journaled  
- **Decades, not demos** — vendor neutrality and lifecycle clarity beat short-term coupling  

See also: [PRINCIPLES.md](./PRINCIPLES.md), [ARCHITECTURE.md](./ARCHITECTURE.md), [RELATIONSHIP.md](./RELATIONSHIP.md).
