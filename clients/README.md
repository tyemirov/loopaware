# LoopAware Clients

First-party clients for LoopAware live under this directory. LA Sentry currently has Go, Python, and browser client surfaces.

## LA Sentry

| Runtime | Location | Ingest route | Notes |
| --- | --- | --- | --- |
| Go | [`clients/go/lasentry`](go/lasentry) | `/sentry/errors` | Protected server-side client. |
| Python | [`clients/python/la_sentry`](python/la_sentry) | `/sentry/errors` | Protected server-side client with WSGI and ASGI middleware helpers. |
| Browser | [`clients/browser`](browser) | `/sentry/browser-errors` | Browser harness documentation. The served asset remains `web/la-sentry.js` so the public script URL stays stable. |

Server-side clients require a per-site LA Sentry ingest token from the dashboard. Do not embed that token in browser JavaScript.
