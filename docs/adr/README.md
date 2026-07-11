# Architecture Decision Records

| ID | Title | Status |
|----|-------|--------|
| [0001](./0001-product-boundary.md) | Hermes as product; AESP remains protocol | Accepted |
| [0002](./0002-everything-is-a-plugin.md) | Everything is a plugin | Accepted |
| [0003](./0003-provider-neq-runtime.md) | Provider ≠ Runtime | Accepted |
| [0004](./0004-capability-based-routing.md) | Capability-based routing | Accepted |
| [0005](./0005-multi-repo-topology.md) | Multi-repo topology (monorepo first) | Accepted |
| [0006](./0006-host-api-surface.md) | Host API surface (`/api/v1`) | Accepted |
| [0007](./0007-plugin-runtime-h2.md) | Plugin runtime (disk loader + factories) | Accepted |
| [0008](./0008-mission-control-host.md) | Mission Control as a Hermes host | Accepted |
| [0009](./0009-interchangeability.md) | Interchangeability of providers and runtimes | Accepted |
| [0010](./0010-monorepo-retention.md) | Retain product monorepo through H5 | Accepted |
| [0011](./0011-security-modes.md) | Agent modes, scopes, sandbox, signed plugins | Accepted |
| [0012](./0012-aesp-conformance-claim.md) | AESP conformance claim for Hermes product | Accepted |
| [0013](./0013-command-deck-and-openai-compat.md) | Command Deck + OpenAI-compatible provider | Accepted |
| [0014](./0014-catalog-gaps-closed.md) | Close remaining AESP catalog gaps by impact | Accepted |
| [0015](./0015-multi-provider-failover.md) | Multi-provider failover + model discovery | Accepted |

Process: propose → challenge (reviewer personas) → accept/reject → implement.  
Superseding an ADR requires a new ADR that links the old one.
