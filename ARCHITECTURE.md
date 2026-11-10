# Architecture

## Overview

LoopAware is split into two parts:

- **Backend API**: `cmd/server` serves protected `/api/*` endpoints and public collection routes under `/public/*` and `/sentry/*`.
- **Static frontend**: `web/` holds handwritten LoopAware-owned HTML/JS/CSS served by a CDN or reverse proxy (no generator).
  The public crawl surface uses `https://loopaware.mprlab.com/` as the canonical root and
  `https://loopaware.mprlab.com/resources/` as the crawlable resource hub.

Deployments can preserve a single browser origin (for example via `ghttp`) so TAuth cookies remain same-origin. Otherwise
configure CORS on the API to allow the frontend origin.

## Frontend Dependency Delivery

The static frontend has a strict delivery contract for browser dependencies:

1. `web/` stores only LoopAware-authored frontend assets and markup. Do not commit vendored copies of third-party
   JavaScript or CSS under `web/` (including `web/vendor/`).
2. Every third-party browser dependency must be consumed from a CDN URL at the point of use. A browser dependency that
   is not delivered by CDN is forbidden.
3. Local fallbacks, mirrored bundles, and checked-in vendor copies are forbidden because they bypass the CDN-only
   delivery contract.
4. MPR UI uses literal `@latest` CDN URLs. Other third-party dependencies use pinned versions and SHA-384 integrity values.
5. If a dependency cannot use CDN delivery, change the architecture decision before you add it.
6. Page markup declares the shared CDN assets. `web/runtime-env.js` loads service origins from `web/config.yml`.
   `web/config-ui.yaml` supplies browser authentication settings.

## Components

- **Auth**: the frontend delegates browser sign-in scaffolding to `mpr-ui`, TAuth issues the session cookie, and `/api/*` validates it with TAuth's verifier.
- **Dashboard**: a static HTML/JS application backed by JSON APIs and server-sent events (SSE) for live updates.
- **Public assets**: `GET /widget.js`, `GET /subscribe.js`, and `GET /pixel.js` are served from the static frontend and
  call their public collection endpoints at runtime. The visit pixel returns a GIF.
- **Storage**: `internal/storage` opens the configured DB driver and runs migrations on startup. `internal/model` defines
  domain structs and smart constructors.
- **Notifications**: feedback and subscription notifications are sent to the Pingwin gRPC service. Calls include the
  configured tenant metadata and shared auth token.
- **Backend runtime config**: `cmd/server` selects one YAML file with `--config`. `internal/serverconfig` loads it through
  `github.com/tyemirov/utils/runtimeconfig`, expands shell placeholders during parse, validates the populated contract,
  and returns typed values to the server. LoopAware does not read runtime settings from Viper, config-value flags, or
  direct environment lookups after that boundary.

## Auth Bootstrap Constraints

The static frontend depends on a strict auth bootstrap order. Treat the following as architectural constraints, not
implementation details:

1. LoopAware pages render the shared shell with `<mpr-header data-config-url="/config-ui.yaml">` and the shared
   `mpr-ui-config.js`/bundle-marker path. LoopAware pages do not load provider-specific browser auth scripts directly.
2. `/config-ui.yaml` is the browser-facing auth configuration surface for `mpr-ui`. Page markup must not hand-wire a
   second auth path through `tauth.js` or manual helper globals.
3. LoopAware frontend code can react to public `mpr-ui:auth:*` lifecycle events for redirects, overlays, and product state.
   It must not inspect provider-specific auth controls inside the shared shell.
4. Slotted `mpr-user` elements on static pages must include all required auth attributes in the page markup itself:
   `tauth-tenant-id`, `logout-url`, and `logout-label`. Do not rely on a parent component to backfill required values
   after the child has already connected.
5. Public static pages (`/login`, `/privacy`) must boot without logging `mpr-ui.tenant_id_required`, because that error
   means the user menu connected before receiving required auth configuration.
6. Browser regression tests must cover both runtime-origin wiring and the “no tenant bootstrap error on public pages”
   contract before auth-related frontend changes can land.

## Key flows

### Feedback

1. The widget (`/widget.js`) posts JSON feedback to `POST /public/feedback`.
2. The server validates the request origin against the site’s `allowed_origin` list (space/comma-separated values).
3. Feedback is persisted and broadcast over SSE (`GET /api/sites/feedback/events`) for dashboard updates.

### Subscriptions (double opt-in)

1. The subscribe form (`/subscribe.js`) posts JSON to `POST /public/subscriptions`, which records a pending subscriber.
2. A confirmation email is sent containing `GET /subscriptions/confirm?token=...`.
3. Visiting the link confirms the subscriber and (when enabled) notifies the site owner.
4. Confirmation and unsubscribe mutations are available only through signed email-link tokens. The static confirmation and
   unsubscribe pages call `GET /public/subscriptions/confirm-link?token=...` and
   `GET /public/subscriptions/unsubscribe-link?token=...`. There is no email-address-based public mutation or status API.

### Traffic

1. The pixel (`/pixel.js`) sends beacons to `GET /public/visits` with a stable visitor ID, current URL, browser timezone, browser locale, viewport, and screen resolution.
2. The server stores visits (including bot classification metadata, browser location hints, and supported edge geo headers) and serves aggregated stats to the dashboard (`GET /api/sites/:id/visits/stats`).
3. Selected-site dashboard traffic endpoints accept `interval=all|1day|7days|30days`. The dashboard defaults to `all`.
4. Daily trend data is available at `GET /api/sites/:id/visits/trend` (default 7 days. Optional `days` query parameter when no `interval` is supplied).
5. Attribution breakdown data is available at `GET /api/sites/:id/visits/attribution` (default top 10 values per dimension. Optional `limit` query parameter up to 50).
6. Engagement data is available at `GET /api/sites/:id/visits/engagement` (default 30 days. Optional `days` query parameter when no `interval` is supplied).
7. Device breakdown data is available at `GET /api/sites/:id/visits/devices`.
8. Inferred visitor locations are available at `GET /api/sites/:id/visits/locations`. Edge geo is preferred when present, with timezone, locale, network, and unknown fallbacks plus confidence metadata.
9. Operators can download filtered selected-site traffic rows from `GET /api/sites/:id/visits/export`.

## Current operator references

See [setup](docs/setup.md), [HTTP API](docs/api.md), and [integrations](docs/integrations.md) for current operational contracts.
Daily counts use a separate immutable traffic profile with a bounded retention period.
See [aggregate analytics](.mprlab/CHILD-AUDIENCE-ANALYTICS.md) for its data and log restrictions.

## Historical implementation notes

These entries describe earlier changes.
Use the current guides above for setup and validation commands.

## LA-60: Unified Owner Assignment

- All authenticated dashboard roles can now create sites with any valid owner email address. The system continues to
  record the authenticated creator in `creator_email`.
- No schema changes are required. Existing sites already contain the necessary fields. Verify that historical records
  have `creator_email` populated before relying on creator-based scoping.

## LA-61 & LA-62: Favicon Task Scheduler and Notifications

- The server now launches a background task queue (`SiteFaviconManager`) that refreshes favicons at most every
  24 hours and immediately after site creation or updates. Make sure process supervisors keep the binary alive so the
  scheduler can execute.
- Reverse proxies terminating `/api/sites/favicons/events` must permit streaming responses. Do not buffer the SSE
  connection or it will delay dashboard updates.

## LA-63 & LA-64: Privacy Policy and Sitemap

- The static frontend now serves a privacy policy at `/privacy`. Update any CDN caches so the new route is immediately
  available to end users and compliance tooling.
- If you publish a sitemap at `/sitemap.xml`, keep the URLs aligned with the canonical origin before submitting the
  sitemap to search engines.

## LA-77: Session Timeout Prompt

- The dashboard shows an inactivity prompt after the configured delay (default 60 seconds).
  Without a response, it signs the user out at the configured timeout (default 120 seconds). Confirm and dismiss buttons keep the landing-page
  redirect behavior unchanged.
- The prompt applies the selected light or dark theme automatically. Make sure session lifetime settings on the server
  exceed the 120-second inactivity window to preserve a predictable experience.
- Current browser tests use Playwright. Failure artifacts are stored under `tests/test-results/`.
  See [development](docs/development.md) for the test commands.

## LA-80: Widget Placement Controls

- Sites now persist widget placement metadata (`widget_bubble_side`, `widget_bubble_bottom_offset_px`). Auto-migrate
  the database so these columns default to the legacy right-aligned, 16px offset configuration.
- The dashboard exposes placement controls beside the widget snippet. Operators can choose left or right alignment and
  a bottom offset between 0 and 240 pixels. Existing sites automatically adopt the previous layout until adjusted.
- The embeddable widget consumes the stored placement values. Headless integration tests now assert bubble alignment on
  the chosen edge and run faster thanks to a 2-second auto-hide timer.
