# Docker Compose configuration

This directory holds local `.env.*` files and service config templates consumed by `docker compose`.

Notes:

- `configs/.env.*` files are intentionally gitignored. Create them locally.
- `configs/.env.*.example` files are tracked templates; copy them into `configs/.env.*`.
- GitHub Actions CI writes minimal env fixtures for `make ci` (see `.github/workflows/ci.yml`).

## Local compose (`docker-compose.yml`)

Create these env files:

- `configs/.env.loopaware`
- `configs/.env.tauth`
- `configs/.env.pinguin`

Copy from the tracked templates:

```bash
cp configs/.env.loopaware.example configs/.env.loopaware
cp configs/.env.tauth.example configs/.env.tauth
cp configs/.env.pinguin.example configs/.env.pinguin
```

Required config file pointers:

- `configs/.env.tauth` must set `TAUTH_CONFIG_FILE=/config/config.yml` (Compose mounts `configs/config.tauth.yml` at that path).
- `configs/.env.pinguin` must set `PINGUIN_CONFIG_PATH=/config/config.yml` (Compose mounts `configs/config.pinguin.yml` at that path).

## computercat TLS compose (`docker-compose.computercat.yml`)

This variant exposes only `https://computercat.tyemirov.net:4443` and uses `ghttp` as the TLS terminator + reverse proxy (no nginx).

### Certificates

The compose file mounts the host directory `/media/share/Drive/exchange/certs/computercat` into the proxy container at `/certs`.
Expected files:

- `/media/share/Drive/exchange/certs/computercat/computercat-cert.pem`
- `/media/share/Drive/exchange/certs/computercat/computercat-key.pem`

### Proxy env (`configs/.env.ghttp`)

Create `configs/.env.ghttp`:

```dotenv
GHTTP_SERVE_PORT=4443
GHTTP_SERVE_LOGGING_TYPE=JSON
GHTTP_SERVE_TLS_CERTIFICATE=/certs/computercat-cert.pem
GHTTP_SERVE_TLS_PRIVATE_KEY=/certs/computercat-key.pem
GHTTP_SERVE_PROXIES=/tauth.js=http://la-tauth:8082,/me=http://la-tauth:8082,/auth/=http://la-tauth:8082,/=http://loopaware:8080
```

You can also copy the tracked template:

```bash
cp configs/.env.ghttp.example configs/.env.ghttp
```

### Service env updates

Update the existing service env files so browser traffic uses the public origin:

- `configs/.env.loopaware`: set `TAUTH_BASE_URL` and `PUBLIC_BASE_URL` to `https://computercat.tyemirov.net:4443`.
- `configs/.env.tauth`: set `TAUTH_CORS_ORIGIN_*`, `TAUTH_TENANT_ORIGIN_*`, and `TAUTH_COOKIE_DOMAIN=computercat.tyemirov.net` as needed.
- `configs/.env.pinguin`: set `TAUTH_BASE_URL` and any LoopAware domain settings used in notifications to `https://computercat.tyemirov.net:4443`.

TAuth requires HTTPS for secure cookies when `allow_insecure_http=false`. gHTTP’s reverse proxy does not currently set `X-Forwarded-Proto`,
so keep `TAUTH_ALLOW_INSECURE_HTTP=true` unless you front TAuth with a proxy that forwards `X-Forwarded-Proto=https`.
