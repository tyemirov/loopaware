# LoopAware LA Sentry Browser Harness

The browser harness is documented here with the other first-party clients, but the served asset remains `web/la-sentry.js` so existing public script URLs continue to work.

```html
<script defer src="https://loopaware.mprlab.com/la-sentry.js?site_id=YOUR_SITE_ID&environment=production&release=2026.04.24"></script>
```

The script installs `window.LASentry.captureError(error, attrs)` and `window.LASentry.captureMessage(message, attrs)`. It also captures uncaught `error` and `unhandledrejection` events by default.

Browser events do not use the server-side ingest token. They post to `/sentry/browser-errors`, are accepted only from configured site origins, and are rate-limited by client IP.
