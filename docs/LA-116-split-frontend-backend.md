# LA-116: Split Frontend From Backend

## Goal

Run LoopAware as two separately deployable services while keeping cookie-based auth via TAuth:

- **Frontend (web)**: serves **all HTML/CSS/JS** and validates TAuth sessions to gate protected pages.
- **Backend (api)**: serves **API-only** routes (JSON/SSE/CSV) and validates TAuth sessions for authorization.

The split is designed to preserve a **single browser origin** via a reverse proxy so LoopAware does not need
credentialed CORS for session cookies.

## Implementation (current)

LoopAware remains one Go binary (`cmd/server`) with a new `--serve-mode` flag:

- `monolith` (default): serves everything (existing behavior).
- `web`: serves all HTML pages and public JS assets.
- `api`: serves backend endpoints under `/api/*` only.

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
- Public JS assets:
  - `GET /widget.js`
  - `GET /subscribe.js`
  - `GET /pixel.js`
- Public pages:
  - `GET /subscribe-demo`
  - `GET /subscriptions/confirm`
  - `GET /subscriptions/unsubscribe`
- Operator tool pages (require TAuth session; data loaded via API):
  - `GET /app/sites/:id/widget-test`
  - `GET /app/sites/:id/traffic-test`
  - `GET /app/sites/:id/subscribe-test`

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
For the computercat orchestration, `ghttp` routes:

- TAuth:
  - `/tauth.js` -> `la-tauth`
  - `/me` -> `la-tauth`
  - `/auth/*` -> `la-tauth`
- LoopAware backend:
  - `/api/*` -> `loopaware-api`
- LoopAware frontend:
  - everything else -> `loopaware-web`

See `configs/README.md` and the `configs/.env.ghttp*.example` templates for the exact `GHTTP_SERVE_PROXIES` string.

## Rollout

1. Keep existing deployments in `monolith` mode (default) until the proxy routes and two-container deployment are validated.
2. Deploy split mode behind the proxy (web + api) on a staging host (computercat).
3. Migrate remaining backend-served HTML pages if the strict split is desired.
