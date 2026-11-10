# HTTP API reference

[Project overview](../README.md)

## REST API

In this guide, a site manager is an owner, creator, or global administrator.
The `manager` role in the route table refers to that access scope.

Authenticated management endpoints live under `/api` and require the configured TAuth session cookie.
Browser collection enforces per-site origin rules without an operator session.
Registered mobile feedback rejects browser origin headers and validates the mobile app registration.
Signed subscription links authorize their operation through the token.
JSON responses include Unix timestamps in seconds.

Detailed sites use `interval=1day` for the trailing 24 hours.
Aggregate sites use `interval=1day` for the current UTC day. `interval=7days` and `interval=30days` use UTC day windows.
Each window includes the current UTC day and its preceding days.

`interval=all` reports all recorded human visits for detailed sites.
For aggregate sites, it reports all retained accepted-request counts.

| Method  | Path                                  | Role        | Description                                                                                             |
|---------|---------------------------------------|-------------|---------------------------------------------------------------------------------------------------------|
| `GET`   | `/api/me`                             | any         | Current account metadata (email, name, `role`, `avatar.url`)                                            |
| `GET`   | `/api/sites`                          | any         | Sites visible to the caller. Each row includes `access_role` (`admin` or `team_member`)                  |
| `POST`  | `/api/sites`                          | any         | Create a site (`name` and `allowed_origin` required. `owner_email` defaults to the caller)                                        |
| `PATCH` | `/api/sites/:id`                      | manager | Update name, origins, widget settings, or ownership                                                       |
| `DELETE`| `/api/sites/:id`                      | manager | Delete a site                                                                                            |
| `GET`   | `/api/sites/:id/team`                 | manager | List per-site team member email assignments                                                             |
| `POST`  | `/api/sites/:id/team`                 | manager | Add a per-site team member by email                                                                     |
| `DELETE`| `/api/sites/:id/team/:member_id`      | manager | Remove a per-site team member assignment                                                                |
| `GET`   | `/api/sites/:id/mobile-apps`          | manager/team member | List native mobile apps registered for feedback submissions                                    |
| `POST`  | `/api/sites/:id/mobile-apps`          | manager | Register a native mobile app for mobile feedback submissions                                            |
| `GET`   | `/api/sites/:id/messages`             | manager/team member | List feedback messages (newest first)                                                         |
| `GET`   | `/api/sites/:id/subscribers`          | manager/team member | List subscribers for a site                                                                   |
| `GET`   | `/api/sites/:id/subscribers/export`   | manager/team member | Download subscribers as CSV                                                                   |
| `PATCH` | `/api/sites/:id/subscribers/:subscriber_id` | manager | Update a subscriber’s status (confirm or unsubscribe)                                             |
| `DELETE`| `/api/sites/:id/subscribers/:subscriber_id` | manager | Delete a subscriber                                                                                |
| `GET`   | `/api/sites/:id/visits/stats`         | manager/team member | Aggregate visit and unique visitor counts plus recent visits and top pages (optional `interval=all\|1day\|7days\|30days`) |
| `GET`   | `/api/sites/:id/visits/export`        | manager/team member | Download traffic visits as CSV (optional `interval=all\|1day\|7days\|30days`)                      |
| `GET`   | `/api/sites/:id/sentry/issues`        | manager/team member | List grouped developer error issues for a site                                                |
| `GET`   | `/api/sites/:id/sentry/issues/:issue_id` | manager/team member | Inspect latest and recent LA Sentry error occurrences                                      |
| `PATCH` | `/api/sites/:id/sentry/issues/:issue_id` | manager | Update issue status (`unresolved`, `resolved`, or `ignored`)                                         |
| `POST`  | `/api/sites/:id/sentry/token`         | manager | Rotate and reveal a per-site LA Sentry ingest token                                                     |
| `GET`   | `/api/sites/:id/visits/trend`         | manager/team member | Daily visit trend (default 7 days, optional `days` query param up to 30, or `interval=all\|1day\|7days\|30days`) |
| `GET`   | `/api/sites/:id/visits/attribution`   | manager/team member | Source/medium/campaign attribution breakdown (optional `limit` query param up to 50, optional `interval=all\|1day\|7days\|30days`) |
| `GET`   | `/api/sites/:id/visits/engagement`    | manager/team member | Visitor engagement metrics (default 30 days, optional `days` query param up to 90, or `interval=all\|1day\|7days\|30days`) |
| `GET`   | `/api/sites/:id/visits/devices`       | manager/team member | Device, screen resolution, and viewport breakdowns (optional `limit` query param up to 50, optional `interval=all\|1day\|7days\|30days`) |
| `GET`   | `/api/sites/:id/visits/locations`     | manager/team member | Inferred visitor locations from edge geo, timezone, locale, network, or unknown signals with confidence metadata (optional `limit` query param up to 50, optional `interval=all\|1day\|7days\|30days`) |
| `GET`   | `/api/sites/:id/traffic-report-schedule` | manager | Read the selected-site traffic report schedule, including `recipient_mode` (`manager`, `team`, or `selected`) and selected team member emails |
| `PUT`   | `/api/sites/:id/traffic-report-schedule` | manager | Save the selected-site traffic report schedule. `recipient_mode: "selected"` accepts only current per-site team member emails in `recipient_emails` |
| `POST`  | `/api/sites/:id/traffic-report-schedule/test` | manager | Send the selected-site traffic report immediately to the schedule's resolved recipients |
| `GET`   | `/api/sites/:id/health-monitor`       | manager/team member | Read the selected-site uptime monitor configuration and current status                                      |
| `PUT`   | `/api/sites/:id/health-monitor`       | manager | Save the selected-site uptime monitor. `recipient_mode: "selected"` accepts only current per-site team member emails in `recipient_emails` |
| `POST`  | `/api/sites/:id/health-monitor/check` | manager | Run one immediate backend health check for the selected site monitor                                         |
| `GET`   | `/api/sites/favicons/events`          | any         | Server-sent events stream announcing refreshed site favicons                                            |
| `GET`   | `/api/sites/feedback/events`          | any         | Server-sent events stream announcing new feedback                                                      |
| `POST`  | `/public/feedback`                       | public      | Submit feedback (requires `site_id`, valid `contact` as email or phone, at least one of `message` or `sentiment`, and optional `source_url` for the submitting page) |
| `POST`  | `/public/mobile-feedback`                | public      | Submit feedback from a registered mobile app with screen/app context                                   |
| `POST`  | `/public/subscriptions`                  | public      | Submit an email subscription (JSON body with `site_id`, `email`, optional `name` and `source_url`)      |
| `POST`  | `/public/sites/:id/visit-counts` | site origin | Increment aggregate daily counts with an empty JSON object. Returns `204` on success. |
| `GET`   | `/public/visits`                         | public      | Record a page visit for a site (returns a 1×1 GIF for use as a tracking pixel)                          |
| `POST`  | `/sentry/errors`                         | ingest token | Submit developer error events with `Authorization: Bearer <token>` or `X-LoopAware-Sentry-Token`       |
| `POST`  | `/sentry/browser-errors`                 | site origin | Submit browser JavaScript error events from configured site origins                                     |

## Portfolio reports

| Method | Path | Access | Description |
| --- | --- | --- | --- |
| `GET` | `/api/reports/traffic/portfolio` | authenticated | Read an owned or custom report. `scope=visible` includes all sites the caller can view. |
| `GET` | `/api/reports/traffic/portfolio/reports` | authenticated | List the caller's custom report definitions. |
| `POST` | `/api/reports/traffic/portfolio/reports` | authenticated | Create a custom report definition. |
| `PUT` | `/api/reports/traffic/portfolio/reports/:report_id` | authenticated | Update a custom report definition and its included sites. |
| `GET` | `/api/reports/traffic/portfolio/schedule` | authenticated | Read the selected portfolio report's email schedule. |
| `PUT` | `/api/reports/traffic/portfolio/schedule` | authenticated | Update the selected portfolio report's email schedule. |
| `POST` | `/api/reports/traffic/portfolio/schedule/test` | authenticated | Send the selected portfolio report immediately. |

The default scope is `owned`.
Report definitions and email schedules use owner and creator rules.
The `visible` scope uses the same access rules as `GET /api/sites`.
The API rejects `scope=visible` with a custom report ID or an unknown scope.

## Collection limits and access

Subscriptions use signed confirmation and unsubscribe links sent via email. The static frontend pages at
`/subscriptions/confirm?token=...` and `/subscriptions/unsubscribe?token=...` call the matching signed-token API
endpoints without requiring browser origin headers. Public callers cannot inspect or mutate subscription state by email.

The aggregate collector accepts at most 256 bytes and requires an empty JSON object.
A larger aggregate body returns `413` with `{"error":"invalid_json"}`.

The HTTP edge accepts at most 64 KiB for standard JSON mutations and 1 MiB for protected LA Sentry event ingestion.
Pixel-style visit requests accept no body. Requests above the global HTTP-edge limits return `413` with `{"error":"request_too_large"}` before
route dispatch. The server bounds header reads to 5 seconds, complete request reads to 10 seconds, idle connections to 60
seconds, and headers to 64 KiB. It intentionally has no response write deadline so authenticated server-sent event
streams remain open.

Public visit collection separately allows 120 requests per 30 seconds for each site and verified
client address, then returns `429`. Forwarding-header spoofing cannot create a new window.

LA Sentry ingest accepts JSON with `site_id`, `event_id`, `timestamp`, `platform`, `environment`, `release`, `level`,
`message`, `exception_type`, `stacktrace`, `request`, `user_hash`, `tags`, and `extra`. Rotate the per-site token from
the dashboard `LA Sentry` tab. Tokens are shown only once and are intended for server-side clients. The browser harness uses
`/sentry/browser-errors` without a token. Browser events are accepted only from the site's configured `allowed_origin`
values, are rate-limited by client IP, and store minimized request metadata.

The `allowed_origin` field for a site may contain multiple origins separated by spaces or commas (for example `https://mprlab.com http://localhost:8080`). Widgets, subscribe forms, and pixels will accept requests from any configured origin while still rejecting traffic from unknown sites.

Detailed traffic breakdowns support `sort=page_views` and `sort=unique_visitors`.
This applies to `/visits/stats`, `/visits/attribution`, `/visits/devices`, and `/visits/locations` for one site.
The default is `page_views`. Invalid sort values return `400 invalid_sort`.

Each detailed bucket includes `visit_count` for page views and `unique_visitor_count` for distinct visitors.
The API applies the selected order before the result limit.
Unique counts exclude empty visitor identifiers and use the final normalized bucket.
Page views include visits without a visitor identifier.

Equal unique counts use the canonical bucket key in ascending order.
The site, interval, and bot filters apply to both metrics.
A visitor can appear in multiple buckets or dates. These unique counts are not additive.

Aggregate sites keep their unavailable visitor metrics.

The `/api/me` response includes a `role` value of `admin` or `user` and an `avatar.url` pointing to the caller's cached
profile image (served from `/api/me/avatar`). The dashboard uses this payload to render the account card and determine
site scope.

Authenticated users can create sites. Owners, creators, and global admins can update and delete sites and add per-site team member emails.
They can send site traffic reports and health alerts to the manager, the whole site team, or selected team members. Team members can read assigned site data after signing in with
the matching Google email, but cannot manage site settings, memberships, schedules, or health monitors.

Site health checks are backend synthetic GET probes against the configured public HTTP(S) target. They do not require a
customer-site heartbeat script or any extra JavaScript beyond the existing widget/pixel/Sentry embeds. Monitor configuration rejects private, loopback, link-local, and special-use addresses.
The probe validates redirect targets again.
Alerts occur only on status transitions after the configured consecutive-failure threshold.

The server startup migration fills missing `creator_email` values with `temirov@gmail.com`.
This one-time migration preserves creator-based visibility for sites from before LA-57. New site creations
store the authenticated creator separately from the configured owner mailbox.
