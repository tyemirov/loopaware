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

Loopaware is a proprietary software, see [LICENSE](LICENSE) for details.

---
