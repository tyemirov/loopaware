# Collection clients and integrations

[Project overview](../README.md)

## Client selection

Use a separate collector for each feature:

| Feature | Browser client | Other clients |
| --- | --- | --- |
| Feedback | `widget.js` | React Native feedback SDK or the mobile feedback REST endpoint. |
| Subscriptions | `subscribe.js` | Subscription REST endpoint. |
| Detailed traffic | `pixel.js` | Image pixel. |
| Aggregate counts | `count.js` | Native count REST endpoint. |
| Application errors | `la-sentry.js` | Go and Python LA Sentry clients. |

The JavaScript assets use the frontend hostname.
Direct REST calls use the API hostname.
For the public service, these are `https://loopaware.mprlab.com` and `https://loopaware-api.mprlab.com`.
For another deployment, use its configured origins.

## Embedding the widget

1. Sign in, create a site, and copy its feedback snippet from the dashboard or API response.
2. Embed the script on a page served from a configured `allowed_origin`.
   Separate multiple origins with spaces or commas. Include the `defer` attribute so the widget loads without blocking the page. The script waits for the body before rendering the UI.
3. Open the feedback bubble and submit a valid email or phone number with a message or sentiment.
   Submissions appear under `/api/sites/:id/messages` and in the dashboard.

Example snippet (replace the base URL with your LoopAware deployment and the site identifier with the value returned by the API):

```html
<script defer src="https://loopaware.mprlab.com/widget.js?site_id=6f50b5f4-8a8f-4e4a-9d69-1b2a3c4d5e6f"></script>
```

## Adding mobile feedback

React Native and Expo apps can use the first-party client under `clients/react-native` for feedback buttons on selected screens.
See the [client guide](../clients/react-native/README.md) for imports and complete application examples. Mobile feedback is separate from LA Sentry error capture.

Install the package from npm:

```bash
npm install @loopaware/react-native
```

1. Register the native app for the site:

   ```json
   POST /api/sites/6f50b5f4-8a8f-4e4a-9d69-1b2a3c4d5e6f/mobile-apps
   {
     "platform": "ios",
     "app_identifier": "com.example.app",
     "display_name": "Example iOS"
   }
   ```

2. Store the returned public `client_id` in the app configuration.
3. Render the feedback button on screens that need customer feedback:

   ```tsx
   <LoopAwareProvider
     siteId="6f50b5f4-8a8f-4e4a-9d69-1b2a3c4d5e6f"
     mobileClientId="client-id-from-dashboard-api"
     apiOrigin="https://loopaware-api.mprlab.com"
     app={{
       platform: "ios",
       applicationId: "com.example.app",
       version: "1.2.3",
       build: "44",
       environment: "production",
     }}
   >
     <CheckoutScreen />
     <LoopAwareFeedbackButton
       screen={{ name: "Checkout", path: "/checkout/payment" }}
       context={{ step: "payment", plan: "pro" }}
     />
   </LoopAwareProvider>
   ```

The public `client_id` identifies the app registration. It is not a secret. Mobile submissions validate the client ID, platform, and application identifier against the registered site app.
They store the supplied screen, app version, and bounded context with the feedback message.

For Swift, Kotlin, and other native apps, register the app with `/api/sites/:id/mobile-apps`.
Implement the same REST contract directly against `/public/mobile-feedback`.
The repository supplies a React Native SDK. Other native clients use REST directly.

## Embedding the subscribe form

Each site exposes a subscribe snippet that renders an email capture form and posts subscriptions to `/public/subscriptions`.

1. In the dashboard, select a site and use the Subscribers panel to copy the subscribe snippet.
2. Embed the script on pages served from any of the site’s `allowed_origin` entries. The basic form looks like:

   ```html
   <script defer src="https://loopaware.mprlab.com/subscribe.js?site_id=6f50b5f4-8a8f-4e4a-9d69-1b2a3c4d5e6f"></script>
   ```

3. Optional query parameters let you adjust behavior and styling:
   - `mode=inline` (default) or `mode=bubble` for a floating button.
   - `accent=#0d6efd` to override the accent color.
   - `cta=Subscribe` to customize the button text.
   - `success=You%27re+on+the+list%21` and `error=Please+try+again.` for inline messages.
   - `name_field=false` to hide the optional name field.

The form enforces the site’s `allowed_origin` list with request headers and `source_url`.
It shows success or error messages on the same page.

## Embedding the traffic pixel

The traffic pixel records page visits per site and powers the dashboard Traffic card and top-pages table.

1. In the dashboard, select a site and use the Traffic panel to copy the pixel snippet.
2. Embed the script on every page served from any of the site’s `allowed_origin` entries:

   ```html
   <script defer src="https://loopaware.mprlab.com/pixel.js?site_id=6f50b5f4-8a8f-4e4a-9d69-1b2a3c4d5e6f"></script>
   ```

3. On load, `pixel.js` sends a request to `/public/visits` with the site ID and browser context.
   The context includes the URL, referrer, timezone, locale, viewport, screen resolution, and a persistent visitor ID from `localStorage`.
   The server also stores the client IP and user agent with each detailed visit. The server accepts Cloudflare, Vercel, and CloudFront edge geo headers only when an explicitly trusted owned proxy overwrites them.
   It then prefers that location signal over browser timezone and locale hints. The tracked deployments
   reject caller-supplied edge geo metadata and use browser timezone or locale instead. Requests from origins outside the site’s `allowed_origin`
   list are rejected. Traffic from known bot user-agent signatures is stored but excluded from default dashboard totals,
   top-page rankings, trends, attribution, engagement, devices, and locations.

The collector stores the `referrer` query value as the page referrer. An empty value means no reported referral.
The HTTP `Referer` header identifies the page that sends the pixel request. It does not identify the visitor source.

Campaign parameters in the page URL supply source, medium, and campaign attribution.

For a page without JavaScript, use a plain image pixel:

```html
<img src="https://loopaware-api.mprlab.com/public/visits?site_id=6f50b5f4-8a8f-4e4a-9d69-1b2a3c4d5e6f&url=https%3A%2F%2Fexample.com%2F" alt="" width="1" height="1" />
```

## Daily counts

Choose **Daily counts only** when you create a site.
The traffic profile is fixed after creation.
Existing sites keep individual visits.
Use a separate aggregate site for native analytics.

The public browser client is `count.js?site_id=SITE_ID`.
It sends one request when the script starts.
For the two LoopAware-owned frontend hosts, `count.js` maps requests to the LoopAware API hostname.
For another script host, it sends requests to that script's origin.
It does not read `api_origin` or `data-api-origin` overrides.
For self-hosting, serve or proxy the count endpoint on that origin, or call the API directly.

The native client uses this HTTP contract:

```http
POST /public/sites/SITE_ID/visit-counts
Content-Type: application/json
Origin: CONFIGURED_ORIGIN

{}
```

Send the request to the LoopAware API hostname.
Omit cookies and referrers. Keep the body empty except for the JSON object delimiters.
Do not retry an uncertain response.

A `204` response confirms the increment.
An incorrect traffic profile returns `409`.
Invalid fields or query data return `400`. A disallowed origin returns `403`.

The database stores only the site ID, UTC date, and accepted request count.
Reports cannot determine unique visitors, individual visits, pages, devices, or locations for aggregate sites.
These metrics are unavailable in the API and dashboards.
A mixed portfolio reports totals by UTC calendar day and marks visitor metrics unavailable.

`analytics.aggregate_retention_days` specifies the retention window, from 1 to 90 calendar days.
The canonical configuration uses 90 days, including the current UTC date.
Cleanup runs before the server accepts requests and each day afterward.
A restored database passes the same startup check.

Site deletion also removes the daily totals.

Collector request logs and crash logs exclude identifying request fields.
Verify proxy logs separately before production activation.
See [the application contract](../.mprlab/CHILD-AUDIENCE-ANALYTICS.md) for the required deployment evidence.

## Capturing developer errors

For server-side clients, use the protected `/sentry/errors` endpoint with a per-site ingest token. The repository
collects first-party client entrypoints under `clients/`:

- [Go client](../clients/go/README.md): `clients/go/lasentry`.
- [Python client](../clients/python/README.md): `clients/python/la_sentry`.
- [Browser client](../clients/browser/README.md): `clients/browser` documents the harness served from `web/la-sentry.js`.

Browser pages can use the standalone harness without exposing the server-side token:

```html
<script defer src="https://loopaware.mprlab.com/la-sentry.js?site_id=6f50b5f4-8a8f-4e4a-9d69-1b2a3c4d5e6f&environment=production&release=2026.04.24"></script>
```

The browser harness installs `window.LASentry.captureError(error, attrs)` and automatically captures uncaught
`error` and `unhandledrejection` events. It sends sanitized URL/referrer/user-agent metadata, stack frames, tags, and
explicit `extra` values supplied by application code.
