# Hermes Agent OS — Program Plan v1.0

> Product roadmap for **Hermes-Agent-OS**.  
> Protocol and historical monorepo work remain in the AESP suite; see [RELATIONSHIP.md](./RELATIONSHIP.md).  
> AESP-RI `docs/PLAN.md` still governs Phase 8–9 on the reference monorepo until parity ports land here.

---

## 1. Current state

**Shipped in this repository (foundation):**

- Product boundary documented (VISION, PRINCIPLES, ARCHITECTURE, RELATIONSHIP, ADRs)  
- Kernel skeleton: types, host, plugin registry, provider/runtime contracts, capability engine, router, `hermesd` stub  
- Unit tests: capability normalize, plugin registry, router decision path  
- Multi-plugin directory scaffold  

**Still in AESP-Reference-Implementation (product prototype track):**

- Full agent loop, memory, artifacts, HITL, conformance  
- Mission Control UI (UI-GATES 1–7), Command Deck K1–K7  
- Phase 8–9 ADTs (full CLI loops, Telegram, cost tiers, modes, heartbeat, signed manifests)

---

## 2. Program phases (Hermes product)

### Phase H0 — Product foundation (this milestone)

- [x] Create Hermes-Agent-OS repository  
- [x] ADR-0001…0005  
- [x] Kernel contracts + skeleton  
- [x] Makefile / README / LICENSE  
- [x] First tagged foundation on `master` (`85213e7`)  

### Phase H1 — Host API parity

- [x] Host HTTP `/api/v1` + event `seq` (JSON catch-up + WebSocket)  
- [x] Mission submit/cancel/list/get; event journal; health  
- [x] Registry: providers / runtimes / tools from plugin registry  
- [x] OpenAPI under `schemas/openapi-host-v1.yaml`  
- [x] `hermesd serve`, `make smoke`  
- [ ] Conformance harness that validates Hermes against full AESP Host expectations (H1.1)  

### Phase H2 — Plugin runtime

- [x] Disk plugin loader (manifest discovery + driver factories)  
- [x] In-tree example providers (echo free-local + budget) + echo runtime  
- [x] Credential broker (handles only; Host lists metadata)  
- [x] Memory store interface + in-memory implementation + search API  
- [x] Mission path: route → credential → runtime → provider complete → memory write  
- [x] Events: `route.decided`, `credential.issued`, `runtime.*`, `memory.written`  
- [ ] Out-of-process / RPC plugin sandbox (H2.1 / later)  

### Phase H3 — Mission Control product home

- [x] Re-home Mission Control under `mission-control/` (Vite/React/Tailwind)  
- [x] Bind exclusively to Hermes Host API (no vendor SDKs, no kernel imports)  
- [x] Cherenkov ops aesthetic (overview, missions, fleet, memory, events, credentials)  
- [x] Vite proxy + `hermesd` SPA serve (`mission-control/dist`)  
- [ ] Port full Command Deck (connections/board/routines) when Host API grows (H3.1)  

### Phase H4 — Interchangeability proof

- [x] ≥2 provider plugins (echo free-local + budget)  
- [x] ≥2 runtime plugins (echo single-step + steps multi-step)  
- [x] Same mission succeeds under 2×2 swap via labels only (no kernel edit)  
- [x] Replay includes `route.decided` with capabilities + reason  
- [x] `make prove-h4` / `hermesd prove-h4` automated gate  

### Phase H5 — Production hardening

- Security review (modes, scopes, sandbox tiers, signed manifests)  
- Performance baselines  
- Evaluation harness  
- Multi-repo extract decision (ADR amend if splitting)  

---

## 3. Success criteria (program complete)

From Master Execution Program:

1. Multi-agent workflows execute under Hermes.  
2. Providers interchangeable.  
3. Runtimes interchangeable.  
4. New vendors = plugins only.  
5. Mission Control vendor-independent.  
6. AESP conformance green.  
7. Security boundaries enforced.  
8. Shared memory & knowledge.  
9. Deterministic replay.  
10. Policies enforceable.  
11. Production-ready maintainability.  

---

## 4. Human checkpoints

| ID | Gate | Who |
|----|------|-----|
| HC-H0 | Foundation docs + skeleton merge | Program lead |
| HC-H1 | Host API smoke on Mac | Operator |
| HC-H3 | Mission Control against Hermes kernel | Operator |
| HC-H5 | Security posture review | Security + lead |

AESP-RI HC-1…3 still apply to the reference monorepo until Hermes owns those demos.

---

## 5. Execution discipline

Every task:

Architecture Review → Dependency Analysis → Implementation Plan → Implementation → Unit Tests → Integration Tests → Protocol Conformance → Security Review → Performance Review → Documentation → Peer Review → Refactor → Merge  

Autonomous reviewer personas (approve / reject / request changes):

Principal Architect · Platform · Security · Performance · DX · Documentation · Protocol  

---

## 6. Explicit non-goals for H0–H1

- Do not couple AESP protocol docs to Hermes UI themes  
- Do not hardcode OpenAI/Anthropic in kernel  
- Do not declare Phase 8–9 complete until gates pass on the owning repo  
- Do not force-split multi-repo before H4 proof  
