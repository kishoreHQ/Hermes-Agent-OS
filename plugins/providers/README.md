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
