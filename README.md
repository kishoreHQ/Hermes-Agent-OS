# Hermes Agent OS

**Vendor-neutral AI Operating System / middleware platform** built on the [AESP](https://github.com/kishoreHQ/AESP) protocol suite.

Hermes is not a chatbot, a single-vendor assistant, or a workflow toy.  
It is a **runtime orchestration platform**: agents, providers, runtimes, memory, knowledge, policy, and Mission Control — with **everything as a plugin**.

```
Mission Control UI
        │
Agent Runtime Kernel
        │
Planning · Execution · Memory · Knowledge
        │
Capability → Provider → Runtime routers
        │
Tools · Plugins · Security
```

## Repository roles

| Repo | Role |
|------|------|
| [AESP](https://github.com/kishoreHQ/AESP) | Protocol standard (vendor-neutral) |
| [AESP-Examples](https://github.com/kishoreHQ/AESP-Examples) | Canonical examples only |
| [AESP-Reference-Implementation](https://github.com/kishoreHQ/AESP-Reference-Implementation) | Protocol compliance (+ transitional product monorepo) |
| **Hermes-Agent-OS** (this repo) | **Product platform** |

See [docs/RELATIONSHIP.md](./docs/RELATIONSHIP.md).

## Principles (non-negotiable)

- **Provider ≠ Runtime** — models vs harnesses  
- **Everything is a plugin** — no hardcoded vendors in the kernel  
- **Capability-based routing** — never primary-key on model names  
- **Unified memory, credentials, tools** — owned by Hermes  
- **Host-neutral kernel** — Mission Control is one host among many  

Full list: [docs/PRINCIPLES.md](./docs/PRINCIPLES.md).

## Quick start

```bash
# Unit tests
make test

# Build hermesd
make build

# Serve Host API (default :8080)
make serve

# Mission Control (dev — two terminals)
make serve          # :8080
make ui-dev         # :5173, proxies /api → kernel

# Mission Control (prod SPA via hermesd)
make ui-install && make ui-build
make serve          # open http://127.0.0.1:8080

# Smoke
make smoke
```

```bash
curl -s localhost:8080/api/v1/health
curl -s -X POST localhost:8080/api/v1/missions \
  -H 'Content-Type: application/json' \
  -d '{"goal":"hello","requiredCapabilities":["coding"]}'
curl -s 'localhost:8080/api/v1/events?since=0&format=json'
```

Requirements: Go 1.22+, Node 20+ (for Mission Control).

Host OpenAPI: [`schemas/openapi-host-v1.yaml`](./schemas/openapi-host-v1.yaml).

## Layout

```
kernel/              # Agent Runtime Kernel (Go)
plugins/             # Provider, runtime, tool, channel, memory, policy plugins
mission-control/     # Operator UI (host) — product home
sdk/                 # Client SDKs
schemas/             # Wire contracts
docs/                # Vision, architecture, ADRs, gates
evaluations/         # Process logs & golden traces
examples/            # Product examples
scripts/             # Dev helpers
```

## Documentation

| Doc | Purpose |
|-----|---------|
| [docs/VISION.md](./docs/VISION.md) | What Hermes is and is not |
| [docs/ARCHITECTURE.md](./docs/ARCHITECTURE.md) | Target architecture |
| [docs/PRINCIPLES.md](./docs/PRINCIPLES.md) | INV-01…11 |
| [docs/PLAN.md](./docs/PLAN.md) | Program phases H0–H5 |
| [docs/adr/](./docs/adr/) | Architecture decisions |

## Status

| Phase | State |
|-------|--------|
| **H0** Product foundation | Done |
| **H1** Host API `/api/v1` + events | Done |
| **H2** Plugin loader + execute path | Done |
| **H3** Mission Control re-home | Done (core host surfaces) |
| **H4** Interchangeability proof | Next |

Working Agent OS + Mission Control prototypes currently also live in AESP-Reference-Implementation until Hermes reaches full product parity (deliberate migration, not abandonment).

## License

Apache-2.0 — see [LICENSE](./LICENSE).
