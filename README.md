# go-high-load-wallet console

An operator console for [go-high-load-wallet](https://github.com/Ricardoleaom/go-high-load-wallet):
sign in through the wallet's Keycloak, watch wallets, transactions,
ledgers and reconciliation, follow the service live, and put it under
real load from a server-side load generator.

Work in progress; see the sections below as they land.

## Layout

| Path | What |
| --- | --- |
| `web/` | the single-page console: React, TypeScript, Vite |
| `web/openapi.yaml` | the backend's API contract, copied at a recorded commit by `web/scripts/sync-openapi.sh` |

## Develop

The backend must be running (`make up` in go-high-load-wallet), with the
console's origins allowed (its Compose file does that).

```sh
cd web
npm ci
npm run dev        # http://localhost:5173, sign in as operator / operator
npm test
```
