# ADR-0008: Mission Control as a Hermes host (H3)

## Status

Accepted — 2026-07-11

## Context

AESP-RI hosts a full Command Deck UI. Hermes needs a **product** Mission Control that:

- Lives in Hermes-Agent-OS  
- Depends only on Hermes Host API  
- Does not import kernel packages or vendor SDKs  
- Does not require AESP-RI at runtime  

Porting the entire deck (connections, boards, routines) would invent Host endpoints Hermes does not yet expose.

## Decision

1. Re-home Mission Control under `mission-control/` as a Vite + React + Tailwind host.  
2. **H3 scope** = surfaces backed by existing Host API: Overview, Missions, Fleet, Memory, Events, Credentials.  
3. Dev: Vite proxies `/api` → `hermesd :8080`.  
4. Prod: `hermesd` serves `mission-control/dist` SPA when present (`HERMES_UI_DIST` override).  
5. AESP-RI UI remains the reference for Phase 8–9 deck demos until Hermes Host API grows those routes.

## Consequences

- Operators can drive H0–H2 kernel from a first-party UI.  
- Deck parity is explicit future work (not silent stubs that 404).  
- UI and kernel version independently; contract is OpenAPI Host.

## Alternatives considered

| Alternative | Why rejected |
|-------------|--------------|
| Git-subtree full AESP-RI ui | Couples product to RI; many dead routes |
| Keep UI only in external hermes-mission-control-ui | Violates product-home decision (ADR-0001) |
