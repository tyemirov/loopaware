# Development and validation

[Project overview](../README.md)

## Prerequisites

Install Go 1.27, Docker with Compose, Node.js 24 or later, npm, Python 3, and `uv`.
Full `make ci` also requires the installed `mprlab-gateway` runtime.
See [publication qualification](npm-publication.md#npm-version-verification) for the local runtime requirements.
The test runner requires a local Docker endpoint and installs the locked browser test dependencies.
Native build prerequisites are in the [deployment guide](deployment.md).

## Development workflow

```bash
make format
make lint
make test
```

The Go suite uses table-driven tests and fast in-memory SQLite fixtures.
The integration suite exercises public APIs and browser behavior.

`make test` runs the Playwright integration suite against `tests/docker-compose.yml`, with test-owned env fixtures under
`tests/configs/`. That stack builds the API image, serves `web/` via gHTTP, and exercises both UI and `/api/*` flows.
Use `make test-resource-theme` to verify theme storage on a resource page through the browser.
This target installs the locked test dependencies when `tests/node_modules` is absent.

Use `make test-resource-page-integration` to verify resource page contracts through the integration stack.

The integration network uses `10.203.242.0/24` by default.
Set `LOOPAWARE_TEST_NETWORK_PREFIX=10.212.34` to select another private test subnet before a test command.
The runner validates this single prefix and uses it for the subnet, proxy address, and trusted proxy.
An invalid prefix stops the runner before Docker operations.

The runner does not select another subnet after a collision.

Use `make test-unit` for Go-only tests and `make test-integration-api` to focus on API specs. Playwright artifacts
(traces, screenshots, videos) land under `tests/test-results/` on failure. The integration runner tears its compose
project down on exit, including failures and signal exits.

The runner rejects inherited `LOOPAWARE_BASE_URL`, `LOOPAWARE_API_BASE_URL`, `LOOPAWARE_ENV_FILE`,
`COMPOSE_PROJECT_NAME`, `DOCKER_HOST`, `DOCKER_CONTEXT`, and Docker TLS inputs. It requires a local `unix://` or `npipe://`
Docker endpoint. It supplies its own localhost frontend, random loopback-only backend port, test env file, and unique
Compose project. The test helper independently rejects any non-loopback frontend or backend target.

Use `make test-live-favicons` when validating customer-site favicon collection against known public websites. That
target performs live network requests and is intentionally outside `make ci` so third-party uptime does not gate normal
development.

Run `make security-audit` to execute the pinned Go vulnerability analyzer, full dependency audits for every npm
project, and the immutable supported-container-base contract. This audit is part of `make ci`. It requires network access
to the Go vulnerability database and npm advisory service. To validate the complete shipped image after a base or Go
dependency update, build it with `docker build --pull -t loopaware:security-audit .` and inspect it with
`docker scout cves --only-severity critical,high --exit-code local://loopaware:security-audit`.

Repository administrators can run the read-only `make github-security-audit` gate.
It verifies live Actions, branch protection, Dependabot, secret-scanning, and gateway-owned Pages controls against the tracked security contract.
