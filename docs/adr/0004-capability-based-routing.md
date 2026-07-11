# ADR-0004: Capability-based routing

## Status

Accepted — 2026-07-11

## Context

Routing by model name (`gpt-4`, `claude-3.5`) hardcodes vendors into missions, policies, and UI. It prevents free-tier escalation, local-first policies, and replay that explains *why* a model was chosen.

## Decision

1. Missions declare **requiredCapabilities** (e.g. `coding`, `tools`, `long-context`), never model names as capabilities.  
2. Capability engine **normalizes** and **rejects** known model-name anti-patterns.  
3. Router path: Intent → Capabilities → Policy → Budget → Security → Availability → Provider → Model → Runtime → …  
4. Every `router.Decision` records reason, tier, required caps, provider/runtime ids for journal/replay.  
5. Cost tiers are ordered: `free-local` < `free-hosted` < `budget` < `standard` < `premium`.

## Consequences

- Policies like `free-first-coding` become data, not code forks.  
- UI shows capability + tier chips, not only vendor logos.  
- Incomplete capability taxonomies need iterative design (evaluation loop).

## Alternatives considered

| Alternative | Why rejected |
|-------------|--------------|
| User always picks model | Breaks automation, budget, and fleet policies |
| Embed model id in mission as soft preference only | Allowed later as *preference*, never as sole routing key — must still pass capability path |
