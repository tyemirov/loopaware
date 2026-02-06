# LoopAware

LoopAware collects customer feedback through a lightweight widget, authenticates operators with Google, and offers a
role-aware dashboard for managing sites and messages.

## Highlights

- Google Identity Services authentication via TAuth
- Role-aware dashboard (`/app`) with admin and creator/owner scopes
- YAML configuration for privileged accounts (`config.yaml`)
- REST API to create, update, and inspect sites, feedback, subscribers, and traffic
- Background favicon refresh scheduler with live dashboard notifications
- Embeddable JavaScript widget with strict origin validation
- Email subscription capture via an embeddable subscribe form
- Privacy-safe traffic pixel with per-site visit and visitor counts
- SQLite-first storage with pluggable drivers
- Public privacy policy and sitemap endpoints for compliance visibility
- Table-driven tests and fast in-memory SQLite fixtures

## Configuration

### 1. Admin roster (`config.yaml`)

Create a YAML file next to the binary with the email addresses that should receive administrator privileges (the file is optional if you prefer environment-only configuration):

```yaml
admins:
  - temirov@gmail.com
```

LoopAware loads the file specified by `--config` (default `config.yaml`) before starting the HTTP server.
Set the `ADMINS` environment variable with a comma-separated list (for example `ADMINS=alice@example.com,bob@example.com`) to override the YAML roster without editing the file. When neither source is present the server starts without administrators and records a warning in the logs.

### 2. Environment variables

| Variable               | Required | Description                                                 |
|------------------------|----------|-------------------------------------------------------------|
| `GOOGLE_CLIENT_ID`     | ✅        | OAuth client ID from Google Cloud Console                   |
| `SESSION_SECRET`       | ✅        | 32+ byte secret for subscription confirmation tokens        |
| `TAUTH_BASE_URL`       | ✅        | Base URL for the TAuth service (serves `/tauth.js`)          |
| `TAUTH_TENANT_ID`      | ✅        | Tenant identifier configured in TAuth                       |
| `TAUTH_JWT_SIGNING_KEY`| ✅        | JWT signing key used to validate `app_session`              |
| `TAUTH_SESSION_COOKIE_NAME` | ✅   | Session cookie name set by TAuth (defaults to `app_session`) |
| `PINGUIN_AUTH_TOKEN`¹  | ✅        | Bearer token passed to the Pinguin gRPC service             |
| `PINGUIN_TENANT_ID`    | ✅        | Tenant identifier used when calling the Pinguin gRPC API     |
| `ADMINS`               | ⚙️       | Comma-separated admin emails; overrides the YAML roster     |
| `PUBLIC_BASE_URL`      | ⚙️       | Public URL of the service (default `http://localhost:8080`) |
| `APP_ADDR`             | ⚙️       | Listen address (default `:8080`)                            |
| `DB_DRIVER`            | ⚙️       | Storage driver (`sqlite`, etc.)                             |
| `DB_DSN`               | ⚙️       | Driver-specific DSN                                         |

Secrets must come from the environment; only non-sensitive settings belong in `config.yaml`.

When running via Docker Compose, copy the tracked env templates under `configs/` and edit the local `.env.*` files:

```bash
cp configs/.env.loopaware.example configs/.env.loopaware
cp configs/.env.tauth.example configs/.env.tauth
cp configs/.env.pinguin.example configs/.env.pinguin
$EDITOR configs/.env.loopaware configs/.env.tauth configs/.env.pinguin
```

¹Pinguin and LoopAware must share the **exact same** bearer secret. Provide identical values for `GRPC_AUTH_TOKEN` and `PINGUIN_AUTH_TOKEN`, for example:

```dotenv
GRPC_AUTH_TOKEN=loopaware-local-secret
PINGUIN_AUTH_TOKEN=loopaware-local-secret
```

LoopAware falls back to `GRPC_AUTH_TOKEN` when `PINGUIN_AUTH_TOKEN` is empty, so exporting the shared value once at runtime also works.

### 3. Flags

All configuration options are also exposed as Cobra flags:

```
loopaware --config=config.yaml \
  --app-addr=:8080 \
  --db-driver=sqlite \
  --db-dsn="file:loopaware.sqlite?_foreign_keys=on" \
  --google-client-id=$GOOGLE_CLIENT_ID \
  --session-secret=$SESSION_SECRET \
  --tauth-base-url=$TAUTH_BASE_URL \
  --tauth-tenant-id=$TAUTH_TENANT_ID \
  --tauth-signing-key=$TAUTH_JWT_SIGNING_KEY \
  --tauth-session-cookie-name=$TAUTH_SESSION_COOKIE_NAME \
  --public-base-url=https://feedback.example.com
```

Flags are optional when the equivalent environment variables are set.

## Running locally

```bash
GOOGLE_CLIENT_ID=... \
SESSION_SECRET=$(openssl rand -hex 32) \
TAUTH_BASE_URL=http://localhost:8081 \
TAUTH_TENANT_ID=loopaware \
TAUTH_JWT_SIGNING_KEY=replace-with-tauth-jwt-signing-key \
TAUTH_SESSION_COOKIE_NAME=app_session_loopaware \
go run ./cmd/server --config=config.yaml
```

Open `http://localhost:8080/app` to trigger Google Sign-In. Ensure the TAuth service is running at
`TAUTH_BASE_URL` with a tenant that matches `TAUTH_TENANT_ID`. Administrators listed in `config.yaml` can manage every
site; other users see only the sites they own or originally created with their Google account.

## Authentication flow

1. Users visit `/login` (automatic redirect from protected routes).
2. TAuth issues the session cookie configured by `TAUTH_SESSION_COOKIE_NAME` (defaults to `app_session`) via Google Identity Services and keeps it refreshed.
3. `httpapi.AuthManager` validates the session JWT, injects user details into the request context, and enforces admin /
   owner access.
4. The dashboard and JSON APIs consume the authenticated context.

## Public pages

LoopAware serves a minimal public surface derived from `PUBLIC_BASE_URL`:

- `/login` — marketing-focused landing page with TAuth-backed Google Sign-In.
- `/privacy` — static privacy policy linked from the landing and dashboard footers.
- `/sitemap.xml` — XML sitemap enumerating the login and privacy URLs for search engines.

Set `PUBLIC_BASE_URL` to the externally reachable origin so the sitemap emits fully qualified links for crawlers.

## REST API

All authenticated endpoints live under `/api` and require the TAuth session cookie configured by `TAUTH_SESSION_COOKIE_NAME`. Public collection endpoints for
feedback, subscriptions, and visits do not require a session but still enforce per-site origin rules. JSON responses
include Unix timestamps in seconds.

| Method  | Path                                  | Role        | Description                                                                                             |
|---------|---------------------------------------|-------------|---------------------------------------------------------------------------------------------------------|
| `GET`   | `/api/me`                             | any         | Current account metadata (email, name, `role`, `avatar.url`)                                            |
| `GET`   | `/api/sites`                          | any         | Sites visible to the caller (admin = all, user = owned)                                                 |
| `POST`  | `/api/sites`                          | any         | Create a site (requires `name`, `allowed_origin`, `owner_email`)                                        |
| `PATCH` | `/api/sites/:id`                      | owner/admin | Update name/origin; admins may reassign ownership                                                       |
| `DELETE`| `/api/sites/:id`                      | owner/admin | Delete a site                                                                                            |
| `GET`   | `/api/sites/:id/messages`             | owner/admin | List feedback messages (newest first)                                                                   |
| `GET`   | `/api/sites/:id/subscribers`          | owner/admin | List subscribers for a site                                                                             |
| `GET`   | `/api/sites/:id/subscribers/export`   | owner/admin | Download subscribers as CSV                                                                             |
| `PATCH` | `/api/sites/:id/subscribers/:subscriber_id` | owner/admin | Update a subscriber’s status (confirm or unsubscribe)                                             |
| `DELETE`| `/api/sites/:id/subscribers/:subscriber_id` | owner/admin | Delete a subscriber                                                                                |
| `GET`   | `/api/sites/:id/visits/stats`         | owner/admin | Aggregate visit and unique visitor counts plus recent visits and top pages                              |
| `GET`   | `/api/sites/favicons/events`          | any         | Server-sent events stream announcing refreshed site favicons                                            |
| `GET`   | `/api/sites/feedback/events`          | any         | Server-sent events stream announcing new feedback                                                      |
| `POST`  | `/api/feedback`                       | public      | Submit feedback (requires JSON body with `site_id`, `contact`, `message`)                               |
| `POST`  | `/api/subscriptions`                  | public      | Submit an email subscription (JSON body with `site_id`, `email`, optional `name` and `source_url`)      |
| `POST`  | `/api/subscriptions/confirm`          | public      | Confirm a subscription for a given `site_id` and email                                                  |
| `POST`  | `/api/subscriptions/unsubscribe`      | public      | Unsubscribe an email address for a given `site_id`                                                      |
| `GET`   | `/subscriptions/confirm`              | public      | Confirm a pending subscription via email token (HTML page)                                              |
| `GET`   | `/subscriptions/unsubscribe`          | public      | Unsubscribe via email token (HTML page)                                                                 |
| `GET`   | `/api/visits`                         | public      | Record a page visit for a site (returns a 1×1 GIF for use as a tracking pixel)                          |
| `GET`   | `/widget.js`                          | public      | Serve embeddable JavaScript feedback widget                                                             |
| `GET`   | `/subscribe.js`                       | public      | Serve embeddable JavaScript subscribe form                                                              |
| `GET`   | `/pixel.js`                           | public      | Serve embeddable JavaScript visit tracking pixel                                                        |

Subscriptions use confirmation and unsubscribe links sent via email: `GET /subscriptions/confirm?token=...` confirms the pending subscriber, and `GET /subscriptions/unsubscribe?token=...` unsubscribes, both without requiring browser origin headers.

The `allowed_origin` field for a site may contain multiple origins separated by spaces or commas (for example `https://mprlab.com http://localhost:8080`); widgets, subscribe forms, and pixels will accept requests from any configured origin while still rejecting traffic from unknown sites.

The `/api/me` response includes a `role` value of `admin` or `user` and an `avatar.url` pointing to the caller's cached
profile image (served from `/api/me/avatar`). The dashboard uses this payload to render the account card and determine
site scope.

Both roles can create, update, and delete sites. Administrators additionally view every site in the system, while users
see only the sites they own or originally created.

Deployments upgraded from versions prior to LA-57 should allow the server startup migration to run once; it backfills any
sites missing a `creator_email` with `temirov@gmail.com` to preserve creator-based visibility rules. New site creations
store the authenticated creator separately from the configured owner mailbox.

## Dashboard (`/app`)

The Bootstrap front end consumes the APIs above. Features include:

- Account card with avatar, email, and role badge
- Site creation and owner reassignment available to every authenticated user; administrators additionally see all sites
- Owner/admin editor for site metadata
- Widget placement controls that persist the bubble’s side (left/right) and bottom offset without code changes
- Feedback table with human-readable timestamps
- Subscribers panel with per-site subscriber counts, table, CSV export, and a copyable `subscribe.js` snippet
- Section selector tabs to switch between Feedback, Subscriptions, and Traffic
- Subscriber deletion via a confirmation modal
- Traffic card with visit and unique visitor counts, recent visits, and a copyable `pixel.js` snippet
- Real-time favicon refresh notifications delivered through the SSE stream
- Sign-out button wired to TAuth (`/auth/logout`)
- Inactivity prompt appears after the configured delay (defaults to 60 seconds) and logs out automatically after the configured timeout (defaults to 120 seconds) if unanswered

The dashboard automatically redirects unauthenticated visitors to `/login`.

## Embedding the widget

1. Create a site (admin) and copy the generated `<script>` tag from the API response.
2. Embed the script on any page served from one of the site’s configured `allowed_origin` values (you can supply multiple origins separated by spaces or commas). Include the `defer` attribute so the widget loads without blocking the page; the script waits for the body before rendering the UI.
3. Visitors can open the floating bubble, submit feedback, and the messages appear under `/api/sites/:id/messages` and
   in the dashboard.

Example snippet (replace the base URL with your LoopAware deployment and the site identifier with the value returned by the API):

```html
<script defer src="https://loopaware.mprlab.com/widget.js?site_id=6f50b5f4-8a8f-4e4a-9d69-1b2a3c4d5e6f"></script>
```

## Embedding the subscribe form

Each site exposes a subscribe snippet that renders an email capture form and posts subscriptions to `/api/subscriptions`.

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

3. On load, `pixel.js` sends a beacon to `/api/visits` with the site ID, current URL, referrer, and a stable visitor ID
   stored in `localStorage`. Requests from origins outside the site’s `allowed_origin` list are rejected.

For non-JavaScript environments you can fall back to a plain image pixel:

```html
<img src="https://loopaware.mprlab.com/api/visits?site_id=6f50b5f4-8a8f-4e4a-9d69-1b2a3c4d5e6f&url=https%3A%2F%2Fexample.com%2F" alt="" width="1" height="1" />
```

## Development workflow

```bash
make format
make lint
make test
```

The test suite runs entirely in memory using temporary SQLite databases; no external services are required.
Browser-driven integration specs rely on go-rod; if Chromium is not present locally the launcher downloads a sandboxed binary automatically. Screenshots captured during each run are stored under `tests/<date>/<testname>/` for manual inspection.

## Docker

The previous Docker and Compose files remain compatible. Ensure the container receives the OAuth environment variables
and mounts a `config.yaml` containing the admin roster.

```bash
cp configs/.env.loopaware.example configs/.env.loopaware
cp configs/.env.tauth.example configs/.env.tauth
cp configs/.env.pinguin.example configs/.env.pinguin
$EDITOR configs/.env.loopaware configs/.env.tauth configs/.env.pinguin
docker compose up --build --remove-orphans
```

The compose file binds `config.yaml` into the LoopAware container at `/app/config.yaml` and loads per-service environment variables via `env_file` from `configs/.env.*`.
The container now runs as root so the SQLite data volume remains writable; if you need to switch back to an unprivileged
user, update the Docker image to chown the mounted directory before starting the binary.
