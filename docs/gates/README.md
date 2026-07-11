# Gates

Product acceptance gates for Hermes-Agent-OS.

| Gate | Intent | Status |
|------|--------|--------|
| H0 | Foundation docs + kernel skeleton | **PASS** |
| H1 | Host API parity smoke (`make smoke`) | **PASS** (core Host surface) |
| H2 | Plugin load + execute path (`make smoke`) | **PASS** |
| H3 | Mission Control against Hermes Host API | **PASS** (core host surfaces) |
| H4 | Interchangeability proof (`make prove-h4`) | **PASS** |
| H5 | Security + production readiness | Pending |

**H1 demo checklist**

1. `make serve` → `GET /api/v1/health` returns `status: ok` and current `seq`  
2. `POST /api/v1/missions` with `requiredCapabilities` creates a mission  
3. `GET /api/v1/events?since=0&format=json` returns monotonic `seq`  
4. `POST /api/v1/missions/:id/cancel` flips state to `cancelled`  
5. `GET /api/v1/registry/providers` lists plugin manifests  

**H2 demo checklist**

1. `POST /api/v1/missions` returns `state: succeeded` with `providerId` + `runtimeId`  
2. Events include `route.decided` (capability path, free-local preferred)  
3. `GET /api/v1/memory/search` returns episodic entry for the mission  
4. `GET /api/v1/credentials` returns handles **without** secrets  
5. Registry lists ≥2 providers (echo + budget) and ≥1 runtime  

**H3 demo checklist**

1. `make ui-build && make serve` → browser loads Mission Control from `:8080`  
2. Or `make serve` + `make ui-dev` → UI on `:5173` via `/api` proxy  
3. Launch mission from UI → state succeeded, provider/runtime visible  
4. Fleet shows plugins; Memory / Events / Credentials pages populate  
5. UI source contains **zero** vendor SDK imports  

**H4 demo checklist**

1. `make prove-h4` prints 4× PASS (2 providers × 2 runtimes)  
2. `route.decided` events include `required` capabilities and non-empty `reason`  
3. Excluding free-local routes to budget **without** kernel code change  
4. Preferring `runtime.example.steps` changes harness, not provider  
5. Kernel source not modified between matrix cases — labels only  

AESP-RI GATE-1…9 remain authoritative for the reference monorepo until Hermes owns those demos.
