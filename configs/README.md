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

### Computercat env (`configs/.env.*.computercat`)

Copy the tracked templates:

```bash
cp configs/.env.loopaware.computercat.example configs/.env.loopaware.computercat
cp configs/.env.tauth.computercat.example configs/.env.tauth.computercat
cp configs/.env.pinguin.computercat.example configs/.env.pinguin.computercat
cp configs/.env.ghttp.computercat.example configs/.env.ghttp.computercat
```

Edit the copied files and replace placeholder secrets (Google client ID, signing keys, shared bearer token, etc.).

### Proxy config

The `ghttp` container reads TLS + reverse-proxy settings from `configs/.env.ghttp.computercat`:

```dotenv
GHTTP_SERVE_PORT=4443
GHTTP_SERVE_LOGGING_TYPE=JSON
GHTTP_SERVE_TLS_CERTIFICATE=/certs/computercat-cert.pem
GHTTP_SERVE_TLS_PRIVATE_KEY=/certs/computercat-key.pem
GHTTP_SERVE_PROXIES=/tauth.js=http://la-tauth:8082,/me=http://la-tauth:8082,/auth/=http://la-tauth:8082,/api/=http://loopaware-api:8080,/widget.js=http://loopaware-api:8080,/subscribe.js=http://loopaware-api:8080,/pixel.js=http://loopaware-api:8080,/app/sites/=http://loopaware-api:8080,/subscriptions/=http://loopaware-api:8080,/subscribe-demo=http://loopaware-api:8080,/=http://loopaware-web:8080
```

### Service env updates

The computercat templates default to the public origin `https://computercat.tyemirov.net:4443` so the browser uses the reverse proxy for both LoopAware and TAuth.

TAuth requires HTTPS for secure cookies when `allow_insecure_http=false`. gHTTP’s reverse proxy does not currently set `X-Forwarded-Proto`,
so keep `TAUTH_ALLOW_INSECURE_HTTP=true` unless you front TAuth with a proxy that forwards `X-Forwarded-Proto=https`.
