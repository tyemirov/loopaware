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
