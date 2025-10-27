# Migrations

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
