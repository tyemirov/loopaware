# LA-116: Split Frontend From Backend

## Goal

Run LoopAware as a backend API plus a fully independent static frontend, while keeping cookie-based auth via TAuth:

- **Frontend (static)**: serves **all HTML/CSS/JS** from `public/` via `ghttp`.
- **Backend (api)**: serves **API-only** routes (JSON/SSE/CSV) and validates TAuth sessions for authorization.

The split is designed to preserve a **single browser origin** via a reverse proxy so LoopAware does not need
credentialed CORS for session cookies.

## Implementation (current)

LoopAware remains one Go binary (`cmd/server`) with `--serve-mode`:

- `monolith` (default): serves everything (existing behavior).
- `api`: serves backend endpoints under `/api/*` only.

For the computercat orchestration (`docker-compose.computercat.yml`):

- `loopaware-api` runs `--serve-mode=api`
- `loopaware-proxy` (`ghttp`) serves static files from `./public` and reverse-proxies `/api/*` and TAuth paths.

## Route Boundaries

### Frontend (static, served by `ghttp`)

- `GET /` -> redirect to `/login` (static redirect page)
- `GET /login` (public)
- `GET /privacy` (public)
- `GET /app` (requires TAuth session; page hydrates via API)
- Public JS assets:
  - `GET /widget.js`
  - `GET /subscribe.js`
  - `GET /pixel.js`
- Public pages:
  - `GET /subscribe-demo`
  - `GET /subscriptions/confirm`
  - `GET /subscriptions/unsubscribe`
- Operator tool pages (require TAuth session; data loaded via API):
  - `GET /app/widget-test?site_id=...`
  - `GET /app/traffic-test?site_id=...`
  - `GET /app/subscribe-test?site_id=...`

### Backend (`--serve-mode=api`)

All backend routes live under `/api/*`:

- Public endpoints:
  - `POST /api/feedback`
  - `POST /api/subscriptions`
  - `POST /api/subscriptions/confirm`
  - `POST /api/subscriptions/unsubscribe`
  - `GET /api/visits`
  - `GET /api/widget-config`
  - `GET /api/subscriptions/confirm-link`
  - `GET /api/subscriptions/unsubscribe-link`
- Authenticated endpoints (requires TAuth session):
  - `/api/me`, `/api/me/avatar`
  - `/api/sites` (CRUD + SSE + stats + exports)
  - Tool endpoints under `/api/sites/:id/...` (e.g. widget-test feedback, subscribe-test events)

## Reverse Proxy / Single-Origin Model

Split mode assumes a reverse proxy presents one public origin and routes by path prefix.
For the computercat orchestration, `ghttp`:

- TAuth:
  - `/tauth.js` -> `la-tauth`
  - `/me` -> `la-tauth`
  - `/auth/*` -> `la-tauth`
- LoopAware backend:
  - `/api/*` -> `loopaware-api`
- LoopAware frontend:
  - everything else is served from `./public` (no LoopAware web container)

See `configs/README.md` and the `configs/.env.ghttp*.example` templates for the exact `GHTTP_SERVE_PROXIES` string.

## Rollout

1. Keep existing deployments in `monolith` mode (default) until the proxy routes and two-container deployment are validated.
2. Generate the static frontend into `public/` and mount it into `ghttp`.
3. Deploy split mode behind the proxy (static frontend + api) on a staging host (computercat).
