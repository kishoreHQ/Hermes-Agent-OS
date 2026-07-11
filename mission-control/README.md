# Hermes Mission Control

Operator UI for **Hermes Agent OS**.

## Principles

- Binds **only** to Hermes Host API (`/api/v1/*`)
- Zero vendor SDKs
- Host-neutral: Mission Control is one host among many (INV-11)
- Cherenkov ops aesthetic (cyan ladder on deep blue)

## Pages (H3)

| Route | Host API |
|-------|----------|
| Overview | health, missions, registry counts |
| Missions | list / submit / detail / cancel |
| Fleet | registry providers · runtimes · tools |
| Memory | memory/search |
| Events | events journal |
| Credentials | credential handles only |

Deck features from AESP-RI (connections, boards, routines) stay on the reference monorepo until Hermes Host API grows those surfaces.

## Dev

```bash
# Terminal 1 — kernel
cd .. && make serve

# Terminal 2 — UI (proxies /api → :8080)
npm install
npm run dev
# http://127.0.0.1:5173
```

## Production build (served by hermesd)

```bash
npm run build
# produces mission-control/dist
# hermesd serve discovers dist automatically (or HERMES_UI_DIST=...)
```

From repo root:

```bash
make ui-build
make serve
# open http://127.0.0.1:8080
```
