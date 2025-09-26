# Loopaware

A tiny HTTP service to collect user feedback from your site via a lightweight embeddable widget.
Built with Go + Gin, GORM, and Postgres.

## Features

* 💬 Public API to submit feedback
* 🧰 Admin API to create sites and list messages
* 🔒 Admin bearer token auth
* 🚦 Simple per-IP rate limiting on public submissions
* 🧩 Copy-paste `<script>` widget
* 🧪 Fast tests with **embedded Postgres** (no Docker required)
* 🐳 Optional Docker & Docker Compose for local runs

---

## Quick Start (Docker)

```bash
# from repo root
docker compose up --build
```

The API will be available at `http://localhost:8080`.

Default env (see `docker-compose.yml`):

* `APP_ADDR=:8080`
*
`DB_DSN=host=postgres user=feedback_user password=feedback_password dbname=feedback port=5432 sslmode=disable TimeZone=UTC`
* `ADMIN_BEARER_TOKEN=replace-with-long-random` (change this!)

### Create a site (Admin)

```bash
curl -X POST http://localhost:8080/api/admin/sites \
  -H "Authorization: Bearer replace-with-long-random" \
  -H "Content-Type: application/json" \
  -d '{"name":"My Site","allowed_origin":"http://localhost:8080"}'
```

Response:

```json
{
  "id": "…",
  "name": "My Site",
  "allowed_origin": "http://localhost:8080",
  "widget": "<script src=\"http://localhost:8080/widget.js?site_id=…\"></script>"
}
```

### Embed the widget

Paste the `widget` tag into your site HTML (any page on the allowed origin).

## Integrating with your website

Successful integrations start with the admin workflow. Create a site via the admin API and set `allowed_origin` to the exact scheme, host, and optional port where the widget will load. That origin becomes the only third-party domain whose browsers may submit feedback for that site. Each site you configure can target a different partner or product environment by giving it a distinct `allowed_origin` and distributing the generated `<script>` tag to that team.

### Example production setup

Assume your customer-facing app is served from `https://app.example.com` and Loopaware is hosted at `https://feedback.yourcompany.com`.

1. Create a production site:

   ```bash
   curl -X POST https://feedback.yourcompany.com/api/admin/sites \
     -H "Authorization: Bearer <your-admin-token>" \
     -H "Content-Type: application/json" \
     -d '{"name":"Example App Prod","allowed_origin":"https://app.example.com"}'
   ```

2. Add the returned widget `<script>` tag to the pages on `https://app.example.com` where you want the feedback button to appear. The admin API currently renders the `<script>` tag with a `src` rooted at your `allowed_origin` (e.g., `https://app.example.com/widget.js?...`), so ensure that file is served from your site and proxies requests back to Loopaware (`https://feedback.yourcompany.com/widget.js?...`) or copies the script into your own static assets.

3. Double-check the proxied `<script src>` ultimately loads from your production Loopaware domain so the widget posts back to the correct API origin.

4. Verify submissions by triggering the widget on `https://app.example.com`, then list recent messages:

   ```bash
   curl "https://feedback.yourcompany.com/api/admin/sites/<SITE_ID>/messages" \
     -H "Authorization: Bearer <your-admin-token>"
   ```

When working with multiple partners, repeat the process per domain. For example, a partner at `https://partners.example.net` should receive a dedicated site whose `allowed_origin` matches that domain and whose widget script includes that site’s identifier.

### Troubleshooting tips

* Ensure browsers send an `Origin` header that matches the configured `allowed_origin`. Static file hosts and reverse proxies sometimes strip or rewrite headers, which will cause the API to reject requests with `403` errors.
* Confirm the widget is loading from the same Loopaware domain that served the `widget.js` file; mixed environments (e.g., staging widget pointing at production API) will break CORS validation.
* Rotate the admin bearer token regularly and redistribute the updated value to teams calling admin APIs. A stale or revoked token will return `401 Unauthorized` errors when creating sites or listing messages.
* If a page embeds multiple third-party scripts, load Loopaware last to avoid other scripts mutating the DOM container the widget depends on.

### Submit feedback (Public)

```bash
curl -X POST http://localhost:8080/api/feedback \
  -H "Origin: http://localhost:8080" \
  -H "Content-Type: application/json" \
  -d '{"site_id":"<SITE_ID>","contact":"user@example.com","message":"Hello!"}'
```

### List messages (Admin)

```bash
curl "http://localhost:8080/api/admin/sites/<SITE_ID>/messages" \
  -H "Authorization: Bearer replace-with-long-random"
```

---

## Local Development (no Docker)

You need a Postgres DSN. Examples:

```bash
export DB_DSN="host=127.0.0.1 port=5432 user=postgres password=postgres dbname=feedback sslmode=disable TimeZone=UTC"
export ADMIN_BEARER_TOKEN="change-me"

go run ./cmd/server
# -> listening on :8080
```

---

## Testing

Tests spin up an **embedded Postgres** instance in-process (via `github.com/fergusstrange/embedded-postgres`) — no
Docker needed.

```bash
go test ./... -v
```

Notes:

* Each test **process** gets its own ephemeral data/runtime dirs, so multiple packages can run in parallel.
* To serialize packages (optional): `go test -p 1 ./...`

---

## Configuration

Environment variables:

| Name                 | Default  | Description                               |
|----------------------|----------|-------------------------------------------|
| `APP_ADDR`           | `:8080`  | HTTP listen address                       |
| `DB_DSN`             | *(none)* | GORM Postgres DSN                         |
| `ADMIN_BEARER_TOKEN` | *(none)* | Required for all `/api/admin/*` endpoints |

If `ADMIN_BEARER_TOKEN` is empty, admin routes return `503` (disabled).

---

## API

### Public

* `POST /api/feedback`
  Body:

  ```json
  { "site_id": "…", "contact": "email or phone", "message": "text" }
  ```

  Returns `200` on success, with `{ "status": "ok" }`.
  Validates:

    * `site_id`, `contact`, `message` are required
    * `Origin`/`Referer` must match the site’s `allowed_origin` (if set)
    * Basic per-IP rate limiting

* `GET /widget.js?site_id=<SITE_ID>`
  Returns the embeddable widget script.

### Admin (Bearer auth)

* `POST /api/admin/sites`
  Body:

  ```json
  { "name": "My Site", "allowed_origin": "https://example.com" }
  ```

  Returns site info and a ready-made `<script>` tag.

* `GET /api/admin/sites/:id/messages`
  Returns recent messages for a site.

---

## Project Layout

```
.
├── cmd/server/               # App entrypoint
├── internal/
│   ├── httpapi/              # HTTP handlers, middleware, tests
│   ├── model/                # GORM models
│   ├── storage/              # DB open/migrate helpers + tests
│   └── testutil/             # Embedded Postgres bootstrap (tests)
├── docker-compose.yml
├── Dockerfile
├── go.mod / go.sum
└── .github/workflows/ci.yml  # Go build/vet/test on PRs
```

---

## Development Tips

* Prefer a **long, random** `ADMIN_BEARER_TOKEN` in any environment.
* `allowed_origin` must match exactly (scheme + host + optional port).
  Example: `http://localhost:3000` is different from `http://localhost:8080`.
* The widget auto-detects its host from the `<script src="...">` and posts to the same origin.

---

## CI

GitHub Actions builds, vets, and runs tests:

* `go build ./...`
* `go vet ./...`
* `go test ./... -v -race -count=1`

---

## License

Loopaware is proprietary software, see [LICENSE](LICENSE) for details.

---
