# ADR-0011: Agent modes, scopes, sandbox, signed plugins

## Status

Accepted — 2026-07-11

## Context

Production hardening requires enforceable security boundaries without vendor hardcoding.

## Decision

1. **Modes:** `full` | `assist` | `observe` (mission field or `security.mode` label).  
   - Observe: route + journal only; no runtime execute.  
   - Assist: external actions (`security.externalAction=true`) → `awaiting_approval`.  
   - Full: execute within policy budgets.  
2. **Scopes:** mode-derived capability grants (memory/provider/runtime/events).  
3. **Sandbox:** policy `minSandboxTier` enforced against runtime descriptor.  
4. **Signed manifests:** optional HMAC-SHA256 via `labels.hermes.signature`;  
   `HERMES_PLUGIN_HMAC_KEY` + `HERMES_REQUIRE_SIGNED_PLUGINS=1` for enforce mode.  
5. **Credentials:** remain handle-only on Host API (INV-07).

## Consequences

- Security decisions are journaled as `security.evaluated` for replay.  
- Production deployments can require signed plugins without kernel vendor coupling.
