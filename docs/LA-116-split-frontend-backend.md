# LA-116: Split Frontend From Backend

## Goal

Run LoopAware as two separately deployable services while keeping cookie-based auth via TAuth:

- **Frontend (web)**: serves the core HTML UI and validates TAuth sessions to gate protected pages.
- **Backend (api)**: serves LoopAware APIs and public JS assets and validates TAuth sessions for authorization.

The split is designed to preserve a **single browser origin** via a reverse proxy so LoopAware does not need
credentialed CORS for session cookies.

## Implementation (current)

LoopAware remains one Go binary (`cmd/server`) with a new `--serve-mode` flag:

- `monolith` (default): serves everything (existing behavior).
- `web`: serves the core UI pages (`/`, `/login`, `/privacy`, `/sitemap.xml`, `/app`).
- `api`: serves backend routes (public JS assets, public collection endpoints, authenticated APIs, and DB-backed helper pages).

The docker orchestration (`docker-compose.computercat.yml`) runs two containers from the same image:

- `loopaware-web` runs `--serve-mode=web`
- `loopaware-api` runs `--serve-mode=api`

## Route Boundaries

### Frontend (`--serve-mode=web`)

- `GET /` -> redirect to `/login`
- `GET /login`
- `GET /privacy`
- `GET /sitemap.xml`
- `GET /app` (requires TAuth session)

### Backend (`--serve-mode=api`)

- Authenticated API: `/api/*` (requires TAuth session)
- Public collection endpoints:
  - `POST /api/feedback`
  - `POST /api/subscriptions`
  - `POST /api/subscriptions/confirm`
  - `POST /api/subscriptions/unsubscribe`
  - `GET /api/visits`
- Public JS assets:
  - `GET /widget.js`
  - `GET /subscribe.js`
  - `GET /pixel.js`
- DB-backed helper pages (migration staging):
  - `/app/sites/*` (widget/traffic/subscribe test pages)
  - `/subscriptions/*` (confirm/unsubscribe pages)
  - `/subscribe-demo`

Follow-up work can move the remaining DB-backed HTML pages into the frontend service (or re-home them under a dedicated backend namespace)
to make the backend strictly API-only.

## Reverse Proxy / Single-Origin Model

Split mode assumes a reverse proxy presents one public origin and routes by path prefix.
For the computercat orchestration, `ghttp` routes:

- TAuth:
  - `/tauth.js` -> `la-tauth`
  - `/me` -> `la-tauth`
  - `/auth/*` -> `la-tauth`
- LoopAware backend:
  - `/api/*` -> `loopaware-api`
  - `/widget.js`, `/subscribe.js`, `/pixel.js` -> `loopaware-api`
  - `/app/sites/*` -> `loopaware-api`
  - `/subscriptions/*`, `/subscribe-demo` -> `loopaware-api`
- LoopAware frontend:
  - everything else -> `loopaware-web`

See `configs/README.md` and the `configs/.env.ghttp*.example` templates for the exact `GHTTP_SERVE_PROXIES` string.

## Rollout

1. Keep existing deployments in `monolith` mode (default) until the proxy routes and two-container deployment are validated.
2. Deploy split mode behind the proxy (web + api) on a staging host (computercat).
3. Migrate remaining backend-served HTML pages if the strict split is desired.

