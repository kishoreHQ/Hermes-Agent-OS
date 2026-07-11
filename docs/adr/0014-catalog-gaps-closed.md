# ADR-0014: Close remaining AESP catalog gaps by impact order

## Status

Accepted — 2026-07-11

## Context

Hermes claimed `aesp.profile.hermes-core` while `hermes-agent-os` listed gaps. Gaps closed in **impact order** for an AI OS:

1. Content-addressed artifacts (provenance / handoff)  
2. Agent roles (authorization surface)  
3. Versioned plans (multi-step structure)  
4. Workflow orchestration (multi-step DAG)  
5. Knowledge graph  
6. MCP-shaped tools  
7. A2A peers  
8. Remediation playbooks  
9. Deploy sessions  
10. Docgen  

## Decision

| ID | Module | Status |
|----|--------|--------|
| CG-ARTIFACT | `pkg/artifact` | implemented |
| CORE-ROLES | `pkg/agentregistry` | implemented |
| INT-PLAN | `pkg/planner` | implemented |
| WF-ORCH | `pkg/workflow` | implemented |
| KG-GRAPH | `pkg/knowledge` | implemented |
| INT-MCP | `pkg/mcpbridge` | implemented (MCP-shaped bridge) |
| INT-A2A | `pkg/a2a` | implemented (peer registry + local tasks) |
| REM-PLAYBOOK | `pkg/remediation` | implemented |
| DEP-ROLLOUT | `pkg/deploy` | implemented |
| DOC-GEN | `pkg/docgen` | implemented |

Each has executable conformance checks. Host routes under `/api/v1/*`.

## Consequences

- `make conform-full` can claim `aesp.profile.hermes-agent-os` green at product pilot level.  
- MCP/A2A are **fixtures / bridges**, not full wire-protocol servers — sufficient for catalog pilot, deepen as needed.  
- Workflow runs child missions via kernel (capability routing preserved).
