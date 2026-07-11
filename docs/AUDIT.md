# Hermes Agent OS — Deep Audit (2026-07-11)

Honest assessment against Master Execution Program success criteria and production readiness.

## Executive verdict

| Area | Verdict |
|------|---------|
| Automated proofs (unit, conform, H4/H5) | **PASS** on this machine |
| Architecture fit (provider≠runtime, plugins, capability routing) | **Sound** for a product pilot |
| Security for **internet exposure** | **Not ready** without auth + network controls |
| Live multi-vendor + multi-MCP production | **Not yet** — needs your API URL/key + real MCP process adapters |
| Multi-agent communication | **Basic path works** (A2A → missions; workflow DAG → child missions) |

---

## Security findings (priority)

### Critical (fixed or mitigable)

| Issue | Status |
|-------|--------|
| Host API had **no authentication** | **Mitigated:** optional `HERMES_API_TOKEN` + Bearer / `X-Hermes-Token` |
| Credentials API was list-only (couldn't ingest keys safely) | **Fixed:** `POST /api/v1/credentials` returns **handle only** |
| OpenAI key couldn't be wired without code edit | **Fixed:** `HERMES_OPENAI_BASE_URL` + `HERMES_OPENAI_API_KEY` env; prefer plugin handle over demo token |

### High (document / configure)

| Issue | Risk | Guidance |
|-------|------|----------|
| CORS default `*` | CSRF-ish browser abuse if API is public | Set `HERMES_CORS_ORIGIN` to UI origin in prod |
| WS `InsecureSkipVerify` + open origins | Any site can open WS if no token | Set `HERMES_API_TOKEN` + `HERMES_WS_ORIGINS` |
| All state **in-process memory** | Loss on restart; no multi-node | Expected for pilot; storage plugins later |
| Plugin signatures off by default | Supply-chain | `HERMES_REQUIRE_SIGNED_PLUGINS=1` + HMAC key |
| `spec.apiKey` allowed on openai-compat | Secret in manifest | Prefer broker handles only |

### Medium

| Issue | Notes |
|-------|--------|
| Scopes recorded but not enforced on every tool call | Mode gates execute/HITL; fine-grained tool RBAC incomplete |
| Assist HITL has no resolve API beyond state | Approval *decision* endpoint still thin vs AESP-RI |
| Demo token `hermes-demo-token` still used when no real key | Echo path only; live provider uses env/operator handle |

---

## Architecture findings

### What is solid

- Clear **protocol vs product** split (AESP vs Hermes)
- **INV-01…03** held in kernel (providers ≠ runtimes; plugins; capability routing)
- Host-neutral `/api/v1` envelope + journal `seq`
- Deck, tools, plans, workflow, KG, deploy, rem, docgen as modules
- Conformance catalog with **executable** checks

### What is still shallow (honest)

| Capability | Reality |
|------------|---------|
| MCP | **Bridge** over Hermes tools — not full MCP JSON-RPC multi-server process host |
| A2A | Local peers + **real missions**; not remote HTTP peer protocol |
| Workflow | In-process DAG of missions; not durable Temporal-class |
| Runtimes | Echo/steps harnesses — not Claude Code / PTY production adapters |
| Multi-tenant | Types exist; no isolation enforcement |
| Persistence | Memory stores only |

---

## Missing vs original “best possible OS” vision

1. **Real MCP servers** as subprocesses (stdio/SSE) with multi-server registry  
2. **Durable workflow** + crash recovery  
3. **Remote A2A** protocol  
4. **Named CLI runtimes** (sandbox-agent / PTY)  
5. **AuthN/Z** beyond single shared token (OIDC, per-principal)  
6. **Disk-backed** memory/artifacts/credentials  
7. **Rate limits / audit log retention**  
8. Mission Control UI for new platform routes (plans/workflow/KG/deploy) — API exists, UI partial  

---

## End-to-end tests (automated)

```bash
make test
make conform
make conform-full
make prove-h4
make prove-h5
make e2e          # scripts/e2e-host.sh — full HTTP surface
```

`e2e-host.sh` covers: health, mission, events, tools, agents, plan+workflow, **two A2A peers**, knowledge, artifacts, credentials (no secret echo), board/routine, session.

---

## Ready for your OpenAPI URL + API key

When you provide:

1. **Base URL** (OpenAI-compatible chat completions root, e.g. `https://api.example.com/v1`)  
2. **API key**  
3. Optional **model id**  
4. Optional **MCP server** commands/URLs  

Configure **without committing secrets**:

```bash
export HERMES_API_TOKEN='dev-local-token'          # recommended
export HERMES_OPENAI_BASE_URL='https://…/v1'
export HERMES_OPENAI_API_KEY='…'                   # never commit
export HERMES_OPENAI_MODEL='…'

make serve
# or
HERMES_OPENAI_BASE_URL=… HERMES_OPENAI_API_KEY=… ./bin/hermesd serve :8080

# force live provider
curl -s -X POST localhost:8080/api/v1/missions \
  -H "Authorization: Bearer $HERMES_API_TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{
    "goal":"live e2e",
    "requiredCapabilities":["coding","tools"],
    "labels":{
      "route.preferProvider":"provider.openai.compat",
      "route.preferLocal":"false"
    }
  }'
```

Or POST the key as a handle:

```bash
curl -s -X POST localhost:8080/api/v1/credentials \
  -H "Authorization: Bearer $HERMES_API_TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"pluginId":"provider.openai.compat","label":"live","secret":"YOUR_KEY"}'
# response: { "data": { "handle": "cred_…", … } }  — secret never returned
```

**MCP multi-server / deep multi-agent:** next increment after live inference works — register MCP process plugins and expand A2A/workflow. The host paths for multi-agent missions + two peers already exercise the kernel path in `e2e-host.sh`.

---

## Challenge checklist (self-grill)

| Question | Answer |
|----------|--------|
| Did we skip stages? | Catalog “implemented” includes pilot-depth modules; not all are production-grade |
| Security theater? | Modes/signing exist; **network exposure was open** until token auth |
| Vendor lock-in? | No — openai-compat is wire-format neutral |
| Replay? | Routing + security events journaled with seq |
| Multi-agent? | Yes at mission/A2A/workflow level; not distributed swarm |
| Ready for your key? | **Yes** for OpenAI-compatible live path after env config |

---

*Audit performed against tree at H0–H5 + H1.1 + H3.1 + catalog closures + hardening patches in this session.*
