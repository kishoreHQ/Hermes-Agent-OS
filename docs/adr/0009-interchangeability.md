# ADR-0009: Interchangeability of providers and runtimes (H4)

## Status

Accepted — 2026-07-11

## Context

Success criteria require that providers and runtimes are interchangeable without kernel changes.  
Soft preferences (prefer local, prefer a runtime id) must not replace capability-based routing as the primary key.

## Decision

1. Ship **≥2 providers** and **≥2 runtimes** as plugins (echo free-local, budget, echo runtime, steps runtime).  
2. Router selects by: capability match → exclusions/health → soft prefer → cost tier → stable id.  
3. Mission **labels** may soft-steer (`route.preferProvider`, `route.preferRuntime`, `route.exclude*`, `route.preferLocal`) without code changes.  
4. Every decision emits `route.decided` with `required`, `reason`, candidate counts — never model-name primary key.  
5. H4 gate is automated: `hermesd prove-h4` / `make prove-h4` runs the 2×2 matrix.

## Consequences

- Kernel path is fixed; swaps are plugin/label configuration.  
- Adding a third vendor is a new plugin + optional labels only.  
- Operator-facing "prefer plugin X" is allowed as preference, not as capability.

## Alternatives considered

| Alternative | Why rejected |
|-------------|--------------|
| Hardcode if/else per vendor in kernel | Violates INV-02 |
| Model name as capability | Violates INV-03 |
