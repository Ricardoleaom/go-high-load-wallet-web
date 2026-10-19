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
| `loadlab/` | the load generator: an open-model Go service, behind its own HTTP API |
| `compose.yaml` | builds and runs `web` and `loadlab` beside the backend's own stack |

## Develop

The backend must be running first (`POSTGRES_HOST_PORT=55432 make up` in
go-high-load-wallet), with the console's origins allowed (its Compose
file does that by default).

```sh
cd web
npm ci
npm run dev        # http://localhost:5173, sign in as operator / operator
npm test
```

```sh
cd loadlab
go test -race ./...
go run ./cmd/loadlab   # :8090, points at the backend's local defaults
```

## Run as containers

With the backend's own stack up:

```sh
docker compose up --build
```

This builds and starts:

- `web`, the console served by nginx, at <http://localhost:8088>;
- `loadlab`, on the backend's Docker network so it reaches the API and
  Keycloak directly, published at <http://localhost:8090>.

`docker compose down` stops both; it never touches the backend's own
stack.
