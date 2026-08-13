# LoopAware

[![CI](https://github.com/tyemirov/loopaware/actions/workflows/ci.yml/badge.svg)](https://github.com/tyemirov/loopaware/actions/workflows/ci.yml)
[![License: Source Available](https://img.shields.io/badge/License-Source%20Available-blue)](./LICENSE)
[![Go 1.26](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go)](https://go.dev)
[![Latest Release](https://img.shields.io/github/v/release/tyemirov/loopaware)](https://github.com/tyemirov/loopaware/releases)

**Privacy-first feedback widget, traffic analytics, uptime checks, and developer monitoring.** Drop a single script tag on your site to collect customer feedback, capture email subscribers, track visits, and report browser errors -- all backed by a role-aware dashboard and a self-hosted SQLite backend.

- **Free** for personal and non-revenue projects
- **Commercial license** required for revenue-generating use
- See [LICENSE](./LICENSE) for details

<!-- TODO: Add screenshot of dashboard here -->
<!-- ![LoopAware Dashboard](docs/screenshot-dashboard.png) -->

## Quick Start

```bash
# Clone and start the development stack
git clone https://github.com/tyemirov/loopaware.git
cd loopaware
./scripts/up.sh

# Open the dashboard
open http://localhost:8080/login
```

Embed the feedback widget on any page:

```html
<script src="https://loopaware.mprlab.com/widget.js?site_id=YOUR_SITE_ID" defer></script>
```

## Highlights

- Shared `mpr-ui` sign-in with TAuth-issued sessions and TAuth verifier-backed API protection
- Role-aware dashboard (`/app`) with admin, creator/owner, and per-site team-member scopes
- YAML configuration for privileged accounts (`configs/config.loopaware.yml`)
- REST API to create, update, and inspect sites, feedback, subscribers, and traffic
- Background favicon refresh scheduler with live dashboard notifications
- Embeddable JavaScript widget with strict origin validation
- Email subscription capture via an embeddable subscribe form
- Privacy-safe traffic pixel with per-site visit and visitor counts
- Daily, weekly, or monthly traffic report emails delivered through Pinguin to a manager, the whole site team, or selected members
- Backend site health monitoring with public-target validation, thresholded down/recovered transitions, and Pinguin email alerts
- First-class LA Sentry developer error monitoring with protected server-to-server ingest and origin-bound browser capture
- SQLite-first storage with pluggable drivers
- Public privacy policy and compliance endpoints for visibility
- Table-driven tests and fast in-memory SQLite fixtures

## Configuration

Tracked LoopAware product configuration lives under `configs/`. Its `.env.*.example` files document variable names with
intentionally unusable values; they are not templates or runtime inputs. The directory also holds service config templates
and the LoopAware backend runtime config consumed by local Docker Compose and mounted by the declared production service. Real local `configs/.env.*`
files are intentionally ignored and must be created explicitly, never copied or sourced from the examples. Legacy repo-root
`.env.*` files are unsupported duplicates and should be moved into `configs/` and deleted. Test-only Compose files and the
canonical config-audit environment fixtures belong under `tests/`, not `configs/`.

### 1. Backend runtime config (`configs/config.loopaware.yml`)

LoopAware reads one backend runtime YAML file through `github.com/tyemirov/utils/runtimeconfig`. The selected file is parsed strictly, shell placeholders are expanded during that parse, and the server consumes the populated typed config after validation. Runtime values are not read from environment variables or config flags after parsing.

```yaml
server:
  address: "${APP_ADDR}"
  public_base_url: "${PUBLIC_BASE_URL}"
  trusted_proxy_cidrs:
    - "${TRUSTED_PROXY_CIDR}"
  trusted_edge_geo_proxy_cidrs: []

database:
  driver: "${DB_DRIVER}"
  dsn: "${DB_DSN}"

auth:
  session_secret: "${SESSION_SECRET}"
  tauth:
    base_url: "${TAUTH_BASE_URL}"
    tenant_id: "${TAUTH_TENANT_ID}"
    jwt_signing_key: "${TAUTH_JWT_SIGNING_KEY}"
    session_cookie_name: "${TAUTH_SESSION_COOKIE_NAME}"

pinguin:
  address: "${PINGUIN_ADDR}"
  auth_token: "${PINGUIN_AUTH_TOKEN}"
  tenant_id: "${PINGUIN_TENANT_ID}"
  connection_timeout_seconds: 5
  operation_timeout_seconds: 30

notifications:
  subscription_enabled: true
  traffic_report_emails_enabled: true

admins:
  - temirov@gmail.com
```

LoopAware loads the file specified by `--config` (default `configs/config.loopaware.yml`) before starting the HTTP server. Administrator emails come from the YAML `admins` list. When the list is empty the server starts without administrators and records a warning in the logs.

### 2. Placeholder inputs

The local Compose env files provide shell values for placeholders used by `configs/config.loopaware.yml`; they are not an alternate runtime config source.

| Variable               | Required | Description                                                 |
|------------------------|----------|-------------------------------------------------------------|
| `APP_ADDR`             | ✅        | Listen address, for example `:8080`                         |
| `PUBLIC_BASE_URL`      | ✅        | Frontend origin used for CORS and subscription links        |
| `TRUSTED_PROXY_CIDR`   | ✅        | Canonical CIDR of the reverse proxy allowed to set client forwarding headers |
| `DB_DRIVER`            | ✅        | Storage driver (`sqlite`, etc.)                             |
| `DB_DSN`               | ✅        | Driver-specific DSN                                         |
| `SESSION_SECRET`       | ✅        | 32+ byte secret for subscription confirmation tokens        |
| `TAUTH_BASE_URL`       | ✅        | Base URL for the TAuth API                                  |
| `TAUTH_TENANT_ID`      | ✅        | Tenant identifier configured in TAuth                       |
| `TAUTH_JWT_SIGNING_KEY`| ✅        | JWT signing key used to validate `app_session`              |
| `TAUTH_SESSION_COOKIE_NAME` | ✅   | Session cookie name set by TAuth                            |
| `PINGUIN_ADDR`         | ✅        | Pinguin gRPC address                                        |
| `PINGUIN_AUTH_TOKEN`   | ✅        | Bearer token passed to the Pinguin gRPC service             |
| `PINGUIN_TENANT_ID`    | ✅        | Tenant identifier used when calling the Pinguin gRPC API     |

Secrets can remain outside the tracked file by using placeholders. Non-secret settings can be literal YAML values when that is clearer for the environment.

`server.trusted_proxy_cidrs` must contain canonical CIDRs and cannot contain a catch-all network. Gin accepts
`X-Forwarded-For` only when the immediate peer is inside that boundary; forwarding headers from every other peer are
removed before logging, HTTPS detection, storage, or rate limiting. `server.trusted_edge_geo_proxy_cidrs` is a separate
subset of the proxy boundary. Keep it empty unless the named owned proxy overwrites caller geo headers with canonical
values. The tracked local, test, and ComputerCat stacks give gHTTP a fixed address and trust only that address. The
production manifest trusts the gateway-owned internal network because Caddy is the only public path to the API service.

When running via Docker Compose, create private local `.env.*` files explicitly. The tracked examples document variable names only; their values are intentionally unusable and must never be copied, sourced, or passed to Compose.

```bash
install -m 0600 /dev/null configs/.env.loopaware
install -m 0600 /dev/null configs/.env.tauth
install -m 0600 /dev/null configs/.env.pinguin
install -m 0600 /dev/null configs/.env.ghttp
$EDITOR configs/.env.loopaware configs/.env.tauth configs/.env.pinguin configs/.env.ghttp
```

Pinguin and LoopAware must share the same bearer secret. Set Pinguin's `GRPC_AUTH_TOKEN` and LoopAware's `PINGUIN_AUTH_TOKEN` to identical values in their respective service env files.
`configs/.env.tauth` must set `TAUTH_CONFIG_FILE=/config/config.yml`, and `configs/.env.pinguin` must set
`PINGUIN_CONFIG_PATH=/config/config.yml`, matching the files mounted by Docker Compose. `configs/.env.loopaware` must
define every placeholder referenced by `configs/config.loopaware.yml`. Those values are expanded only while parsing the
YAML file; they are not a second runtime config source. Production deployment values use the separate ignored
`.mprlab/deploy/.env` input and the manifest's `private_values` bindings. The sibling gateway reads that file only during
deployment; release and publication do not read it.
`config-audit` validates real local env files when they are present. When they are absent, each production Compose service
declares a tracked audit-only fixture under `tests/configs/` with `x-config-audit-env-file`; `.env.*.example` files are never
loaded. Runtime still requires the real local env files. Every existing repo-root or `configs/` dotenv input,
`.mprlab/deploy/.env`, and mounted private-key file must be a regular file with mode `0600`; tracked examples and audit
fixtures are not private runtime inputs, and absent private inputs remain valid for source-only checkouts.

Frontend runtime host mapping lives in `web/config.yml`, which is served directly as `/config.yml` by the static site.
It also carries per-environment frontend service settings such as `siteWidgetSiteId` for the first-party landing and
dashboard widget.

### 3. Config selection

The server command keeps only the config-file selector:

```
loopaware --config=configs/config.loopaware.yml
```

## Running locally

For Docker-based local development, use the helper script:

```bash
./scripts/up.sh
```

Stop the local stack with:

```bash
./scripts/down.sh
```

`scripts/up.sh` is the canonical startup path for Dockerized LoopAware. With no argument it opens an interactive selector.
You can also call it explicitly as `./scripts/up.sh local` or `./scripts/up.sh computercat`.
The local compose stack now includes a gHTTP proxy that serves `web/` at `http://localhost:8080` and forwards `/api`,
`/auth`, and `/public` to the backend services. That proxy is also responsible for the browser-facing
security headers on the static HTML and proxied API responses in the local stack.

If you want to run only the API process without Docker, use:

```bash
APP_ADDR=:8080 \
DB_DRIVER=sqlite \
DB_DSN="file:loopaware.sqlite?_foreign_keys=on" \
SESSION_SECRET=$(openssl rand -hex 32) \
TAUTH_BASE_URL=http://localhost:8081 \
TAUTH_TENANT_ID=loopaware \
TAUTH_JWT_SIGNING_KEY=replace-with-tauth-jwt-signing-key \
TAUTH_SESSION_COOKIE_NAME=loopaware_development_session \
PUBLIC_BASE_URL=http://localhost:8080 \
TRUSTED_PROXY_CIDR=192.0.2.1/32 \
PINGUIN_ADDR=localhost:50051 \
PINGUIN_AUTH_TOKEN=replace-with-pinguin-token \
PINGUIN_TENANT_ID=loopaware \
go run ./cmd/server --config=configs/config.loopaware.yml
```

When serving the static frontend directly from `web/`, no preparation step is required. Keep the tracked runtime
config in `web/config.yml` and serve `web/` from the frontend origin or reverse proxy that will answer `/config.yml`,
`/api`, and `/auth`.

Then open `/app` on that frontend origin to trigger the shared sign-in flow.
Ensure the TAuth service is running at the configured `auth.tauth.base_url` with a tenant that matches `auth.tauth.tenant_id`.
Administrators listed in `configs/config.loopaware.yml` can manage every site; other users see sites they own, sites
they originally created with their authenticated account, or sites where an admin added their email as a team member.

The static frontend pins `mpr-ui` to its full release commit through CDN URLs and lets `mpr-ui` own browser
authentication scaffolding. Direct jsDelivr JavaScript and CSS declarations carry SHA-384 subresource integrity;
the mpr-ui bundle is loaded by the upstream mpr-ui configurator from that immutable commit URL. Every HTML entry point
declares the canonical `configs/content-security-policy.txt` static-hosting CSP before executable or styled content. The local proxies enforce the same policy
as a response header and add `frame-ancestors 'none'`, which CSP metadata cannot enforce. Run
`make browser-security-audit` to detect dependency, integrity, CSP, or proxy-policy drift. Do not copy third-party
browser bundles into `web/`; non-CDN frontend dependencies are forbidden by architecture.

## Authentication flow

1. Users visit `/login` (automatic redirect from protected routes).
2. `mpr-ui` drives the browser sign-in lifecycle against the configured TAuth tenant.
3. TAuth issues and refreshes the session cookie configured by `auth.tauth.session_cookie_name`.
4. `api.AuthManager` validates the session with TAuth's verifier, injects user details into the request context, and enforces admin,
   owner, or team-member site access.
5. The dashboard and JSON APIs consume the authenticated context.

## Static frontend

LoopAware’s frontend lives in `web/` and is hosted separately (CDN or reverse proxy). It includes:

- `/login` — landing page with shared `mpr-ui`/TAuth sign-in.
- `/resources` — crawlable public resource index with focused product and use-case pages.
- `/privacy` — static privacy policy linked from the landing and dashboard footers.
- `/app` — dashboard shell (data loaded via `/api/*`).
- `/subscriptions/confirm` and `/subscriptions/unsubscribe` — email link pages.
- `/widget.js`, `/subscribe.js`, `/pixel.js`, `/la-sentry.js` — embeddable JavaScript assets.
- `/sentry/errors` — protected server-to-server developer error ingest.
- `/sentry/browser-errors` — origin-bound browser developer error ingest.

The repository does not vendor third-party browser dependencies into `web/`. External JavaScript and CSS, including UI
libraries, must be referenced through pinned CDN URLs. Any browser dependency that is not delivered by CDN is
forbidden. `web/` is reserved for LoopAware-authored assets only, so deployments, cache behavior, and browser tests
exercise the same delivery path used in production.

Set `PUBLIC_BASE_URL` to the frontend origin so the API emits correct links and CORS allows browser access. Use
absolute `data-api-origin` attributes (or `api_origin` query params) on embed scripts when the API runs on a different
origin. The dashboard and login pages call `/api` and `/auth` relative to the frontend origin, so split-origin
deployments should use a reverse proxy or update the static HTML in `web/` to point at those services.
The tracked runtime host mapping lives in `web/config.yml`, which `web/runtime-env.js` fetches directly at runtime.
Canonical SEO metadata, Open Graph URLs, `robots.txt`, and `sitemap.xml` are fixed to the single public site
`https://loopaware.mprlab.com` and are not environment-specific. The public resource hub lives at
`https://loopaware.mprlab.com/resources/` and links focused guides for feedback, subscriber workflows, traffic analytics,
uptime checks, access control, self-hosted deployments, and LA Sentry monitoring.
Each environment may also define `services.siteWidgetSiteId` there to bootstrap the first-party feedback widget on
`/login` and `/app` without hard-coding a site UUID into the static HTML.

## REST API

All authenticated endpoints live under `/api` and require the configured TAuth session cookie. Public collection endpoints for
feedback, subscriptions, and visits do not require a session but still enforce per-site origin rules. JSON responses
include Unix timestamps in seconds.

Traffic endpoints with `interval=1day` report the trailing 24 hours. `interval=30days` uses the current 30-day UTC day window, and `interval=all` reports all recorded human visits.

| Method  | Path                                  | Role        | Description                                                                                             |
|---------|---------------------------------------|-------------|---------------------------------------------------------------------------------------------------------|
| `GET`   | `/api/me`                             | any         | Current account metadata (email, name, `role`, `avatar.url`)                                            |
| `GET`   | `/api/sites`                          | any         | Sites visible to the caller; each row includes `access_role` (`admin` or `team_member`)                  |
| `POST`  | `/api/sites`                          | any         | Create a site (requires `name`, `allowed_origin`, `owner_email`)                                        |
| `PATCH` | `/api/sites/:id`                      | owner/admin | Update name/origin; admins may reassign ownership                                                       |
| `DELETE`| `/api/sites/:id`                      | owner/admin | Delete a site                                                                                            |
| `GET`   | `/api/sites/:id/team`                 | owner/admin | List per-site team member email assignments                                                             |
| `POST`  | `/api/sites/:id/team`                 | owner/admin | Add a per-site team member by email                                                                     |
| `DELETE`| `/api/sites/:id/team/:member_id`      | owner/admin | Remove a per-site team member assignment                                                                |
| `GET`   | `/api/sites/:id/mobile-apps`          | owner/admin/team member | List native mobile apps registered for feedback submissions                                    |
| `POST`  | `/api/sites/:id/mobile-apps`          | owner/admin | Register a native mobile app for mobile feedback submissions                                            |
| `GET`   | `/api/sites/:id/messages`             | owner/admin/team member | List feedback messages (newest first)                                                         |
| `GET`   | `/api/sites/:id/subscribers`          | owner/admin/team member | List subscribers for a site                                                                   |
| `GET`   | `/api/sites/:id/subscribers/export`   | owner/admin/team member | Download subscribers as CSV                                                                   |
| `PATCH` | `/api/sites/:id/subscribers/:subscriber_id` | owner/admin | Update a subscriber’s status (confirm or unsubscribe)                                             |
| `DELETE`| `/api/sites/:id/subscribers/:subscriber_id` | owner/admin | Delete a subscriber                                                                                |
| `GET`   | `/api/sites/:id/visits/stats`         | owner/admin/team member | Aggregate visit and unique visitor counts plus recent visits and top pages (optional `interval=all\|1day\|30days`) |
| `GET`   | `/api/sites/:id/visits/export`        | owner/admin/team member | Download traffic visits as CSV (optional `interval=all\|1day\|30days`)                      |
| `GET`   | `/api/sites/:id/sentry/issues`        | owner/admin/team member | List grouped developer error issues for a site                                                |
| `GET`   | `/api/sites/:id/sentry/issues/:issue_id` | owner/admin/team member | Inspect latest and recent LA Sentry error occurrences                                      |
| `PATCH` | `/api/sites/:id/sentry/issues/:issue_id` | owner/admin | Update issue status (`unresolved`, `resolved`, or `ignored`)                                         |
| `POST`  | `/api/sites/:id/sentry/token`         | owner/admin | Rotate and reveal a per-site LA Sentry ingest token                                                     |
| `GET`   | `/api/sites/:id/visits/trend`         | owner/admin/team member | Daily visit trend (default 7 days, optional `days` query param up to 30, or `interval=all\|1day\|30days`) |
| `GET`   | `/api/sites/:id/visits/attribution`   | owner/admin/team member | Source/medium/campaign attribution breakdown (optional `limit` query param up to 50; optional `interval=all\|1day\|30days`) |
| `GET`   | `/api/sites/:id/visits/engagement`    | owner/admin/team member | Visitor engagement metrics (default 30 days, optional `days` query param up to 90, or `interval=all\|1day\|30days`) |
| `GET`   | `/api/sites/:id/visits/devices`       | owner/admin/team member | Device, screen resolution, and viewport breakdowns (optional `limit` query param up to 50; optional `interval=all\|1day\|30days`) |
| `GET`   | `/api/sites/:id/visits/locations`     | owner/admin/team member | Inferred visitor locations from edge geo, timezone, locale, network, or unknown signals with confidence metadata (optional `limit` query param up to 50; optional `interval=all\|1day\|30days`) |
| `GET`   | `/api/sites/:id/traffic-report-schedule` | owner/admin | Read the selected-site traffic report schedule, including `recipient_mode` (`manager`, `team`, or `selected`) and selected team member emails |
| `PUT`   | `/api/sites/:id/traffic-report-schedule` | owner/admin | Save the selected-site traffic report schedule; `recipient_mode: "selected"` accepts only current per-site team member emails in `recipient_emails` |
| `POST`  | `/api/sites/:id/traffic-report-schedule/test` | owner/admin | Send the selected-site traffic report immediately to the schedule's resolved recipients |
| `GET`   | `/api/sites/:id/health-monitor`       | owner/admin/team member | Read the selected-site uptime monitor configuration and current status                                      |
| `PUT`   | `/api/sites/:id/health-monitor`       | owner/admin | Save the selected-site uptime monitor; `recipient_mode: "selected"` accepts only current per-site team member emails in `recipient_emails` |
| `POST`  | `/api/sites/:id/health-monitor/check` | owner/admin | Run one immediate backend health check for the selected site monitor                                         |
| `GET`   | `/api/sites/favicons/events`          | any         | Server-sent events stream announcing refreshed site favicons                                            |
| `GET`   | `/api/sites/feedback/events`          | any         | Server-sent events stream announcing new feedback                                                      |
| `POST`  | `/public/feedback`                       | public      | Submit feedback (requires `site_id`, valid `contact` as email or phone, at least one of `message` or `sentiment`, and optional `source_url` for the submitting page) |
| `POST`  | `/public/mobile-feedback`                | public      | Submit feedback from a registered mobile app with screen/app context                                   |
| `POST`  | `/public/subscriptions`                  | public      | Submit an email subscription (JSON body with `site_id`, `email`, optional `name` and `source_url`)      |
| `GET`   | `/public/visits`                         | public      | Record a page visit for a site (returns a 1×1 GIF for use as a tracking pixel)                          |
| `POST`  | `/sentry/errors`                         | ingest token | Submit developer error events with `Authorization: Bearer <token>` or `X-LoopAware-Sentry-Token`       |
| `POST`  | `/sentry/browser-errors`                 | site origin | Submit browser JavaScript error events from configured site origins                                     |

Subscriptions use signed confirmation and unsubscribe links sent via email. The static frontend pages at
`/subscriptions/confirm?token=...` and `/subscriptions/unsubscribe?token=...` call the matching signed-token API
endpoints without requiring browser origin headers. Public callers cannot inspect or mutate subscription state by email.

The HTTP edge accepts at most 64 KiB for standard JSON mutations and 1 MiB for protected LA Sentry event ingestion;
pixel-style visit requests accept no body. Oversized requests return `413` with `{"error":"request_too_large"}` before
route dispatch. The server bounds header reads to 5 seconds, complete request reads to 10 seconds, idle connections to 60
seconds, and headers to 64 KiB. It intentionally has no response write deadline so authenticated server-sent event
streams remain open. Public visit collection separately allows 120 requests per 30 seconds for each site and verified
client address, then returns `429`; forwarding-header spoofing cannot create a new window.

LA Sentry ingest accepts JSON with `site_id`, `event_id`, `timestamp`, `platform`, `environment`, `release`, `level`,
`message`, `exception_type`, `stacktrace`, `request`, `user_hash`, `tags`, and `extra`. Rotate the per-site token from
the dashboard `LA Sentry` tab; tokens are shown only once and are intended for server-side clients. The browser harness uses
`/sentry/browser-errors` without a token. Browser events are accepted only from the site's configured `allowed_origin`
values, are rate-limited by client IP, and store minimized request metadata.

The `allowed_origin` field for a site may contain multiple origins separated by spaces or commas (for example `https://mprlab.com http://localhost:8080`); widgets, subscribe forms, and pixels will accept requests from any configured origin while still rejecting traffic from unknown sites.

The `/api/me` response includes a `role` value of `admin` or `user` and an `avatar.url` pointing to the caller's cached
profile image (served from `/api/me/avatar`). The dashboard uses this payload to render the account card and determine
site scope.

Authenticated users can create sites. Owners, creators, and global admins can update and delete sites, can add
per-site team member emails, and can choose whether selected-site traffic reports and health alerts go only to the
manager, the whole site team, or selected team members. Team members can read assigned site data after signing in with
the matching Google email, but cannot manage site settings, memberships, schedules, or health monitors.

Site health checks are backend synthetic GET probes against the configured public HTTP(S) target. They do not require a
customer-site heartbeat script or any extra JavaScript beyond the existing widget/pixel/Sentry embeds. Direct private,
loopback, link-local, and special-use addresses are rejected when a monitor is saved, redirects are revalidated during
probing, and alerts are emitted only on status transitions after the configured consecutive-failure threshold.

Deployments upgraded from versions prior to LA-57 should allow the server startup migration to run once; it backfills any
sites missing a `creator_email` with `temirov@gmail.com` to preserve creator-based visibility rules. New site creations
store the authenticated creator separately from the configured owner mailbox.

## Dashboard (`/app`)

The Bootstrap front end consumes the APIs above. Features include:

- Account Settings modal with avatar, email, role badge, reports, and inactivity controls
- Site creation and owner reassignment available to every authenticated user; administrators additionally see all sites
- Owner/admin editor for site metadata, with per-site team member emails managed from the Admin dashboard section
- Selected-site traffic report scheduling with recipient selection for only the manager, the whole site team, or checked team members
- Selected-site health monitoring with enablement, target URL, interval, timeout, failure threshold, manual check, and alert-recipient controls
- Widget appearance controls that persist the bubble’s accent color, side (left/right), and bottom offset without code changes
- Feedback table with human-readable timestamps
- Subscribers panel with per-site subscriber counts, table, spreadsheet-formula-safe CSV export, and a copyable `subscribe.js` snippet
- Section selector tabs to switch between Feedback, Subscriptions, Traffic, Health, LA Sentry, and manager-only Admin tools
- Subscriber deletion via a confirmation modal
- Traffic card with visit and unique visitor counts, recent visits, and a copyable `pixel.js` snippet
- Real-time favicon refresh notifications delivered through the SSE stream
- Sign-out button provided by the shared `mpr-ui`/TAuth shell
- Inactivity prompt appears after the configured delay (defaults to 60 seconds) and logs out automatically after the configured timeout (defaults to 120 seconds) if unanswered

The dashboard automatically redirects unauthenticated visitors to `/login`.

## Embedding the widget

1. Create a site (admin) and copy the generated `<script>` tag from the API response.
2. Embed the script on any page served from one of the site’s configured `allowed_origin` values (you can supply multiple origins separated by spaces or commas). Include the `defer` attribute so the widget loads without blocking the page; the script waits for the body before rendering the UI.
3. Visitors can open the floating bubble, submit feedback with a valid email or phone plus a message and/or sentiment, and the messages appear under `/api/sites/:id/messages` and
   in the dashboard.

Example snippet (replace the base URL with your LoopAware deployment and the site identifier with the value returned by the API):

```html
<script defer src="https://loopaware.mprlab.com/widget.js?site_id=6f50b5f4-8a8f-4e4a-9d69-1b2a3c4d5e6f"></script>
```

## Adding mobile feedback

React Native and Expo apps can use the first-party client under `clients/react-native` to render a native feedback
button on selected screens. Mobile feedback is separate from LA Sentry error capture.

Install the package from npm:

```bash
npm install @loopaware/react-native
```

1. Register the native app for the site:

   ```json
   POST /api/sites/6f50b5f4-8a8f-4e4a-9d69-1b2a3c4d5e6f/mobile-apps
   {
     "platform": "ios",
     "app_identifier": "com.example.app",
     "display_name": "Example iOS"
   }
   ```

2. Store the returned public `client_id` in the app configuration.
3. Render the feedback button on screens where users should be able to comment:

   ```tsx
   <LoopAwareProvider
     siteId="6f50b5f4-8a8f-4e4a-9d69-1b2a3c4d5e6f"
     mobileClientId="client-id-from-dashboard-api"
     apiOrigin="https://loopaware.mprlab.com"
     app={{
       platform: "ios",
       applicationId: "com.example.app",
       version: "1.2.3",
       build: "44",
       environment: "production",
     }}
   >
     <CheckoutScreen />
     <LoopAwareFeedbackButton
       screen={{ name: "Checkout", path: "/checkout/payment" }}
       context={{ step: "payment", plan: "pro" }}
     />
   </LoopAwareProvider>
   ```

The public `client_id` identifies the app registration; it is not a secret. Mobile submissions validate that the client
ID, platform, and application identifier match the registered site app, then store the supplied screen, app version, and
bounded context with the feedback message.

Swift, Kotlin, and other non-React Native apps should implement the same REST contract directly against
`/public/mobile-feedback` after registering the app with `/api/sites/:id/mobile-apps`; the repository does not ship
separate native iOS or Android SDKs.

## Embedding the subscribe form

Each site exposes a subscribe snippet that renders an email capture form and posts subscriptions to `/public/subscriptions`.

1. In the dashboard, select a site and use the Subscribers panel to copy the subscribe snippet.
2. Embed the script on pages served from any of the site’s `allowed_origin` entries. The basic form looks like:

   ```html
   <script defer src="https://loopaware.mprlab.com/subscribe.js?site_id=6f50b5f4-8a8f-4e4a-9d69-1b2a3c4d5e6f"></script>
   ```

3. Optional query parameters let you adjust behavior and styling:
   - `mode=inline` (default) or `mode=bubble` for a floating button.
   - `accent=#0d6efd` to override the accent color.
   - `cta=Subscribe` to customize the button text.
   - `success=You%27re+on+the+list%21` and `error=Please+try+again.` for inline messages.
   - `name_field=false` to hide the optional name field.

The form enforces the site’s `allowed_origin` list using request headers and `source_url` and responds with inline success or
error messages so visitors never leave the page.

## Embedding the traffic pixel

The traffic pixel records page visits per site and powers the dashboard Traffic card and top-pages table.

1. In the dashboard, select a site and use the Traffic panel to copy the pixel snippet.
2. Embed the script on every page served from any of the site’s `allowed_origin` entries:

   ```html
   <script defer src="https://loopaware.mprlab.com/pixel.js?site_id=6f50b5f4-8a8f-4e4a-9d69-1b2a3c4d5e6f"></script>
   ```

3. On load, `pixel.js` sends a beacon to `/public/visits` with the site ID, current URL, referrer, browser timezone,
   browser locale, viewport, screen resolution, and a stable visitor ID stored in `localStorage`. The server also stores
   supported edge geo headers from Cloudflare, Vercel, and CloudFront only when an explicitly trusted owned proxy
   overwrites them, then prefers that location signal over browser timezone and locale hints. The tracked deployments
   reject caller-supplied edge geo metadata and use browser timezone or locale instead. Requests from origins outside the site’s `allowed_origin`
   list are rejected. Traffic from known bot user-agent signatures is stored but excluded from default dashboard totals,
   top-page rankings, trends, attribution, engagement, devices, and locations.

For non-JavaScript environments you can fall back to a plain image pixel:

```html
<img src="https://loopaware.mprlab.com/public/visits?site_id=6f50b5f4-8a8f-4e4a-9d69-1b2a3c4d5e6f&url=https%3A%2F%2Fexample.com%2F" alt="" width="1" height="1" />
```

## Capturing developer errors

Server-side clients should use the protected `/sentry/errors` endpoint with a per-site ingest token. The repository
collects first-party client entrypoints under `clients/`:

- Go: `clients/go/lasentry`
- Python: `clients/python/la_sentry`
- Browser: `clients/browser` documents the harness served from `web/la-sentry.js`

Browser pages can use the standalone harness without exposing the server-side token:

```html
<script defer src="https://loopaware.mprlab.com/la-sentry.js?site_id=6f50b5f4-8a8f-4e4a-9d69-1b2a3c4d5e6f&environment=production&release=2026.04.24"></script>
```

The browser harness installs `window.LASentry.captureError(error, attrs)` and automatically captures uncaught
`error` and `unhandledrejection` events. It sends sanitized URL/referrer/user-agent metadata, stack frames, tags, and
explicit `extra` values supplied by application code.

## Development workflow

```bash
make format
make lint
make test
```

`make test` runs the Playwright integration suite against `tests/docker-compose.yml`, with test-owned env fixtures under
`tests/configs/`. That stack builds the API image, serves `web/` via gHTTP, and exercises both UI and `/api/*` flows.
Use `make test-unit` for Go-only tests and `make test-integration-api` to focus on API specs. Playwright artifacts
(traces, screenshots, videos) land under `tests/test-results/` on failure. The integration runner tears its compose
project down on exit, including failures and signal exits.
The runner rejects inherited `LOOPAWARE_BASE_URL`, `LOOPAWARE_API_BASE_URL`, `LOOPAWARE_ENV_FILE`,
`COMPOSE_PROJECT_NAME`, `DOCKER_HOST`, `DOCKER_CONTEXT`, and Docker TLS inputs; requires a local `unix://` or `npipe://`
Docker endpoint; and supplies its own localhost frontend, random loopback-only backend port, test env file, and unique
Compose project. The test helper independently rejects any non-loopback frontend or backend target.

Use `make test-live-favicons` when validating customer-site favicon collection against known public websites. That
target performs live network requests and is intentionally outside `make ci` so third-party uptime does not gate normal
development.

Run `make security-audit` to execute the pinned Go vulnerability analyzer, full dependency audits for every npm
project, and the immutable supported-container-base contract. This audit is part of `make ci`; it requires network access
to the Go vulnerability database and npm advisory service. To validate the complete shipped image after a base or Go
dependency update, build it with `docker build --pull -t loopaware:security-audit .` and inspect it with
`docker scout cves --only-severity critical,high --exit-code local://loopaware:security-audit`.
Repository administrators can run the read-only `make github-security-audit` gate to verify the live Actions, branch
protection, Dependabot, secret-scanning, and gateway-owned Pages controls against the tracked security contract.

## Release, Publish, Deploy

LoopAware declares its complete production shape in `.mprlab/deploy/resources.yml`. That schema-v3 manifest is the only tracked production lifecycle declaration. It owns the backend image and service, retained data, runtime capability, public route, health check, Pages site, React Native npm package, native mobile app, TAuth tenant, and private-value bindings.

Run the three phases in order from this repository:

```bash
make release
make publish
make deploy
```

Each target is a zero-argument wrapper around the exact sibling `../mprlab-gateway` checkout. The gateway owns lifecycle validation, sealed artifact assembly, provider publication, host assembly, and verification. LoopAware owns only the local native builders and provider clients selected by its manifest; it does not carry a parallel lifecycle controller.

Deployment-only private values belong in the ignored `.mprlab/deploy/.env` file with mode `0600`. The manifest binds those values through its `private_values` resource; secret bytes never belong in the manifest. Release and publish do not read this deployment input.

The mobile resource builds locally. Release runs Expo prebuild only as the native-project generator, then creates one signed App Store Connect IPA with Xcode and one signed Google Play AAB with Gradle. Publish validates and uploads those exact sealed files through `xcrun altool` and the Google Play Android Publisher API. No Expo account, EAS project, hosted build, or EAS submit is part of the lifecycle.

The `mobile_application` entry is the native lifecycle authority: its four build and publish paths are executable directly from this repository. Native build and store-provider acceptance therefore uses LoopAware's real scripts and credentials and does not depend on a gateway checkout. Gateway involvement is limited to coordinating the complete application lifecycle, including the backend service and Caddy route.

The local machine must have the canonical Apple distribution signing identity and Android upload key described by `mobile/android-release-identity.json`. App Store Connect uses the ignored `configs/AuthKey_82P4KZ86HM.p8`; Google Play uses Application Default Credentials and the ignored LoopAware upload-keystore files under `~/.local/share/loopaware/android-upload/`. These provider credentials remain outside Git and are checked before their platform build or publication begins.

There are no app-owned dry-run lifecycle aliases. For a non-mutating inspection, run the gateway's `plan-app-release`, `plan-app-publish`, or `plan-app-deploy` target with `MPRLAB_APP_ROOT` set to this repository. Production activation remains an operator action.

## Docker

Ensure the container receives the placeholder inputs used by `configs/config.loopaware.yml` and mounts that backend runtime config file.

```bash
install -m 0600 /dev/null configs/.env.loopaware
install -m 0600 /dev/null configs/.env.tauth
install -m 0600 /dev/null configs/.env.pinguin
install -m 0600 /dev/null configs/.env.ghttp
$EDITOR configs/.env.loopaware configs/.env.tauth configs/.env.pinguin configs/.env.ghttp
./scripts/up.sh
```

The compose file binds `configs/config.loopaware.yml` into the LoopAware container at `/app/configs/config.loopaware.yml`
and loads per-service placeholder values via `env_file` from `configs/.env.*`.
The production Dockerfile pins the Go 1.26.6/Alpine 3.24 build image and Alpine 3.24 runtime image by multi-architecture manifest-list digests.
When you advance a base, update its human-readable tag and digest together.
The container runs as root so the SQLite data volume remains writable.
To use an unprivileged user, configure the Docker image to change the mounted directory owner before it starts the binary.

The default local stack uses `configs/.env.ghttp` for the gHTTP static proxy. Start and stop local Docker stacks only
through the helper scripts:

```bash
./scripts/up.sh local
./scripts/down.sh local
```

For the computercat TLS stack, use:

```bash
install -m 0600 /dev/null configs/.env.loopaware.computercat
install -m 0600 /dev/null configs/.env.tauth.computercat
install -m 0600 /dev/null configs/.env.pinguin.computercat
install -m 0600 /dev/null configs/.env.ghttp.computercat
$EDITOR configs/.env.loopaware.computercat configs/.env.tauth.computercat configs/.env.pinguin.computercat configs/.env.ghttp.computercat
./scripts/up.sh computercat
```

The computercat stack exposes `https://computercat.tyemirov.net:4443` through gHTTP as the TLS terminator and reverse
proxy. The proxy expects certificates at `/media/share/Drive/exchange/certs/computercat/computercat-cert.pem` and
`/media/share/Drive/exchange/certs/computercat/computercat-key.pem`; Compose mounts only those two files into `/certs`,
and `config-audit` requires the private key to use mode `0600` whenever it exists.
`configs/.env.ghttp.computercat` owns the TLS and proxy settings, including `GHTTP_SERVE_DIRECTORY=/data`,
`GHTTP_SERVE_PORT=4443`, `GHTTP_SERVE_TLS_CERTIFICATE=/certs/computercat-cert.pem`,
`GHTTP_SERVE_TLS_PRIVATE_KEY=/certs/computercat-key.pem`, and the reverse-proxy routes for TAuth, LoopAware public/API,
and Sentry paths:

```dotenv
GHTTP_SERVE_DIRECTORY=/data
GHTTP_SERVE_PORT=4443
GHTTP_SERVE_LOGGING_TYPE=JSON
GHTTP_SERVE_TLS_CERTIFICATE=/certs/computercat-cert.pem
GHTTP_SERVE_TLS_PRIVATE_KEY=/certs/computercat-key.pem
GHTTP_SERVE_PROXIES=/tauth.js=http://la-tauth:8082,/me=http://la-tauth:8082,/auth/=http://la-tauth:8082,/public/=http://loopaware-api:8080,/sentry/=http://loopaware-api:8080,/api/=http://loopaware-api:8080
```

Set the private computercat env files to the public origin `https://computercat.tyemirov.net:4443` so the browser uses the
reverse proxy for both LoopAware and TAuth; the documentation-only examples intentionally do not provide runnable defaults.
TAuth requires HTTPS for secure cookies when
`allow_insecure_http=false`; gHTTP’s reverse proxy does not currently set `X-Forwarded-Proto`, so keep
`TAUTH_ALLOW_INSECURE_HTTP=true` unless a fronting proxy forwards `X-Forwarded-Proto=https`.
