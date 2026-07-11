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

1. `plugin.yaml` (or `.json`) manifest — `apiVersion: hermes.plugin/v1`  
2. Implementation package (Go preferred for in-tree)  
3. Tests proving health + capability descriptors  

See [docs/adr/0002-everything-is-a-plugin.md](../docs/adr/0002-everything-is-a-plugin.md).
