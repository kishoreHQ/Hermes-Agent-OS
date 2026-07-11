# ADR-0012: AESP conformance claim for Hermes product

## Status

Accepted — 2026-07-11

## Context

AESP `specification/CONFORMANCE.md` defines suite profiles (including `aesp.profile.hermes-agent-os`).  
AESP-RI ships a static MUST catalog. Hermes needs an **executable** product claim that:

1. Lists profile + suite + implementation versions (objective claim rules)  
2. Distinguishes **implemented / partial / gap** honestly  
3. Runs runtime checks for green claims  
4. Does not invent protocol semantics outside AESP  

## Decision

1. Add `kernel/pkg/conformance` with catalog + executable checks.  
2. **Claimed profile (green):** `aesp.profile.hermes-core` — host, plugins, capability routing, memory, credentials, journal, modes, evaluation.  
3. **Target profile (not green):** `aesp.profile.hermes-agent-os` — enumerates gaps (KG, artifacts, MCP, multi-agent DAG, …).  
4. CLI: `hermesd conform [core|full]` · Make: `make conform` / `make conform-full`.  
5. Gaps are first-class status `gap-filed`, never silent omissions.

## Consequences

- Marketing “AESP compliant” must cite `aesp.profile.hermes-core` until full profile is green.  
- Closing a gap = implement + flip catalog status + add check.  
- Protocol remains upstream; Hermes only maps and tests.

## Alternatives considered

| Alternative | Why rejected |
|-------------|--------------|
| Import AESP-RI conformance package | Couples product to RI layout |
| Claim full hermes-agent-os now | Dishonest; many gaps remain |
