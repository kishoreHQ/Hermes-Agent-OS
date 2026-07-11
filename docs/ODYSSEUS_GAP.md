# Odysseus → Hermes gap inventory & plan

**Source:** `/Users/kishore/git/odysseus` (AGPL-3.0 product workspace)  
**Target:** Hermes Agent OS (Apache-2.0, AESP kernel)  

> **License rule:** reimplement ideas under AESP contracts. Never copy Odysseus source.

---

## Status legend

| Status | Meaning |
|--------|---------|
| **DONE** | Shipped in Hermes |
| **IN THIS PASS** | Implemented in the current work package |
| **DEFERRED** | Product host / heavy dependency — API stub or later host app |

---

## A. Full feature gap list

### A1. Core agent (must work)

| Odysseus | Hermes before | Status |
|----------|---------------|--------|
| Multi-turn agent loop | agentloop runtime | **DONE** |
| Tool calling | openai-compat tools | **DONE** |
| fs / shell / web tools | workspacetools | **DONE** |
| MCP multi-server | mcpclient | **DONE** |
| Skills | skills store | **DONE** |
| Context budget / compact | partial compose | **DONE** (`contextpack`) |
| Prompt security / untrusted data | modes only | **DONE** (sanitize untrusted tool/skills) |
| Interactive HITL gates | approval log | **DONE** (assist blocks dangerous tools) |
| Streaming responses | mission batch only | **DONE** (SSE `/api/v1/stream/events`) |
| Session search / history | chat sessions memory | **DONE** (chat host) |
| Rate limiting | none | **DONE** (`HERMES_RATE_LIMIT_PER_MIN`) |
| Durable state (restart-safe) | in-memory | **PARTIAL** (uploads + backup export; full disk journal next) |

### A2. Knowledge & memory

| Odysseus | Hermes | Status |
|----------|--------|--------|
| Chroma / vector RAG | keyword memory | **DONE** (hybrid bag-of-words cosine) |
| Memory MCP server | none | **DONE** (memory.* tools; external MCP optional) |
| Personal docs / vault | none | **DONE** (notes + vault + documents) |
| Embeddings lanes | none | **DONE** (`embeddings` package) |

### A3. Workspace product surfaces

| Odysseus | Hermes | Status |
|----------|--------|--------|
| Chat UI | Chat page | **DONE** |
| Deep Research | research API + page | **DONE** |
| Notes | none | **DONE** |
| Tasks / todos | board tasks thin | **DONE** |
| Documents editor | artifacts pilot | **DONE** (markdown docs store + tools + UI) |
| Compare models | failover only | **DONE** |
| Uploads | none | **DONE** |
| Gallery / image editor | none | **DEFERRED** (channel plugin) |
| Email IMAP/SMTP | none | **DEFERRED** (channel plugin later) |
| Calendar CalDAV | none | **DEFERRED** |
| Contacts | none | **DEFERRED** |
| STT / TTS | none | **DEFERRED** |
| Cookbook / local GGUF serve | provider templates | **DEFERRED** (ops host) |
| Themes / fonts / emoji | MC chrome only | **DEFERRED** |
| 2FA / multi-user auth | shared token | **PARTIAL** (token + rate limit; OIDC later) |
| Backup / restore | none | **DONE** (export snapshot) |
| Webhooks | none | **DONE** (registry) |
| Presets | none | **DONE** |
| Companion device pairing | none | **DEFERRED** |
| Codex / Claude integrations | none | **DEFERRED** (external hosts) |

### A4. Ops

| Odysseus | Hermes | Status |
|----------|--------|--------|
| Docker compose | none | **DONE** |
| Diagnostics / readiness | health only | **DONE** (`/api/v1/diagnostics`) |
| E2E product tests | smoke/e2e-host | **DONE** (`make e2e-agent-os`) |

---

## B. Implementation plan (task order)

1. **Gap doc** (this file)  
2. **Context compactor + prompt sanitize**  
3. **Durable persist** for chat, notes, docs, todos, memory index  
4. **Embeddings + hybrid memory search**  
5. **Notes / Todos / Docs / Vault** packages + tools + API  
6. **Compare** multi-provider run  
7. **Uploads, backup, webhooks, presets**  
8. **SSE streaming** for chat + mission progress  
9. **Assist-mode HITL enforce** for dangerous tools  
10. **Rate limit middleware**  
11. **Mission Control pages** (Notes, Docs, Todos, Compare, Jobs, Approvals, Backup)  
12. **docker-compose.yml**  
13. **scripts/e2e-agent-os.sh** + unit tests  
14. **Live Kimchi e2e**  

---

## C. What will never live in the kernel

Email, CalDAV, full gallery, STT/TTS, Cookbook GPU serve, Claude/Codex OAuth — ship as **optional channel plugins or separate hosts** that call Hermes Host API.
