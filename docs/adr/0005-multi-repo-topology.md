# ADR-0005: Multi-repo topology (bootstrap monorepo first)

## Status

Accepted — 2026-07-11

## Context

Master Execution Program recommends eventual repos:

Hermes-Kernel, Hermes-Providers, Hermes-Runtimes, Hermes-Workflow, Hermes-Memory, Hermes-Knowledge, Hermes-Mission-Control, Hermes-Tools, Hermes-Evaluation, Hermes-SDK, Hermes-Starter-Agents.

Splitting on day one before contracts stabilize creates version hell and blocks coherent review.

## Decision

1. **Bootstrap in one product monorepo:** `Hermes-Agent-OS` with packages `kernel/`, `plugins/*`, `mission-control/`, `sdk/`, etc.  
2. **Keep AESP suite separate** (ADR-0001).  
3. **Extract** sibling repos only when:  
   - Package has independent release cadence, **and**  
   - Public API is stable enough for semver, **and**  
   - H4 interchangeability proof has passed (see PLAN.md).  
4. Extraction requires an ADR amend listing module boundaries and CI.

## Consequences

- Faster H0–H3 iteration.  
- Directory layout mirrors future repo names to reduce future move cost.  
- Resist premature `go.work` multi-module sprawl until needed.

## Alternatives considered

| Alternative | Why rejected |
|-------------|--------------|
| Create all 11 empty GitHub repos now | Process overhead, no code, stale READMEs |
| Keep product inside AESP-RI only | Rejected by ADR-0001 |
