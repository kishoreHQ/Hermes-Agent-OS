# ADR-0013: Command Deck services + OpenAI-compatible provider

## Status

Accepted — 2026-07-11

## Context

H3 shipped core Mission Control. Operators need Command Deck surfaces (connect, sessions, board, routines).  
Production inference needs a real HTTP provider that is still vendor-neutral (OpenAI Chat Completions wire format is a de-facto common API).

## Decision

1. **Deck services** live in `kernel/pkg/deck` (connections, sessions, board, routines), exposed under `/api/v1/*`.  
2. Sessions/routines fire **missions** (capability routing) — no bypass of kernel.  
3. **OpenAI-compatible provider** (`hermes.driver: openai-compat`) uses configurable `baseURL` + credential handle; not a single cloud vendor.  
4. **Tool router** (`pkg/toolrouter`) owns Hermes tools + invocation audit records (closes INT-TOOLS).  
5. UI pages under Mission Control bind only to Host API.

## Consequences

- Deck is product-native on Hermes, not AESP-RI-coupled.  
- Any OpenAI-compatible server (Ollama, LM Studio, gateways) works via plugin config.  
- Full CLI PTY adapters remain future work; sessions use mission path for H3.1.
