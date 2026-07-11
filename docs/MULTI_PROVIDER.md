# Multi-provider routing & failover

## Behavior

1. **Capability match** remains the primary key (never model-name-only).  
2. Healthy providers are ordered: prefer/require → model match → cost tier → id.  
3. **Prefer local** first; with failover, non-local providers are appended as backups.  
4. On runtime `Complete` failure, Hermes tries the **next provider** in the chain.  
5. Events: `provider.failed`, `provider.failover`, `route.decided` (includes `failoverChain`).

## Choose provider / model

### JSON mission body

```json
{
  "goal": "implement feature X",
  "requiredCapabilities": ["coding", "tools"],
  "preferProvider": "provider.openai.compat",
  "preferModel": "gpt-4.1-mini",
  "providers": ["provider.openai.compat", "provider.example.budget"],
  "failover": true
}
```

| Field | Meaning |
|-------|---------|
| `preferProvider` | Soft prefer (still can failover) |
| `requireProvider` | Hard pin (with `failover:false` only that provider) |
| `preferModel` / `model` | Prefer this model id when discovered |
| `providers` | Allowlist of provider plugin ids |
| `failover` | Default true |

### Labels (same semantics)

| Label | Effect |
|-------|--------|
| `route.preferProvider` | soft prefer |
| `route.requireProvider` | hard pin |
| `route.preferModel` / `route.model` | model id |
| `route.providers` | comma allowlist |
| `route.failover` | `true`/`false` |
| `route.preferLocal` | `true`/`false` |

## Model auto-discovery

```http
GET /api/v1/providers/models
GET /api/v1/providers/{id}/models
```

OpenAI-compatible providers call `GET {baseURL}/models` when healthy. Echo providers return static manifest models.

## UI: add / delete providers

Mission Control → **Providers**:

1. Pick a **template** (OpenAI, Groq, OpenRouter, Ollama, …) or **Custom**.  
2. Edit **base URL** / **default model** if needed.  
3. Paste **API key** (stored as credential handle — never returned).  
4. **Add provider** → registered live for routing/failover.  
5. **Delete** only works for UI-managed providers (bootstrap seeds are protected).

### Host API

```http
GET  /api/v1/provider-templates
GET  /api/v1/provider-configs
POST /api/v1/provider-configs
PUT  /api/v1/provider-configs/{id}
DELETE /api/v1/provider-configs/{id}
```

```bash
# Add OpenAI from template
curl -s -X POST localhost:8080/api/v1/provider-configs \
  -H 'Content-Type: application/json' \
  -d '{
    "fromTemplate":"openai",
    "apiKey":"sk-…",
    "defaultModel":"gpt-4o-mini"
  }'

# Custom endpoint
curl -s -X POST localhost:8080/api/v1/provider-configs \
  -d '{"fromTemplate":"custom","name":"My GW","baseUrl":"https://gw.example/v1","apiKey":"…"}'

# Delete
curl -s -X DELETE localhost:8080/api/v1/provider-configs/provider.ui.openai
```

Popular templates (OpenAI-compatible base URLs):

| Category | Templates |
|----------|-----------|
| Cloud | **Kimchi**, OpenAI, Groq, Together, Fireworks, DeepSeek, Mistral, xAI, Gemini, Azure OpenAI, Perplexity, Cerebras, SambaNova, Hugging Face, Cloudflare Workers AI, NVIDIA NIM, Moonshot, Qwen |
| Gateway | OpenRouter, Anthropic-via-OpenRouter, LiteLLM |
| Local | Ollama, LM Studio, vLLM, LocalAI, Echo (test) |
| Custom | Any Chat Completions endpoint |

Native Anthropic Messages / Google GenAI plugins are future work — use OpenRouter or OpenAI-compat endpoints today.

## Live multi-provider setup

### Kimchi (recommended first live provider)

See **[KIMCHI.md](./KIMCHI.md)** for full steps (Cursor-compatible base URL).

```bash
export KIMCHI_API_KEY='…'   # from https://app.kimchi.dev/settings
make serve
# provider.kimchi → https://llm.kimchi.dev/openai/v1 · models kimi-k2.6, minimax-m3, …
```

### Generic OpenAI-compat

```bash
export HERMES_OPENAI_BASE_URL='https://api.example.com/v1'
export HERMES_OPENAI_API_KEY='…'
export HERMES_OPENAI_MODEL='…'   # optional default

# Second provider: register another openai-compat plugin with different baseURL
# or POST credentials + prefer different plugin ids
```

Mission Control **Missions** page exposes provider/model dropdowns and a failover checkbox.
