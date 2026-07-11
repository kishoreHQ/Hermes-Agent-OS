# Kimchi provider setup

Hermes talks to [Kimchi Inference](https://docs.kimchi.dev/docs/inference-quickstart) as a normal **OpenAI-compatible** provider — same base URL and models Cursor uses.  
Source docs: [Cursor setup](https://docs.kimchi.dev/docs/cursor) · [Quickstart](https://docs.kimchi.dev/docs/inference-quickstart)

| Setting | Value |
|---------|--------|
| Base URL | `https://llm.kimchi.dev/openai/v1` |
| Plugin id | `provider.kimchi` |
| Default model | `kimi-k2.6` |
| Other models | `minimax-m3`, `nemotron-3-ultra-fp4` |
| API key | [app.kimchi.dev/settings](https://app.kimchi.dev/settings) |

> **Note:** Cursor UI model ids use a `kimchi/` prefix (`kimchi/kimi-k2.6`). The **API** model id is without that prefix (`kimi-k2.6`). Hermes uses the API ids.

## 1. Config without a key (ready for final setup)

Plugin ships at `plugins/providers/kimchi/plugin.yaml` and as a Mission Control template **Kimchi**.

```bash
# From Hermes-Agent-OS root
make build
make serve
```

Then either:

**Mission Control → Providers**

1. Select template **Kimchi** (base URL prefilled).
2. Leave API key empty for now — or paste when ready.
3. **Add provider**.

**Host API**

```bash
curl -s -X POST localhost:8080/api/v1/provider-configs \
  -H 'Content-Type: application/json' \
  -d '{
    "fromTemplate": "kimchi",
    "name": "Kimchi",
    "defaultModel": "kimi-k2.6"
  }'
```

## 2. Add the API key (final live setup)

### Option A — environment (recommended for hermesd)

```bash
export KIMCHI_API_KEY='your-key-from-app.kimchi.dev'
# optional overrides
# export HERMES_KIMCHI_BASE_URL='https://llm.kimchi.dev/openai/v1'
# export HERMES_KIMCHI_MODEL='kimi-k2.6'

make serve
```

`KIMCHI_API_KEY` or `HERMES_KIMCHI_API_KEY` is stored as a **credential handle** for `provider.kimchi` (never logged).

### Option B — Mission Control / Host API

```bash
curl -s -X POST localhost:8080/api/v1/provider-configs \
  -H 'Content-Type: application/json' \
  -d '{
    "fromTemplate": "kimchi",
    "apiKey": "your-key-from-app.kimchi.dev",
    "defaultModel": "kimi-k2.6"
  }'
```

Or paste the key in **Providers → API key** when adding/updating.

### Option C — credentials endpoint only

```bash
curl -s -X POST localhost:8080/api/v1/credentials \
  -H 'Content-Type: application/json' \
  -d '{
    "scope": "provider.kimchi",
    "label": "kimchi",
    "pluginId": "provider.kimchi",
    "secret": "your-key-from-app.kimchi.dev"
  }'
```

## 3. Verify (smoke against Kimchi)

```bash
# 1) List providers / models
curl -s localhost:8080/api/v1/providers/models | jq .

# 2) Prefer Kimchi on a mission
curl -s -X POST localhost:8080/api/v1/missions \
  -H 'Content-Type: application/json' \
  -d '{
    "goal": "Explain what a Kubernetes DaemonSet does in one sentence.",
    "requiredCapabilities": ["coding", "tools"],
    "preferProvider": "provider.kimchi",
    "preferModel": "kimi-k2.6",
    "failover": true
  }' | jq .
```

Direct API check (outside Hermes), matching [Kimchi quickstart](https://docs.kimchi.dev/docs/inference-quickstart):

```bash
export KIMCHI_API_KEY='…'
curl https://llm.kimchi.dev/openai/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $KIMCHI_API_KEY" \
  -d '{
    "model": "kimi-k2.6",
    "messages": [{"role":"user","content":"ping"}]
  }'
```

## 4. Multi-model routing tips

| Role | Model |
|------|--------|
| Planning / deep work | `kimi-k2.6` (260K context) |
| Code gen / debug | `minimax-m3` |
| Fast / cheap | `nemotron-3-ultra-fp4` |

```json
{
  "goal": "…",
  "preferProvider": "provider.kimchi",
  "preferModel": "minimax-m3",
  "failover": true
}
```

## Env reference

| Variable | Purpose |
|----------|---------|
| `KIMCHI_API_KEY` | Official Kimchi key (preferred) |
| `HERMES_KIMCHI_API_KEY` | Alias for Hermes |
| `HERMES_KIMCHI_BASE_URL` | Override base (default `https://llm.kimchi.dev/openai/v1`) |
| `HERMES_KIMCHI_MODEL` | Default model id (default `kimi-k2.6`) |
