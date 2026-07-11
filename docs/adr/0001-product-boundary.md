# ADR-0001: Hermes as product; AESP remains protocol

## Status

Accepted — 2026-07-11

## Context

The AESP suite (AESP, AESP-Examples, AESP-Reference-Implementation) must stay vendor-neutral and protocol-focused. Product work (kernel services, Mission Control, plugin marketplace shape, operator UX) was accumulating inside AESP-RI, risking:

- Protocol pollution with product assumptions  
- Slow protocol evolution due to UI coupling  
- Ambiguous ownership of “Agent OS” vs “protocol compliance”  

Master Execution Program v1.0 requires a maintainable, multi-year product platform.

## Decision

1. Create **Hermes-Agent-OS** as the product home for the AI Operating System.  
2. Keep **AESP / AESP-Examples / AESP-Reference-Implementation** as upstream protocol / examples / compliance.  
3. Hermes **implements** AESP; AESP **never** depends on Hermes.  
4. Long-term product features land in Hermes. AESP-RI retains conformance and may host transitional product code until parity.

## Consequences

- Clear dependency arrow: Hermes → AESP (one way).  
- Dual maintenance during migration; no big-bang delete of AESP-RI product.  
- Contributors must check which repo owns a change (protocol vs product).  

## Alternatives considered

| Alternative | Why rejected |
|-------------|--------------|
| Keep everything in AESP-RI forever | Couples protocol to product; poor long-term maintainability |
| Fork AESP into Hermes-owned protocol | Breaks vendor-neutral standard governance |
| Only a UI repo, kernel stays in AESP-RI | Kernel *is* product; half-split still pollutes protocol repo |
