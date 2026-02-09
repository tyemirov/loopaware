# LA-116: Split Frontend From Backend

## Goal

Run LoopAware as a backend API plus a fully independent static frontend, while keeping cookie-based auth via TAuth:

- **Frontend (static)**: serves **all HTML/CSS/JS** from `web/` via a CDN or `ghttp`.
- **Backend (api)**: serves **API-only** routes (JSON/SSE/CSV) and validates TAuth sessions for authorization.

The split is designed to preserve a **single browser origin** via a reverse proxy so LoopAware does not need
credentialed CORS for session cookies.

## Implementation (current)

LoopAware now runs with a dedicated API backend and a fully static frontend:

- **Backend**: `cmd/server` is API-only (JSON/SSE/CSV + public collection endpoints).
- **Frontend**: static files live in `web/` and are served by a CDN or `ghttp` (no generator).

For the computercat orchestration (`docker-compose.computercat.yml`):

- `loopaware-api` runs the API-only `cmd/server`.
- `loopaware-proxy` (`ghttp`) serves static files from `./web` and reverse-proxies `/api/*` and TAuth paths.

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

### Backend (API-only)

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
  - everything else is served from `./web` (no LoopAware web container)

See `configs/README.md` and the `configs/.env.ghttp*.example` templates for the exact `GHTTP_SERVE_PROXIES` string.

## Multi-Origin GitHub Pages Model

If `loopaware.mprlab.com` is served directly from GitHub Pages (no reverse proxy), LoopAware runs as a multi-origin
deployment:

- Frontend: `https://loopaware.mprlab.com` (static Pages/CDN)
- API: `https://loopaware-api.mprlab.com`
- TAuth: `https://tauth-api.mprlab.com`

In this mode:

- The static pages default to the API/TAuth origins above when `window.location.hostname === "loopaware.mprlab.com"`.
- You can override per request using `?api_origin=...&tauth_origin=...` (primarily for local/dev diagnostics).
- The widget/subscription/pixel snippets include `api_origin` so they can call the API from customer sites.
- Authenticated API calls rely on credentialed CORS:
  - Set `PUBLIC_BASE_URL=https://loopaware.mprlab.com` on the API service.
  - Ensure the API CORS config allows that origin with credentials enabled (LoopAware's `/api/*` group is configured
    this way).

## Rollout

1. Publish the static frontend in `web/` (GitHub Pages, CDN, or `ghttp`).
2. Configure the API backend with `PUBLIC_BASE_URL` pointing at the frontend origin.
3. Deploy the API-only backend and validate reverse-proxy routing (if using single-origin mode).
