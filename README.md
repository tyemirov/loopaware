# LoopAware

LoopAware collects customer feedback through a lightweight widget, authenticates operators with Google, and offers a
role-aware dashboard for managing sites and messages.

## Highlights

- Google OAuth 2.0 authentication via [GAuss](https://github.com/temirov/GAuss)
- Role-aware dashboard (`/app`) with admin and owner scopes
- YAML configuration for privileged accounts (`config.yaml`)
- REST API to create, update, and inspect sites and feedback
- Embeddable JavaScript widget with strict origin validation
- SQLite-first storage with pluggable drivers
- Table-driven tests and fast in-memory SQLite fixtures

## Configuration

### 1. Admin roster (`config.yaml`)

Create a YAML file next to the binary with the email addresses that should receive administrator privileges:

```yaml
admins:
  - temirov@gmail.com
```

LoopAware loads the file specified by `--config` (default `config.yaml`) before starting the HTTP server.

### 2. Environment variables

| Variable               | Required | Description                                                 |
|------------------------|----------|-------------------------------------------------------------|
| `GOOGLE_CLIENT_ID`     | ✅        | OAuth client ID from Google Cloud Console                   |
| `GOOGLE_CLIENT_SECRET` | ✅        | OAuth client secret                                         |
| `SESSION_SECRET`       | ✅        | 32+ byte secret for cookie signing                          |
| `PUBLIC_BASE_URL`      | ⚙️       | Public URL of the service (default `http://localhost:8080`) |
| `APP_ADDR`             | ⚙️       | Listen address (default `:8080`)                            |
| `DB_DRIVER`            | ⚙️       | Storage driver (`sqlite`, etc.)                             |
| `DB_DSN`               | ⚙️       | Driver-specific DSN                                         |

Secrets must come from the environment; only non-sensitive settings belong in `config.yaml`.

Copy the provided template and edit the values before running the service or Docker Compose stack:

```bash
cp .env.sample .env
$EDITOR .env
```

### 3. Flags

All configuration options are also exposed as Cobra flags:

```
loopaware --config=config.yaml \
  --app-addr=:8080 \
  --db-driver=sqlite \
  --db-dsn="file:loopaware.sqlite?_foreign_keys=on" \
  --google-client-id=$GOOGLE_CLIENT_ID \
  --google-client-secret=$GOOGLE_CLIENT_SECRET \
  --session-secret=$SESSION_SECRET \
  --public-base-url=https://feedback.example.com
```

Flags are optional when the equivalent environment variables are set.

## Running locally

```bash
GOOGLE_CLIENT_ID=... \
GOOGLE_CLIENT_SECRET=... \
SESSION_SECRET=$(openssl rand -hex 32) \
go run ./cmd/server --config=config.yaml
```

Open `http://localhost:8080/app` to trigger Google Sign-In. Administrators listed in `config.yaml` can manage every
site; other users see only the sites assigned to their Google account.

## Authentication flow

1. Users visit `/login` (automatic redirect from protected routes).
2. GAuss handles OAuth and stores the session in an encrypted cookie.
3. `httpapi.AuthManager` reads the session, injects user details into the request context, and enforces admin / owner
   access.
4. The dashboard and JSON APIs consume the authenticated context.

## REST API

All authenticated endpoints live under `/api` and require the GAuss session cookie. JSON responses include Unix
timestamps in seconds.

| Method  | Path                      | Role        | Description                                                               |
|---------|---------------------------|-------------|---------------------------------------------------------------------------|
| `GET`   | `/api/me`                 | any         | Current account metadata (email, name, `is_admin`)                        |
| `GET`   | `/api/sites`              | any         | Sites visible to the caller (admin = all, user = owned)                   |
| `POST`  | `/api/sites`              | admin       | Create a site (requires `name`, `allowed_origin`, `owner_email`)          |
| `PATCH` | `/api/sites/:id`          | owner/admin | Update name/origin; admins may reassign ownership                         |
| `GET`   | `/api/sites/:id/messages` | owner/admin | List feedback messages (newest first)                                     |
| `POST`  | `/api/feedback`           | public      | Submit feedback (requires JSON body with `site_id`, `contact`, `message`) |
| `GET`   | `/widget.js`              | public      | Serve embeddable JavaScript widget                                        |

## Dashboard (`/app`)

The Bootstrap front end consumes the APIs above. Features include:

- Account card with avatar, email, and role badge
- Admin-only controls to create sites and reassign ownership
- Owner/admin editor for site metadata
- Feedback table with human-readable timestamps
- Logout button (links to `/logout`)

The dashboard automatically redirects unauthenticated visitors to `/login`.

## Embedding the widget

1. Create a site (admin) and copy the generated `<script>` tag from the API response.
2. Embed the script on any page served from the configured `allowed_origin`.
3. Visitors can open the floating bubble, submit feedback, and the messages appear under `/api/sites/:id/messages` and
   in the dashboard.

Example snippet (replace the base URL with your LoopAware deployment and the site identifier with the value returned by the API):

```html
<script src="https://loopaware.mprlab.com/widget.js?site_id=6f50b5f4-8a8f-4e4a-9d69-1b2a3c4d5e6f"></script>
```

## Development workflow

```bash
go fmt ./...
go vet ./...
go test ./...
```

The test suite runs entirely in memory using temporary SQLite databases; no external services are required.

## Docker

The previous Docker and Compose files remain compatible. Ensure the container receives the OAuth environment variables
and mounts a `config.yaml` containing the admin roster.

```bash
cp .env.sample .env
$EDITOR .env             # fill in real secrets
docker compose up --build --remove-orphans
```

The compose file binds `config.yaml` into the container at `/app/config.yaml` and loads environment variables from `.env`.
The container now runs as root so the SQLite data volume remains writable; if you need to switch back to an unprivileged
user, update the Docker image to chown the mounted directory before starting the binary.
