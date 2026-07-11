# Hermes Agent OS — full agent capabilities

This document covers the product surfaces added to close gaps vs full agent workspaces (e.g. Odysseus-class features), implemented as **kernel plugins + Host API + Mission Control** under AESP principles.

## Architecture

```
Chat / Research / Missions UI
        │
   Host API /api/v1
        │
   Kernel (route → provider → runtime.agent.loop)
        │
   Tool-calling loop ── toolrouter ──┬── fs / shell / web / memory
                                     ├── MCP client (stdio|http)
                                     └── skills injection
```

## S1 — Agent loop

- Runtime: `runtime.agent.loop` (`plugins/runtimes/agent-loop`)
- Provider tools API on openai-compat (`tools` / `tool_calls`)
- Default routing prefers agent-loop (`route.preferRuntime=runtime.agent.loop`)

## S2 — Workspace tools

| Tool | Purpose |
|------|---------|
| `fs.read` / `fs.write` / `fs.list` | Sandboxed under `HERMES_WORKSPACE` |
| `shell.exec` | bash in workspace (`HERMES_ALLOW_SHELL=0` to disable) |
| `web.fetch` / `web.search` | HTTP + DuckDuckGo / SearXNG (`HERMES_SEARCH_URL`) |
| `memory.search` / `memory.write` | Unified memory |
| `research.outline` | Research structure helper |
| `http.request` | Allowlisted HTTP |
| `echo` / `time.now` | Builtins |

## S3 — MCP client

```http
GET/POST /api/v1/mcp/servers
POST     /api/v1/mcp/servers/{id}/connect
POST     /api/v1/mcp/servers/{id}/disconnect
DELETE   /api/v1/mcp/servers/{id}
```

Tools appear as `mcp.<serverId>.<toolName>`.

Mission Control → **MCP**.

## S4 — Skills + context budget

```http
GET/POST /api/v1/skills
```

Builtins: `coding`, `research`, `ops`.  
Mission label: `skills=coding,research`.  
Compose capped (~4k chars) to limit prompt bloat.

Disk: `HERMES_SKILLS_DIR` or `./skills/*.md`.

## S5 — Chat host

```http
GET/POST /api/v1/chat/sessions
GET      /api/v1/chat/sessions/{id}
POST     /api/v1/chat/sessions/{id}/messages
```

Mission Control → **Chat**. Each message → agent-loop mission.

## S6 — Deep research + scheduler

```http
POST /api/v1/research   { "topic": "…", "preferProvider": "provider.kimchi", "preferModel": "kimi-k2.7" }
GET/POST /api/v1/jobs
POST     /api/v1/jobs/{id}/run
DELETE   /api/v1/jobs/{id}
```

## S7 — HITL approvals

```http
GET  /api/v1/approvals?status=pending
POST /api/v1/approvals/{id}/approve
POST /api/v1/approvals/{id}/deny
```

Dangerous tools (`shell.exec`, `fs.write`, `http.request`) are logged to the approval gate.

## Live with Kimchi

```bash
export KIMCHI_API_KEY='…'
export HERMES_WORKSPACE="$PWD"
make serve

# Agent mission
curl -s -X POST localhost:8080/api/v1/missions \
  -H 'Content-Type: application/json' \
  -d '{
    "goal": "List the top-level files in the workspace and summarize README if present.",
    "requiredCapabilities": ["coding","tools"],
    "preferProvider": "provider.kimchi",
    "preferModel": "kimi-k2.7",
    "labels": {"skills":"coding"}
  }'
```

## Env reference

| Variable | Purpose |
|----------|---------|
| `KIMCHI_API_KEY` | Kimchi inference key |
| `HERMES_WORKSPACE` | FS/shell sandbox root |
| `HERMES_ALLOW_SHELL` | `0` disables shell.exec |
| `HERMES_SEARCH_URL` | SearXNG base URL |
| `HERMES_SKILLS_DIR` | Skills markdown directory |
| `HERMES_DATA_DIR` | Durable JSON store root |
| `HERMES_API_TOKEN` | Host API auth |

## What we deliberately did not fork

Odysseus (AGPL) email/gallery/editor monolith — use **channel plugins** or separate hosts later. Hermes remains Apache-2.0 with AESP contracts.
