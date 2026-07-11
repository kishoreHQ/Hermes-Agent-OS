# Security

Enforced controls (H5):

| Control | Mechanism |
|---------|-----------|
| Agent modes | `full` · `assist` · `observe` (mission `mode` or `security.mode`) |
| External actions | Assist requires approval (`security.externalAction=true`) |
| Scopes | Mode-derived grants (memory/provider/runtime/events) |
| Sandbox tiers | Policy `minSandboxTier` vs runtime descriptor |
| Credentials | Unified broker; Host lists **handles only** |
| Memory trust | Trust labels on writes |
| Plugin signing | HMAC-SHA256 (`hermes.signature` label) |

## Environment

| Variable | Purpose |
|----------|---------|
| `HERMES_PLUGIN_HMAC_KEY` | Shared secret for manifest HMAC |
| `HERMES_REQUIRE_SIGNED_PLUGINS` | `1`/`true` — reject unsigned plugins on load |
| `HERMES_UI_DIST` | Mission Control SPA path override |

## API

- `GET /api/v1/security/posture`  
- `GET /api/v1/policies`  
- Mission journal: `security.evaluated`

## Signing a plugin (dev)

```bash
# Pseudo: sign with hermes tooling or:
# signature = HMAC-SHA256(key, "apiVersion|kind|id|version")
# put under labels.hermes.signature in plugin.yaml
```

See ADR-0011.
