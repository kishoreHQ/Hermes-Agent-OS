# Provider plugins

Providers **supply models**. They do not execute multi-step agent work.

Contract: `kernel/pkg/provider.Provider`

Example layout (future):

```
providers/
  openai-compatible/
    plugin.yaml
    provider.go
  ollama/
    plugin.yaml
    provider.go
```

Capabilities are declarative (`coding`, `tools`, `vision`, …) — never model-name routing as the primary key.

### Shipped

| Plugin | Driver | Notes |
|--------|--------|-------|
| `example-echo` | `echo-provider` | Deterministic in-process |
| `example-budget` | `echo-provider` | Higher cost tier for routing demos |
| `openai-compat` | `openai-compat` | Real HTTP Chat Completions client (`baseURL` + API key handle) |
| `kimchi` | `openai-compat` | Kimchi Inference (`https://llm.kimchi.dev/openai/v1`) — see [docs/KIMCHI.md](../../docs/KIMCHI.md) |

Configure `openai-compat` via `plugin.yaml` `spec.baseURL` (default `http://127.0.0.1:11434/v1`).

**Kimchi live:** `export KIMCHI_API_KEY=…` then `make serve`. Key from [app.kimchi.dev/settings](https://app.kimchi.dev/settings).
