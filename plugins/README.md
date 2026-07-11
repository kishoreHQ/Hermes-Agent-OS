# Hermes plugins

All integrations live here (or in future extracted repos).  
**Kernel never hardcodes vendors** — only plugin manifests + adapters.

| Directory | Kind | Role |
|-----------|------|------|
| `providers/` | `provider` | Model inference plugins |
| `runtimes/` | `runtime` | Agent harness plugins |
| `tools/` | `tool` | Hermes-defined tools runtimes consume |
| `channels/` | `channel` | Telegram, Slack, etc. |
| `memory/` | `memory` | Episodic / semantic / graph adapters |
| `policy/` | `policy` | Cost, HITL, residency rules |

Each plugin ships:

1. `plugin.yaml` (or `.yml`) — `apiVersion: hermes.plugin/v1`  
2. `labels.hermes.driver` — selects an in-tree factory (e.g. `echo-provider`)  
3. Implementation registered in `kernel/pkg/bootstrap` (or future out-of-process loaders)  
4. Tests proving health + capability descriptors  

### Shipped examples (H2)

| Path | Driver | Role |
|------|--------|------|
| `providers/example-echo` | `echo-provider` | free-local inference |
| `providers/example-budget` | `echo-provider` | budget-tier inference (routing demo) |
| `runtimes/example-echo` | `echo-runtime` | one-step harness via Completer |
| `runtimes/example-steps` | `steps-runtime` | multi-step harness (H4 second runtime) |
| `memory/ephemeral` | `memory-ephemeral` | discovery marker; kernel owns store |

### Soft routing labels (H4)

Missions may set labels (not capabilities) to steer among *capable* plugins:

| Label | Effect |
|-------|--------|
| `route.preferLocal` | `true`/`false` — prefer local providers first |
| `route.preferProvider` | soft prefer plugin id |
| `route.preferRuntime` | soft prefer runtime id |
| `route.excludeProvider` | exclude plugin id |
| `route.excludeRuntime` | exclude runtime id |

Primary key remains **requiredCapabilities**.

Env: `HERMES_PLUGINS` may point at additional roots (`:`-separated).

See [docs/adr/0002-everything-is-a-plugin.md](../docs/adr/0002-everything-is-a-plugin.md), [docs/adr/0007-plugin-runtime-h2.md](../docs/adr/0007-plugin-runtime-h2.md).
