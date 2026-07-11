# Relationship: AESP Suite ↔ Hermes Agent OS

## Product vs protocol

```
┌─────────────────────────────────────────────────────────────┐
│  PROTOCOL LAYER (vendor-neutral, slow-changing)             │
│                                                             │
│   AESP                        AESP-Examples                 │
│   (standard)                  (canonical examples only)     │
│              \                      /                       │
│               \                    /                        │
│                v                  v                         │
│           AESP-Reference-Implementation                     │
│           (protocol compliance + historical product RI)     │
└─────────────────────────────────────────────────────────────┘
                              │
                              │ implements / conforms
                              v
┌─────────────────────────────────────────────────────────────┐
│  PRODUCT LAYER                                              │
│                                                             │
│   Hermes-Agent-OS                                           │
│   · Kernel · Plugins · Mission Control · SDK · Evaluations  │
│                                                             │
│   Future optional splits:                                   │
│   Hermes-Kernel, Hermes-Providers, Hermes-Runtimes, …       │
└─────────────────────────────────────────────────────────────┘
```

---

## Repository roles

| Repository | Role | May contain product UI/kernel? | Couples to Hermes? |
|------------|------|--------------------------------|--------------------|
| **AESP** | Protocol standard, RFCs, schemas | No | No — Hermes depends on AESP, not reverse |
| **AESP-Examples** | Canonical protocol illustrations | No production logic | No |
| **AESP-Reference-Implementation** | Conformance + reference behaviors; currently also hosts shipped Agent OS monorepo | Reference product only; long-term prefer Hermes | Should not import Hermes packages |
| **Hermes-Agent-OS** | Production product platform | Yes | Owns product roadmap |

---

## Dependency rules

1. **Hermes may depend on AESP** (schemas, conformance expectations, vocabulary).  
2. **AESP must never depend on Hermes** (no product types in protocol).  
3. **AESP-Examples** stay free of Hermes and free of production services.  
4. **AESP-RI** may share *ideas* and *ports* with Hermes; prefer dual maintenance of conformance tests in RI, product evolution in Hermes.  
5. Protocol changes that Hermes needs → propose upstream in AESP with justification; do not fork protocol semantics inside Hermes.

---

## Migration posture

| Phase | AESP-RI | Hermes-Agent-OS |
|-------|---------|-----------------|
| Now | Ship gates 1–7, continue Phase 8–9 work as needed | Foundation: contracts, ADRs, kernel skeleton |
| Next | Freeze new product features that are not conformance | Port Host API, plugins, Mission Control as product modules |
| Steady | Conformance + protocol demos | Sole product home; multi-repo split if load demands |

No big-bang delete of AESP-RI product code until Hermes reaches parity gates.

---

## Vocabulary alignment

Hermes adopts AESP-aligned terms where they exist:

- Mission, Work Unit, Artifact, Trust labels  
- Host Interface patterns  
- Capability language (not model-name routing)  

Hermes-specific product terms (Mission Control surfaces, Command Deck, Cherenkov UI) stay in Hermes docs — never in AESP normative text.

---

## Conformance

- Protocol conformance: **AESP-RI** `conformance` remains the green bar for AESP.  
- Product acceptance: Hermes gates (see `docs/gates/`) + AESP conformance still green when Hermes speaks AESP.  
- Hermes must not ship “almost AESP” dialects without an ADR and upstream proposal.
