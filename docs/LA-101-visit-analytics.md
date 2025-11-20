# LA-101: Visit tracking pixel — implementation plan

## Current state (from code)
- Only metric exposed is `feedback_count` via `SiteStatisticsProvider` and responses from `SiteHandlers.ListSites`.
- No visit/traffic data is stored. The public surface includes `/widget.js` and `/api/feedback`, both enforcing `Site.AllowedOrigin` with origin/referrer checks and per-IP rate limiting.
- Dashboard templates and APIs display feedback counts and messages; no charts or counters for traffic exist.

## Goals
- Provide a lightweight “pixel” to record page visits per site (similar to GA/Facebook pixel) with strict origin enforcement and low overhead.
- Surface visit counts (page views, unique visitors) and time buckets in the dashboard; keep growth bounded with retention/rollups.

## End-to-end design

### Data model & invariants
- Add `model.SiteVisit`:
  - `ID` (uuid), `SiteID` (fk), `OccurredAt` (timestamp, default now UTC), `URL` (up to 500, normalized without query for aggregation), `Path` (up to 300), `VisitorID` (uuid persisted in a first-party cookie/localStorage via the pixel script), `IP` (64), `UserAgent` (400), `Referrer` (500).
  - Indexes on `(SiteID, OccurredAt)`, `(SiteID, VisitorID, date)`, `(SiteID, URL)`. Ensure `VisitorID` is optional (still record if blocked).
- Optional rollup table `model.SiteVisitRollup` for daily aggregates: `SiteID`, `Date` (UTC), `PageViews`, `UniqueVisitors`. Populate via background job to keep queries fast.
- Smart constructors validate non-empty `SiteID`, normalized URLs (strip fragment, truncate/trim), optional visitor id UUID format.

### Collection endpoints/pixel
- Public endpoint `GET /pixel.gif` (or `/api/visits`) handled by `PublicHandlers`:
  - Required params: `site_id`; optional: `url`, `referrer`, `visitor_id`, `ts` (client timestamp), `uid` (session/page id).
  - Enforce `AllowedOrigin` by checking `Origin` or `Referer` prefix; if neither header set, require `url` param for verification.
  - Rate-limit per IP similar to feedback to deter abuse.
  - Persist `SiteVisit` and return a 1×1 transparent GIF (or 204 for JS POST) with cache-busting headers.
  - Response always succeeds with opaque body to avoid leaking existence of sites beyond origin checks (`404 unknown_site` when site_id invalid).
- Pixel script `/pixel.js?site_id=...`:
  - ESM with `@ts-check`; generates/stores `visitor_id` cookie (`_loopaware_vid`) scoped to the site origin.
  - Fires `navigator.sendBeacon` (fallback fetch) to `/api/visits` with payload containing site_id, url, referrer, visitor_id, page_id, client_ts.
  - Debounce duplicate sends (one per page load; optional heartbeat not needed).
  - For no-script environments, provide `<img src=".../pixel.gif?site_id=...&url=...">` snippet in dashboard copy.

### Dashboard/admin surfaces
- Extend site list/response with `visit_counts` summary (today, last 7/30 days) and `unique_visitors` if rollups present.
- New endpoints under `/api/sites/:id/visits` (owner/admin):
  - `GET /stats` with query params `range` (`1d`, `7d`, `30d`, `custom`) returning timeseries buckets (date, page_views, unique_visitors) using rollups; fallback to live aggregation when rollup missing.
  - `GET /pages` to list top pages (URL + views) with limits for dashboard table.
- Dashboard updates:
  - Add “Traffic” card showing sparkline/totals for page views and uniques.
  - Add table for “Top pages” with URL and counts; refresh via API and optionally SSE for live stream.

### Background tasks & retention
- Add background scheduler (similar to `SiteFaviconManager`) to:
  - Roll up previous day visits into `SiteVisitRollup`.
  - Trim raw visit rows older than configurable retention (e.g., 35 days) once rolled up.
- Config flags/env: `VISIT_RETENTION_DAYS` (raw), `VISIT_ROLLUP_DAYS` (period to keep rollups).

### Notifications/streaming
- Optional SSE channel `/api/sites/visits/events` broadcasting recent visits (site_id, occurred_at, url) for real-time dashboards; reuse broadcaster pattern from `FeedbackEventBroadcaster`.
- Owner notifications are not required for visits; keep traffic silent by default.

### Testing
- Go tests for collection endpoint: origin/referrer validation, missing params, rate limits, invalid site, storage writes, and transparent GIF response headers.
- Integration test for pixel.js: load test page with script, assert beacon stored visit with visitor_id cookie persisted, and dedup on subsequent sends.
- Tests for rollup job: create sample visits, run scheduler once, assert rollup rows and raw retention trimming.
- Dashboard API tests for stats/top-pages responses and authorization (owner/admin vs. others).

### Migration & rollout
- Add models to `storage.AutoMigrate` with indexes.
- Document new snippets and APIs in README/dashboard widget card.
- Ensure defaults keep system inert unless `pixel.js` or `pixel.gif` is embedded; no behavior change for existing widget/feedback flows.
