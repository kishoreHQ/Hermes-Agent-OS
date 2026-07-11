# ADR-0007: Plugin runtime (disk loader + factories)

## Status

Accepted — 2026-07-11

## Context

H1 exposed a Host API but missions did not execute through real provider/runtime plugins.  
Hardcoding adapters in the kernel would violate INV-02 (everything is a plugin).

## Decision

1. **Manifest discovery** — `plugin.Loader` walks plugin roots for `plugin.yaml`.  
2. **Driver factories** — `labels.hermes.driver` selects an in-tree constructor (`echo-provider`, `echo-runtime`, `memory-ephemeral`).  
3. **Kernel execution path** — SubmitMission → capability normalize → route → credential handle → ContextEnvelope → runtime.Execute → memory write → journal events.  
4. **Credentials** — unified broker; Host API lists **handles only** (INV-07).  
5. **Memory** — kernel-owned `memorystore.Store`; runtimes never own global memory (INV-06).  
6. **Echo adapters** — vendor-neutral deterministic plugins for tests and demos (not production vendors).

## Consequences

- Adding a real OpenAI-compatible provider = new factory + plugin.yaml (kernel path unchanged).  
- Disk load requires `HERMES_PLUGINS` or running from repo layout with `plugins/`.  
- Seed builtins activate when no disk plugins load (tests / bare binary).

## Alternatives considered

| Alternative | Why rejected |
|-------------|--------------|
| External process plugins only (RPC) | Premature for H2; in-process first |
| Compile-time plugin registry only | Blocks drop-in YAML discovery |
