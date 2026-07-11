# ADR-0003: Provider ≠ Runtime

## Status

Accepted — 2026-07-11

## Context

Industry language collapses “Claude”, “GPT”, and “Claude Code” into one “model” concept. That confuses:

- Who supplies tokens (provider)  
- Who runs the agent loop / tools / PTY (runtime)  

Mixing them breaks routing, credentials, sandbox policy, and fleet health.

## Decision

1. **Provider** plugin: models + completion (or equivalent inference). Does not own multi-step agent execution.  
2. **Runtime** plugin: executes work with a ContextEnvelope. Does not own the global model catalog.  
3. Separate Go interfaces: `provider.Provider` and `runtime.Runtime`.  
4. Routing selects **both** independently after capability/policy/budget/security filters.  
5. Product copy and UI must not present “provider” and “runtime” as synonyms.

## Consequences

- Clear fleet views: model providers vs agent harnesses.  
- A local Ollama provider can pair with many runtimes; Claude Code runtime can use non-Anthropic providers where the harness allows (via Hermes tools/context).  
- Adapters that wrap “all-in-one” SaaS must still split concerns at the Hermes boundary (even if the upstream product mixes them).

## Alternatives considered

| Alternative | Why rejected |
|-------------|--------------|
| Single “backend” abstraction | Hides sandbox vs inference policy differences |
| Runtime owns provider SDK | Forces N×M credential and version coupling |
