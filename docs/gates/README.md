# Gates

Product acceptance gates for Hermes-Agent-OS.

| Gate | Intent | Status |
|------|--------|--------|
| H0 | Foundation docs + kernel skeleton | **PASS** |
| H1 | Host API parity smoke (`make smoke`) | **PASS** (core Host surface) |
| H2 | Plugin load + example provider/runtime | Pending |
| H3 | Mission Control against Hermes kernel | Pending |
| H4 | Interchangeability proof (≥2 providers, ≥2 runtimes) | Pending |
| H5 | Security + production readiness | Pending |

**H1 demo checklist**

1. `make serve` → `GET /api/v1/health` returns `status: ok` and current `seq`  
2. `POST /api/v1/missions` with `requiredCapabilities` creates a mission  
3. `GET /api/v1/events?since=0&format=json` returns monotonic `seq`  
4. `POST /api/v1/missions/:id/cancel` flips state to `cancelled`  
5. `GET /api/v1/registry/providers` lists plugin manifests  

AESP-RI GATE-1…9 remain authoritative for the reference monorepo until Hermes owns those demos.
