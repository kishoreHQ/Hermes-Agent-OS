# ADR-0010: Retain product monorepo through H5 (amend ADR-0005)

## Status

Accepted — 2026-07-11

## Context

ADR-0005 deferred multi-repo extraction until H4 interchangeability and API stability.  
H4 passed. H5 adds security, evaluation, and performance baselines.  

Extraction candidates (Hermes-Kernel, Hermes-Providers, Hermes-Mission-Control, …) remain valid long-term, but splitting now would:

- Force premature semver on unstable Host OpenAPI edges  
- Multiply CI without independent release cadence yet  
- Slow security/eval iteration that spans kernel + plugins + UI  

## Decision

1. **Keep Hermes-Agent-OS as a single product monorepo** for the foreseeable post-H5 period.  
2. Extraction criteria remain (all must hold):  
   - Independent release cadence required  
   - Public API stable enough for semver  
   - CI cost of monorepo exceeds multi-repo overhead  
3. Directory layout continues to mirror future repo names (`kernel/`, `plugins/`, `mission-control/`, `sdk/`).  
4. Revisit after H3.1 deck parity or first external plugin consumer — whichever comes first.

## Consequences

- One clone builds Host API + plugins + Mission Control.  
- `go.mod` stays under `kernel/`; UI under npm.  
- No forced version matrix between Hermes packages yet.

## Alternatives considered

| Alternative | Why deferred |
|-------------|--------------|
| Split all 11 repos now | Process cost > benefit; APIs still evolving |
| Extract only Mission Control | Possible later; monorepo SPA serve is simpler for H5 |
