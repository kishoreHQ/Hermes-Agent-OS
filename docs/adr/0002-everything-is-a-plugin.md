# ADR-0002: Everything is a plugin

## Status

Accepted — 2026-07-11

## Context

AI platforms often hardcode first-party vendors (one SDK, one harness, special cases in core). That produces lock-in and forces kernel changes for every integration.

## Decision

1. All integrations are **plugins** discovered via **manifest + registry**.  
2. Plugin kinds include at least: provider, runtime, tool, channel, memory, knowledge, workflow, policy, security, evaluation, storage, credential.  
3. Kernel packages never import concrete vendor SDKs as permanent core dependencies.  
4. First registry implementation is in-process `MemoryRegistry`; loaders (disk, process, remote) are themselves extensible later.

## Consequences

- Adding OpenAI / Ollama / Claude Code / Telegram = new plugin packages + manifests only.  
- Slightly more boilerplate for “hello world” integrations — accepted.  
- Manifest schema (`hermes.plugin/v1`) becomes a stability surface.

## Alternatives considered

| Alternative | Why rejected |
|-------------|--------------|
| Compile-time build tags per vendor | Still couples release matrix to vendors |
| Single mega-adapter with if/else | Unmaintainable; violates open/closed |
| WASM-only plugins from day one | Premature; in-process first, remote later |
