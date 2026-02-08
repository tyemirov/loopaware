# LA-100: Email subscription capture — implementation plan

## Current state (from code)
- Public submissions: `PublicHandlers.CreateFeedback` accepts `site_id/contact/message`, rate limits per IP, enforces `Site.AllowedOrigin`, persists `model.Feedback`, and broadcasts via SSE.
- Widget: `/widget.js` is served as a static asset from `web/widget.js`; requires `site_id` query/header and fetches placement metadata from `/api/widget-config`.
- Admin/app: site CRUD and listing in `SiteHandlers`, feedback tables on the dashboard, and counts via `SiteStatisticsProvider` (feedback only). Models live in `internal/model`; schema managed by `storage.AutoMigrate`.
- Notifications: optional Pinguin gRPC notifier (`notifications.PinguinNotifier`) dispatches feedback alerts to the site owner email/phone.

## Goals
- Let site owners collect visitor emails for news/updates through an embeddable, customizable form.
- Enforce the same origin/rate-limit protections as feedback, keep data scoped per site, and surface subscribers in the dashboard with counts and export/unsubscribe controls.

## End-to-end design

### Data model & invariants
- Introduce `model.Subscriber` with smart constructors (no zero-but-invalid exports):
  - `ID` (uuid, pk, size 36), `SiteID` (fk to `Site`), `Email` (normalized lowercase, size 320), optional `Name` (size 200), `SourceURL` (normalized referer/origin up to 500), `IP` (size 64), `UserAgent` (size 400), `Status` (`pending`, `confirmed`, `unsubscribed`), `ConsentAt`, `ConfirmedAt`, `UnsubscribedAt`, timestamps.
  - Unique composite index on `(SiteID, Email)`; foreign key to sites.
- AutoMigrate updates in `storage.AutoMigrate` to create the table and indexes. Backfills not needed for existing data.

### Public capture flow
- New endpoint `POST /api/subscriptions` handled by `PublicHandlers`:
  - Payload: `{ "site_id": "...", "email": "...", "name": "...", "source_url": "...", "accept_tos": true }`.
  - Reuse/extend `PublicHandlers` rate limiting; reject if required fields missing or invalid email.
  - Enforce `AllowedOrigin` with the same origin/referrer checks used for feedback.
  - Persist subscriber record with `Status=pending` (or `confirmed` when double opt-in is disabled via config flag).
  - Respond with stable error codes (`missing_fields`, `invalid_email`, `origin_forbidden`, `unknown_site`, `save_failed`, `rate_limited`).
- Optional confirmation flow:
  - Config flag `SUBSCRIPTION_DOUBLE_OPT_IN` (env/flag) toggles whether immediate confirmation is allowed.
  - If enabled, issue a signed token (HMAC with `SESSION_SECRET`) and expose `POST /api/subscriptions/confirm` to flip status to `confirmed`.
  - Tokens scoped to `subscriber_id` + `site_id` with short TTL; stored hash in DB for auditing.
- Unsubscribe endpoint `POST /api/subscriptions/unsubscribe` accepting token/email + site_id; sets `Status=unsubscribed` and timestamps.

### Admin/dashboard surfaces
- Extend `SiteStatisticsProvider` and site responses with `subscriber_count` (count of non-unsubscribed subscribers per site).
- New authenticated endpoints under `/api/sites/:id/subscribers` (owner/admin):
  - `GET` list with pagination/search by email/status, returns `{ subscribers: [...], total: ... }`.
  - `PATCH /:subscriber_id` to update status (unsubscribe/resubscribe) and name.
  - `POST /export` to download CSV of confirmed subscribers.
- Dashboard updates in `web/app/index.html`:
  - Add “Subscribers” card/table (status badge, email, name, timestamps) plus count badge mirroring feedback UI patterns.
  - Add controls to copy the subscribe snippet and to export CSV.

### Embeddable subscribe form
- Serve `/subscribe.js?site_id=...` from `web/subscribe.js` (static asset, ESM-friendly with `@ts-check`).
  - Renders a semantic `<form>` (email + optional name + consent checkbox) styled minimally; inline mode by default, optional floating button variant via query param `mode=bubble`.
  - Customizable via query params or `data-*` attributes: button text, accent color, success/error copy, placeholder text, dark/light theme preference.
  - Posts to `/api/subscriptions` with site_id and source URL; handles duplicate submission responses gracefully.
  - Honors `Origin`/`Referer` validation; fails closed with inline error messages.
- Provide a demo/test page (similar to widget demo) for quick preview.

### Notifications & background
- Reuse `FeedbackNotifier` pattern: on new subscription, emit optional owner notification via Pinguin (email/SMS) summarizing the subscriber email and site.
- Add SSE channel mirroring `FeedbackEventBroadcaster` for subscribers (`/api/sites/subscribers/events`) so the dashboard can update counts in real time; broadcaster lives alongside existing feedback broadcaster.

### Migration & rollout
- Add `.env` sample entries for opt-in flags.
- Update README and dashboard instructions to include the subscribe snippet and API endpoints.
- Keep legacy widget behavior unchanged; new endpoints and assets are additive.

### Testing
- Table-driven Go tests for public subscription routes: required fields, email validation, origin enforcement, rate limits, duplicate handling, status transitions (confirm/unsubscribe), and SSE streaming.
- Integration tests for dashboard APIs (listing/exporting subscribers, counts in `ListSites`).
- Browser/integration test that loads `subscribe.js` in a test page, submits a subscription, and asserts DB state + UI success banner.
