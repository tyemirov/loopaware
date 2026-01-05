# Architecture

## Overview

LoopAware is a single Go web service (`cmd/server`) that serves both the authenticated dashboard (`/app`) and the public
collection endpoints and assets (widgets, pixels, and confirmation pages). It uses Gin for routing and Gorm for storage,
with SQLite as the default driver.

## Components

- **Auth**: `/login` and all `/api/*` endpoints are secured by TAuth, which issues the `app_session` JWT cookie via Google Identity Services.
- **Dashboard**: a server-rendered HTML application backed by JSON APIs and server-sent events (SSE) for live updates.
- **Public assets**: `GET /widget.js`, `GET /subscribe.js`, and `GET /pixel.js` are generated JavaScript payloads that
  embed the selected `site_id` and call the public JSON endpoints.
- **Storage**: `internal/storage` opens the configured DB driver and runs migrations on startup; `internal/model` defines
  domain structs and smart constructors.
- **Notifications**: feedback and subscription notifications are sent to the Pinguin gRPC service; calls include the
  configured tenant metadata and shared auth token.

## Key flows

### Feedback

1. The widget (`/widget.js`) posts JSON feedback to `POST /api/feedback`.
2. The server validates the request origin against the site’s `allowed_origin` list (space/comma-separated values).
3. Feedback is persisted and broadcast over SSE (`GET /api/sites/feedback/events`) for dashboard updates.

### Subscriptions (double opt-in)

1. The subscribe form (`/subscribe.js`) posts JSON to `POST /api/subscriptions`, which records a pending subscriber.
2. A confirmation email is sent containing `GET /subscriptions/confirm?token=...`.
3. Visiting the link confirms the subscriber and (when enabled) notifies the site owner.
4. Unsubscribe is available either via the origin-validated JSON endpoint (`POST /api/subscriptions/unsubscribe`) or the
   token-based link (`GET /subscriptions/unsubscribe?token=...`) from the confirmation UI.

### Traffic

1. The pixel (`/pixel.js`) sends beacons to `GET /api/visits` with a stable visitor ID and the current URL.
2. The server stores visits and serves aggregated stats to the dashboard (`GET /api/sites/:id/visits/stats`).

## Migrations

## LA-60: Unified Owner Assignment

- All authenticated dashboard roles can now create sites with any valid owner email address; the system continues to
  record the authenticated creator in `creator_email`.
- No schema changes are required. Existing sites already contain the necessary fields; verify that historical records
  have `creator_email` populated before relying on creator-based scoping.

## LA-61 & LA-62: Favicon Task Scheduler and Notifications

- The server now launches a background task queue (`SiteFaviconManager`) that refreshes favicons at most every
  24 hours and immediately after site creation or updates. Ensure process supervisors keep the binary alive so the
  scheduler can execute.
- Reverse proxies terminating `/api/sites/favicons/events` must permit streaming responses; do not buffer the SSE
  connection or it will delay dashboard updates.

## LA-63 & LA-64: Privacy Policy and Sitemap

- The server now serves a static privacy policy at `/privacy`. Update any CDN caches so the new route is immediately
  available to end users and compliance tooling.
- `/sitemap.xml` returns an XML sitemap listing `/login` and `/privacy`. Configure `PUBLIC_BASE_URL` so the generated
  URLs point at the canonical origin before submitting the sitemap to search engines.

## LA-77: Session Timeout Prompt

- The dashboard surfaces an inactivity prompt after 60 seconds without user input and signs the user out at 120 seconds
  if no action is taken. Confirm and dismiss buttons touch the existing logout endpoint so the landing-page redirect
  remains unchanged.
- The prompt applies the selected light or dark theme automatically. Ensure session lifetime settings on the server
  exceed the 120-second inactivity window to preserve a predictable experience.
- Browser automation tests for this feature now rely on go-rod and store screenshots under `tests/<date>/<testname>/`; keep the directory if you need evidence of completed inactivity flows.

## LA-80: Widget Placement Controls

- Sites now persist widget placement metadata (`widget_bubble_side`, `widget_bubble_bottom_offset_px`). Auto-migrate
  the database so these columns default to the legacy right-aligned, 16px offset configuration.
- The dashboard exposes placement controls beside the widget snippet. Operators can choose left or right alignment and
  a bottom offset between 0 and 240 pixels. Existing sites automatically adopt the previous layout until adjusted.
- The embeddable widget consumes the stored placement values; headless integration tests now assert bubble alignment on
  the chosen edge and run faster thanks to a 2-second auto-hide timer.
