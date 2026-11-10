# Setup and configuration

[Project overview](../README.md)
Run all repository commands from its root directory.
Local setup requires Docker Compose, Go 1.27, Google OAuth configuration, TAuth tenant inputs, and Pingwin email configuration.
The browser frontend requires network access to its CDN dependencies.

## Local startup

1. Clone the repository.

   ```bash
   git clone https://github.com/tyemirov/loopaware.git
   cd loopaware
   ```

2. Create the four private configuration files in the Configuration section below.
3. Supply each placeholder in the three service YAML files.
4. Configure the LoopAware Google client and tenant for `http://localhost:8080`.
5. Set `TAUTH_CONFIG_FILE=/config/config.yml` and `PINGUIN_CONFIG_PATH=/config/config.yml` in their service input files.
6. Set gHTTP to serve `/data` on port `8080`.
7. Configure gHTTP routes for TAuth, `/public/`, `/sentry/`, `/api/`, and `/healthz`.
8. Run `make config-audit` to validate configuration.
9. Start the local stack.

   ```bash
   ./scripts/up.sh local
   ```

10. Open `http://localhost:8080/login` in a browser.
11. Complete the [first-feedback procedure](../README.md#get-your-first-feedback).

A failed audit identifies the invalid configuration or missing input.
Correct that input before startup.
The startup command keeps the service logs in the current terminal.
Use another terminal for other commands.

The mounted YAML files define the required service inputs:

- [`configs/config.loopaware.yml`](../configs/config.loopaware.yml) for LoopAware.
- [`configs/config.tauth.yml`](../configs/config.tauth.yml) for TAuth tenants and Google clients.
- [`configs/config.pinguin.yml`](../configs/config.pinguin.yml) for Pingwin and email delivery.

For local gHTTP, configure these values in `configs/.env.ghttp`:

```dotenv
GHTTP_SERVE_DIRECTORY=/data
GHTTP_SERVE_PORT=8080
GHTTP_SERVE_LOGGING_TYPE=JSON
GHTTP_SERVE_PROXIES=/tauth.js=http://la-tauth:8082,/me=http://la-tauth:8082,/auth/=http://la-tauth:8082,/public/=http://loopaware:8080,/sentry/=http://loopaware:8080,/api/=http://loopaware:8080,/healthz=http://loopaware:8080/healthz
```

## Local service alignment

Use these non-secret LoopAware values for the local Compose topology:

| Input | Local value |
| --- | --- |
| `APP_ADDR` | `:8080` |
| `PUBLIC_BASE_URL` | `http://localhost:8080` |
| `TRUSTED_PROXY_CIDR` | `172.30.240.10/32` |
| `DB_DRIVER` | `sqlite` |
| `DB_DSN` | `file:/app/data/loopaware.sqlite?_foreign_keys=on` |
| `TAUTH_BASE_URL` | `http://la-tauth:8082` |
| `PINGUIN_ADDR` | `la-pinguin:50051` |

Keep the three LoopAware authentication inputs consistent with their TAuth tenant values:

| LoopAware input | TAuth input |
| --- | --- |
| `TAUTH_TENANT_ID` | `TAUTH_TENANT_ID_LOOPAWARE` |
| `TAUTH_JWT_SIGNING_KEY` | `TAUTH_TENANT_JWT_SIGNING_KEY_LOOPAWARE` |
| `TAUTH_SESSION_COOKIE_NAME` | `TAUTH_TENANT_SESSION_COOKIE_NAME_LOOPAWARE` |

Set the local Google provider's `clientId` in `web/config-ui.yaml` to your `TAUTH_TENANT_GOOGLE_WEB_CLIENT_ID_LOOPAWARE` value.
Configure the Google client for the local frontend origin.
Configure the tenant's allowed origins and HTTP cookie settings for that origin.
The tracked browser configuration contains the MPR Lab Google client ID.
Supply your own client configuration for your service.

Configure Pingwin's bearer token and email delivery for the LoopAware tenant.
The public collection forms use Pingwin for confirmation messages and notifications.
See the mounted service YAML files for their complete input lists.

## Configuration

Tracked LoopAware product configuration lives under `configs/`. Its `.env.*.example` files document variable names with
intentionally unusable values. They are not templates or runtime inputs.

The directory also holds service config templates
and the LoopAware backend runtime config consumed by local Docker Compose and mounted by the declared production service. Real local `configs/.env.*`
files are intentionally ignored and must be created explicitly, never copied or sourced from the examples. Keep local service inputs in `configs/`.
Remove obsolete repo-root duplicates after you move their assignments. Test-only Compose files and the
canonical config-audit environment fixtures belong under `tests/`, not `configs/`.

### 1. Backend runtime config (`configs/config.loopaware.yml`)

LoopAware reads one backend runtime YAML file through `github.com/tyemirov/utils/runtimeconfig`. The selected file is parsed strictly, shell placeholders are expanded during that parse, and the server consumes the populated typed config after validation. Runtime values are not read from environment variables or config flags after parsing.

```yaml
analytics:
  aggregate_retention_days: 90
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

The local Compose env files provide shell values for placeholders used by `configs/config.loopaware.yml`. The YAML file remains the only runtime config source.

| Variable               | Required | Description                                                 |
|------------------------|----------|-------------------------------------------------------------|
| `APP_ADDR`             | ✅        | Listen address, for example `:8080`                         |
| `PUBLIC_BASE_URL`      | ✅        | Frontend origin used for CORS and subscription links        |
| `TRUSTED_PROXY_CIDR`   | ✅        | Canonical CIDR of the reverse proxy allowed to set client forwarding headers |
| `DB_DRIVER`            | ✅        | Storage driver (`sqlite`)                             |
| `DB_DSN`               | ✅        | Driver-specific DSN                                         |
| `SESSION_SECRET`       | ✅        | 32+ byte secret for subscription confirmation tokens        |
| `TAUTH_BASE_URL`       | ✅        | Base URL for the TAuth API                                  |
| `TAUTH_TENANT_ID`      | ✅        | Tenant identifier configured in TAuth                       |
| `TAUTH_JWT_SIGNING_KEY`| ✅        | JWT signing key used to validate `app_session`              |
| `TAUTH_SESSION_COOKIE_NAME` | ✅   | Session cookie name set by TAuth                            |
| `PINGUIN_ADDR`         | ✅        | Pingwin gRPC address                                        |
| `PINGUIN_AUTH_TOKEN`   | ✅        | Bearer token passed to the Pingwin gRPC service             |
| `PINGUIN_TENANT_ID`    | ✅        | Tenant identifier used when calling the Pingwin gRPC API     |

Secrets can remain outside the tracked file with placeholders. Non-secret settings can be literal YAML values when that is clearer for the environment.

`server.trusted_proxy_cidrs` must contain canonical CIDRs and cannot contain a catch-all network. Gin accepts
`X-Forwarded-For` only when the immediate peer is inside that boundary. Forwarding headers from every other peer are
removed before logging, HTTPS detection, storage, or rate limiting. `server.trusted_edge_geo_proxy_cidrs` is a separate
subset of the proxy boundary. Keep it empty unless the named owned proxy overwrites caller geo headers with canonical
values.

The tracked local, test, and ComputerCat stacks give gHTTP a fixed address and trust only that address. The
production manifest trusts the gateway-owned internal network because Caddy is the only public path to the API service.

When running via Docker Compose, create private local `.env.*` files explicitly. The tracked examples document variable names only. Their values are intentionally unusable and must never be copied, sourced, or passed to Compose.

```bash
touch configs/.env.loopaware configs/.env.tauth configs/.env.pinguin configs/.env.ghttp
$EDITOR configs/.env.loopaware configs/.env.tauth configs/.env.pinguin configs/.env.ghttp
```

Pingwin and LoopAware must share the same bearer secret. Set Pingwin's `GRPC_AUTH_TOKEN` and LoopAware's `PINGUIN_AUTH_TOKEN` to identical values in their respective service env files.
`configs/.env.tauth` must set `TAUTH_CONFIG_FILE=/config/config.yml`, and `configs/.env.pinguin` must set
`PINGUIN_CONFIG_PATH=/config/config.yml`, matching the files mounted by Docker Compose. `configs/.env.loopaware` must
define every placeholder referenced by `configs/config.loopaware.yml`. Those values are expanded only while parsing the
YAML file. They are not a second runtime config source.

Production uses the separate ignored `.mprlab/deploy/.env` input and the manifest's `environment.private_inputs` bindings.
See [deployment](deployment.md) for the lifecycle input contract.

Gateway resolves this selected file and caller values during manifest processing for each lifecycle phase.
`config-audit` validates real local env files when they are present. When they are absent, each production Compose service
declares a tracked audit-only fixture under `tests/configs/` with `x-config-audit-env-file`. `.env.*.example` files are never
loaded. Runtime requires the real local env files.

Tracked examples and audit fixtures are documentation and audit inputs only.
A source-only checkout can omit private runtime inputs.

The local configuration audit uses `tests/configs/loopaware.local.env` only for `config-audit`.
The runtime input remains `configs/.env.loopaware`.
The computercat audit uses `tests/configs/loopaware.computercat.env` only for `config-audit`.
Its `192.0.2.10/32` trusted proxy is a documentation address.

The computercat runtime input remains `configs/.env.loopaware.computercat`.

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
Make sure TAuth runs at `auth.tauth.base_url` with the tenant from `auth.tauth.tenant_id`.
Administrators listed in `configs/config.loopaware.yml` can manage every site. Other users see their owned or created sites and sites with their email assigned as a team member.

The static frontend uses literal `@latest` CDN URLs for the three MPR UI assets.
MPR UI owns browser authentication and protected management requests.
The provider config uses `auth.providers` in each environment.
The application waits for the shared bundle and authentication before it requests account data.

The shared transport recovers the session after an unauthorized response.
The API authorizes each protected request before domain work, which permits one mutation replay.

The inactivity logout flow reads the current header `auth-config` for its TAuth request.
Other external JavaScript and CSS declarations keep pinned versions and SHA-384 subresource integrity.
The MPR UI declarations omit fixed integrity values because their URLs select the current shared release.
See [shared UI migration](mpr-ui-migration.md) for candidate evidence and the central I009 dependency.

Every HTML entry point
declares the canonical `configs/content-security-policy.txt` static-hosting CSP before executable or styled content. The local proxies enforce the same policy
as a response header and add `frame-ancestors 'none'`, which CSP metadata cannot enforce. Run
`make browser-security-audit` to detect dependency, integrity, CSP, or proxy-policy drift. Do not copy third-party
browser bundles into `web/`. Non-CDN frontend dependencies are forbidden by architecture.

## Authentication flow

1. Users visit `/login` (automatic redirect from protected routes).
2. `mpr-ui` drives the browser sign-in lifecycle against the configured TAuth tenant.
3. TAuth issues and refreshes the session cookie configured by `auth.tauth.session_cookie_name`.
4. `api.AuthManager` validates the session with TAuth's verifier, injects user details into the request context, and enforces admin,
   owner, or team-member site access.
5. The dashboard and JSON APIs consume the authenticated context.

## Static frontend

LoopAware's static frontend lives in `web/`.
Production publishes it through GitHub Pages.
Local stacks serve it through gHTTP.
The frontend contains these pages and assets:

- `/login` — landing page with shared `mpr-ui`/TAuth sign-in.
- `/resources` — crawlable public resource index with focused product and use-case pages.
- `/privacy` — static privacy policy linked from the landing and dashboard footers.
- `/app` — dashboard shell (data loaded via `/api/*`).
- `/subscriptions/confirm` and `/subscriptions/unsubscribe` — email link pages.
- `/widget.js`, `/subscribe.js`, `/pixel.js`, `/count.js`, `/la-sentry.js` — embeddable JavaScript assets.

The API hostname serves `/sentry/errors` for protected server ingest and `/sentry/browser-errors` for origin-bound browser ingest.

The repository does not vendor third-party browser dependencies into `web/`. External JavaScript and CSS, including UI
libraries, use CDN URLs.
MPR UI uses the shared `@latest` release. Other dependencies use pinned versions. Any browser dependency that is not delivered by CDN is
forbidden. `web/` is reserved for LoopAware-authored assets only, so deployments, cache behavior, and browser tests
exercise the same delivery path used in production.

Set `PUBLIC_BASE_URL` to the frontend origin for API links and credentialed CORS.
Configure frontend hostnames and service origins in `web/config.yml`.
`web/runtime-env.js` loads that file at runtime.
The dashboard uses its configured API origin.

The shared browser sign-in uses `web/config-ui.yaml` for its TAuth origin and provider settings.

The production hostnames are:

| Surface | Hostname |
| --- | --- |
| Static frontend | `https://loopaware.mprlab.com` |
| LoopAware API | `https://loopaware-api.mprlab.com` |
| TAuth | `https://tauth-api.mprlab.com` |

Local development uses gHTTP to serve static files and forward service routes on one origin.
LoopAware-owned embed hosts map their API origin internally.
For split-origin widget, subscription, pixel, and Sentry embeds, supply `data-api-origin` or the `api_origin` query parameter.
`count.js` uses a separate origin rule.
See [daily counts](integrations.md#daily-counts) before you configure it on another hostname.

Canonical SEO metadata, Open Graph URLs, `robots.txt`, and `sitemap.xml` are fixed to the single public site
`https://loopaware.mprlab.com` and are not environment-specific. The public resource hub lives at `https://loopaware.mprlab.com/resources/`.
It links guides for feedback, subscribers, analytics, uptime, access control, self-hosting, and LA Sentry.
Each environment can define `services.siteWidgetSiteId` for the first-party feedback widget on `/login` and `/app`.
This setting keeps the site UUID outside the static HTML.

## Docker

Make sure the container receives the placeholder inputs used by `configs/config.loopaware.yml` and mounts that backend runtime config file.

```bash
touch configs/.env.loopaware configs/.env.tauth configs/.env.pinguin configs/.env.ghttp
$EDITOR configs/.env.loopaware configs/.env.tauth configs/.env.pinguin configs/.env.ghttp
./scripts/up.sh
```

The compose file binds `configs/config.loopaware.yml` into the LoopAware container at `/app/configs/config.loopaware.yml`
and loads per-service placeholder values via `env_file` from `configs/.env.*`.
The production Dockerfile pins the Go 1.27.3/Alpine 3.24 build image and Alpine 3.24 runtime image by multi-architecture manifest-list digests.
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
touch configs/.env.loopaware.computercat configs/.env.tauth.computercat configs/.env.pinguin.computercat configs/.env.ghttp.computercat
$EDITOR configs/.env.loopaware.computercat configs/.env.tauth.computercat configs/.env.pinguin.computercat configs/.env.ghttp.computercat
./scripts/up.sh computercat
```

The computercat stack exposes `https://computercat.tyemirov.net:4443` through gHTTP as the TLS terminator and reverse
proxy. The proxy expects these certificates:

- `/media/share/Drive/exchange/certs/computercat/computercat-cert.pem`
- `/media/share/Drive/exchange/certs/computercat/computercat-key.pem`

Compose mounts only these two files into `/certs`.
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
reverse proxy for both LoopAware and TAuth. The documentation-only examples intentionally do not provide runnable defaults.
TAuth requires HTTPS for secure cookies when
`allow_insecure_http=false`. GHTTP’s reverse proxy does not currently set `X-Forwarded-Proto`, so keep
`TAUTH_ALLOW_INSECURE_HTTP=true` unless a fronting proxy forwards `X-Forwarded-Proto=https`.
