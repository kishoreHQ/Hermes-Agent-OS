# Hermes Agent OS — Target Architecture

## System diagram

```
                    Hermes Agent OS

                   Mission Control UI
                     (host client)
                           │
                      Host Interface
                     /api/v1 · WS events
                           │
                   Agent Runtime Kernel
                           │
 ┌──────────────┬──────────────┬──────────────┐
 │  Planning    │  Execution   │  Memory      │  Knowledge
 └──────────────┴──────────────┴──────────────┘
                           │
                   Capability Router
                           │
                   Provider Router
                           │
                   Runtime Router
                           │
        Tool Layer · Plugin Layer · Security Layer
                           │
              ┌────────────┴────────────┐
              ▼                         ▼
         Provider Plugins          Runtime Plugins
         (models)                  (harnesses)
```

Providers sit **beneath** the runtime path for model access.  
Agents never bind to vendor SDKs directly.

---

## Repository layout (this monorepo bootstrap)

Hermes-Agent-OS starts as a **product monorepo** that may later split along clean package boundaries. Recommended long-term topology (see ADR-0005):

| Package / future repo | Responsibility |
|----------------------|----------------|
| `kernel/` | Host-neutral Agent Runtime Kernel |
| `plugins/providers/` | Provider plugins |
| `plugins/runtimes/` | Runtime plugins |
| `plugins/tools/` · `channels/` · `memory/` · `policy/` | Other plugin classes |
| `mission-control/` | Operator UI (host) |
| `sdk/` | Client SDKs |
| `schemas/` | JSON/YAML contracts |
| `docs/` | Vision, ADRs, gates, security |
| `evaluations/` | Process logs, golden traces |
| `examples/` | Product examples (not AESP protocol examples) |
| `scripts/` | Dev and CI helpers |

Protocol upstream remains **outside** this tree: AESP, AESP-Examples, AESP-Reference-Implementation.

---

## Kernel packages (Go)

```
kernel/
  cmd/hermesd/          # process entry
  pkg/
    types/              # domain IDs, trust, tiers, modes
    host/               # Host Interface (INV-11)
    plugin/             # manifest, registry, loader, factories (INV-02)
    provider/           # Provider contract (INV-01)
    runtime/            # Runtime + ContextEnvelope (INV-01, INV-05)
    capability/         # normalize / match (INV-03)
    router/             # capability → provider/runtime (INV-03)
    credentials/        # unified credential broker (INV-07)
    memorystore/        # unified memory (INV-06)
    eventbus/           # monotonic seq journal (INV-10)
    security/           # modes, scopes, sandbox, HMAC signing (H5)
    policy/             # budgets, min sandbox, default mode
    evaluation/         # golden mission suite
    perf/               # latency baselines + benchmarks
    hardening/          # H5 composite prove
    interchange/        # H4 matrix prove
    httpapi/            # /api/v1 Host HTTP + WS
    adapters/echo/      # example provider + runtime (no vendor)
    adapters/steps/     # multi-step example runtime
    bootstrap/          # factory registration + disk load
    kernel/             # Kernel: security → route → execute → memory
```

**Invariant:** no vendor package imports in `pkg/kernel`, `pkg/router`, `pkg/capability`, `pkg/host`.

---

## Plugin model

Every plugin has a **Manifest**:

```yaml
apiVersion: hermes.plugin/v1
kind: provider   # or runtime | tool | channel | ...
metadata:
  id: provider.example.openai-compatible
  version: 1.0.0
  name: Example OpenAI-Compatible
spec: {}         # kind-specific
labels: {}
```

Registry operations: `Register`, `Get`, `List(kind)`.  
In-process `MemoryRegistry` ships first; process/remote loaders are future plugins of kind `storage` / loader ADRs.

---

## Routing path (replayable)

1. Normalize required capabilities (drop model-name anti-patterns).  
2. Filter providers by health + capability compatibility.  
3. Apply policy / budget / security / availability (extensible).  
4. Order by cost tier (`free-local` → … → `premium`).  
5. Select model from provider descriptor.  
6. Select healthy runtime (later: capability-in/out + sandbox tier).  
7. Emit `router.Decision` with reason + policy id for journal.

---

## Host Interface

Hosts (Mission Control, CLI, bots) talk only through:

| Concern | Contract |
|---------|----------|
| Submit / cancel / list / get missions | `host.Interface` + `/api/v1/missions` |
| Event stream | monotonic `seq`; JSON catch-up + WebSocket `/api/v1/events` |
| Replay | `/api/v1/replay/{id}` · `/api/v1/missions/{id}/events` |
| Registry | `/api/v1/registry/{providers\|runtimes\|tools}` |
| Health | `/api/v1/health` |
| Envelope | `{ "data": …, "error": null \| {code,message,remediation} }` |

UI never imports kernel packages. UI binds to HTTP/WS only.  
OpenAPI: `schemas/openapi-host-v1.yaml`.

---

## Security boundaries (target)

| Boundary | Rule |
|----------|------|
| Credentials | Broker issues handles; secrets never in ContextEnvelope plaintext |
| Trust | Memory writes carry `TrustLabel` |
| Sandbox | Runtime descriptor declares tier (`micro-vm` \| `container` \| `process-pty`) |
| Policy | Deny-by-default for external actions in Assist mode; Observe journals only |
| Plugins | Signed manifests (Phase 9 / ADT-12 track) |

---

## Relationship to AESP-RI product code

AESP-Reference-Implementation currently holds a **working** Agent OS + Mission Control monorepo (gates 1–7, UI phases). That code is the **compliance + prototype product track**.

Hermes-Agent-OS is the **clean product home**:

- Re-implements and evolves platform contracts without polluting AESP protocol repos.  
- May **absorb** mature modules from AESP-RI via deliberate ports (not git subtree of protocol).  
- AESP-RI remains the place for protocol conformance runners and reference behaviors.

See [RELATIONSHIP.md](./RELATIONSHIP.md).

---

## Non-goals (near term)

- Replacing AESP as the protocol authority  
- Hardcoding Claude Code / OpenAI as kernel defaults  
- Shipping a consumer chatbot UX as the product definition  
- Merging protocol RFCs into Hermes docs as normative text (link upstream)  
