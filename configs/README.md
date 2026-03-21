# Docker Compose configuration

This directory is the source of truth for tracked LoopAware configuration.

It holds local `.env.*` files, service config templates, the LoopAware admin roster, and the frontend runtime config
source consumed by `docker compose` and deployment workflows.

Notes:

- `configs/.env.*` files are intentionally gitignored. Create them locally.
- `configs/.env.*.example` files are tracked templates; copy them into `configs/.env.*`.
- Legacy repo-root `.env.*` files are unsupported duplicates. Move any remaining values into `configs/.env.*` and delete the root copies.
- There is no supported plain `configs/.env.ghttp` file. Under `configs/`, the only supported gHTTP env file is `configs/.env.ghttp.computercat`; test-only proxy fixtures live under `tests/`.
- `config-audit` validates tracked `.env.*.example` templates when local `.env.*` files are absent. Runtime still requires the real `.env.*` files.
- `configs/config.loopaware.yml` is the tracked LoopAware admin-roster config used by the server `--config` flag.
- `configs/config.frontend.yml` is the tracked frontend runtime config source; deployments publish it as `/config.yml`.
  It also carries per-environment frontend service settings such as `siteWidgetSiteId` for the first-party landing/dashboard widget.
- Test-only compose files and env fixtures do not belong in `configs/`; keep them under `tests/`.

## Local compose (`docker-compose.yml`)

Start and stop the local stack only through the helper scripts:

```bash
./up.sh
./down.sh
```

With no argument the scripts open an interactive selector; use `./up.sh local` and `./down.sh local` when you need an explicit target.

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
- `docker-compose.yml` mounts `configs/config.loopaware.yml` into the LoopAware container at `/app/configs/config.loopaware.yml`.

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
GHTTP_SERVE_DIRECTORY=/data
GHTTP_SERVE_PORT=4443
GHTTP_SERVE_LOGGING_TYPE=JSON
GHTTP_SERVE_TLS_CERTIFICATE=/certs/computercat-cert.pem
GHTTP_SERVE_TLS_PRIVATE_KEY=/certs/computercat-key.pem
GHTTP_SERVE_PROXIES=/tauth.js=http://la-tauth:8082,/me=http://la-tauth:8082,/auth/=http://la-tauth:8082,/public/=http://loopaware-api:8080,/api/=http://loopaware-api:8080
```

### Service env updates

The computercat templates default to the public origin `https://computercat.tyemirov.net:4443` so the browser uses the reverse proxy for both LoopAware and TAuth.
Before starting the proxy stack, publish the tracked frontend runtime config into `web/`:

```bash
cp configs/config.frontend.yml web/config.yml
```

Start and stop the computercat stack only through the helper scripts:

```bash
./up.sh computercat
./down.sh computercat
```

`up.sh computercat` does the config publish step automatically before invoking `docker compose`.

TAuth requires HTTPS for secure cookies when `allow_insecure_http=false`. gHTTP’s reverse proxy does not currently set `X-Forwarded-Proto`,
so keep `TAUTH_ALLOW_INSECURE_HTTP=true` unless you front TAuth with a proxy that forwards `X-Forwarded-Proto=https`.
