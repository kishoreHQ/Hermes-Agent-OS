# ADR-0015: Multi-provider failover and model discovery

## Status

Accepted — 2026-07-11

## Decision

1. Router builds an **ordered candidate chain** of providers (not a single pick).  
2. Prefer-local lists locals first; **failover appends** non-local healthy providers.  
3. On execute/complete failure, kernel tries the next candidate; journals `provider.failed` / `provider.failover`.  
4. Optional **ModelCatalog.ListModels** for auto-discovery (OpenAI-compat `GET /models`).  
5. Operators may select provider/model via mission fields or labels; capabilities remain primary.

See `docs/MULTI_PROVIDER.md`.
