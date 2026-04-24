# LoopAware LA Sentry Python Client

This package submits Python service exceptions to LoopAware's protected LA Sentry ingest endpoint without depending on the commercial Sentry SDK.

```python
from la_sentry import Client, LASentryConfig

client = Client(LASentryConfig(
    endpoint="https://loopaware.mprlab.com/sentry/errors",
    site_id="YOUR_SITE_ID",
    ingest_token="YOUR_SERVER_SIDE_TOKEN",
    environment="production",
    release="2026.04.24",
    default_tags={"service": "checkout"},
))

try:
    raise RuntimeError("payment capture failed")
except RuntimeError as error:
    client.capture_error(error, {"tags": {"queue": "primary"}})
```

Tokens must stay server-side. Do not embed the token in browser JavaScript.

## Middleware

```python
from la_sentry import Client, LASentryConfig, WSGILASentryMiddleware

client = Client(LASentryConfig(
    endpoint="https://loopaware.mprlab.com/sentry/errors",
    site_id="YOUR_SITE_ID",
    ingest_token="YOUR_SERVER_SIDE_TOKEN",
    environment="production",
))

application = WSGILASentryMiddleware(application, client)
```

ASGI apps can use `ASGILASentryMiddleware(app, client)` with the same client configuration.
