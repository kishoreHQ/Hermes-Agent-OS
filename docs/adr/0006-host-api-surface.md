# ADR-0006: Host API surface (`/api/v1`)

## Status

Accepted — 2026-07-11

## Context

Hosts (Mission Control, CLIs, bots) need a stable, host-neutral HTTP/WS contract.  
AESP-RI already proven a `{data,error}` envelope and WS events with monotonic `seq`.  
Hermes must own the **product** Host API without importing AESP-RI packages.

## Decision

1. Expose Host Interface over HTTP under `/api/v1/*` with envelope `{data,error}`.  
2. Journal events with global monotonic `seq` (`eventbus.Bus`).  
3. Support **JSON catch-up** (`GET /api/v1/events?since=&format=json`) and **WebSocket live** (`GET /api/v1/events` upgrade).  
4. Missions require `requiredCapabilities` (capability routing); reject model-name-only submits.  
5. Registry endpoints list **plugin manifests**, not vendor hardcodes.  
6. Document wire contract in `schemas/openapi-host-v1.yaml`.

## Consequences

- Mission Control can bind to Hermes without kernel imports.  
- Integration tests can avoid WS by using `format=json`.  
- Full AESP Host conformance harness remains a follow-up (H1.1).  
- Deck-only routes (connections, boards, …) are **out of scope** for H1 — land later as host extensions or plugins.

## Alternatives considered

| Alternative | Why rejected |
|-------------|--------------|
| gRPC-only host API | Worse browser/Mission Control DX for H1 |
| Copy AESP-RI httpapi wholesale | Couples product to RI package layout |
| SSE only | WS already used by Mission Control; support both JSON + WS |
