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

## Live multi-provider setup

```bash
export HERMES_OPENAI_BASE_URL='https://api.example.com/v1'
export HERMES_OPENAI_API_KEY='…'
export HERMES_OPENAI_MODEL='…'   # optional default

# Second provider: register another openai-compat plugin with different baseURL
# or POST credentials + prefer different plugin ids
```

Mission Control **Missions** page exposes provider/model dropdowns and a failover checkbox.
