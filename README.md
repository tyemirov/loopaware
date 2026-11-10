# LoopAware

[![CI](https://github.com/tyemirov/loopaware/actions/workflows/ci.yml/badge.svg)](https://github.com/tyemirov/loopaware/actions/workflows/ci.yml)
[![License: PolyForm Noncommercial](https://img.shields.io/badge/License-PolyForm%20Noncommercial-blue)](LICENSE)
[![Latest Release](https://img.shields.io/github/v/release/tyemirov/loopaware)](https://github.com/tyemirov/loopaware/releases)
**Customer feedback, traffic reports, and site health in one dashboard.**
LoopAware combines feedback, email subscriptions, traffic analytics, uptime checks, and application error reports.
Teams can manage multiple sites, give teammates access, and collect feedback from websites and mobile apps.

Use LoopAware to learn what visitors report, compare traffic across sites, and receive email reports and outage alerts.
The browser dashboard and native operator app use the same Go API and SQLite database.

## What you can do

| Capability | What it provides |
| --- | --- |
| Customer feedback | A website widget with contact details, messages, sentiment, and source context. A React Native client adds feedback to mobile apps. |
| Email subscriptions | Embedded forms, confirmation and unsubscribe links, subscriber management, and CSV export. |
| Traffic reports | Page views, unique visitors, trends, sources, devices, inferred locations, CSV export, and scheduled reports across selected sites. |
| Daily counts | Daily request totals with no individual visit records. Select this analytics mode when you create a site. |
| Reliability | Backend uptime checks with down/recovered alerts. LA Sentry groups application errors from browser, Go, and Python clients. |

Authenticated users can create sites.
Global administrators can manage all sites.
Owners and creators manage their sites.
Assigned team members can read site data.

Each collection feature has its own client or endpoint.
Enable the features that your site needs.

## Get your first feedback

Use an existing LoopAware service, or complete the [local setup](docs/setup.md) first.
The public frontend address is [loopaware.mprlab.com](https://loopaware.mprlab.com/).

1. Open `/login` on your frontend and sign in with Google.
2. Open `/app` and create a site with its name, owner email, and allowed website origin.
3. Copy the site's feedback snippet from the dashboard.
4. Add the snippet to a page on an allowed origin.
5. Open the feedback bubble and submit a valid email or phone number with a message or sentiment.
6. Open the site's Feedback tab and confirm that the submission appears.

For the public service, the snippet has this form:

```html
<script defer src="https://loopaware.mprlab.com/widget.js?site_id=YOUR_SITE_ID"></script>
```

For your own deployment, use its script hostname and the API origin from your dashboard snippet.
See [collection clients](docs/integrations.md) for subscriptions, analytics, mobile feedback, and error reports.

## Choose your analytics mode

The traffic profile is fixed when you create a site.

| | Detailed traffic | Daily counts only |
| --- | --- | --- |
| Stored analytics | Individual visits with visitor identifiers, IP addresses, user agents, page URLs, referrers, and browser context. | Site ID, UTC date, and accepted-request count. |
| Reports | Page views, unique visitors, pages, attribution, engagement, devices, and inferred locations. | Daily request totals. Visitor and page metrics are unavailable. |
| Client | `pixel.js`, or an image pixel with fewer signals. | `count.js`, or the native HTTP endpoint. |
| Retention | The server configures no automatic deletion window for detailed visits. | Configurable from 1 to 90 UTC calendar days. The tracked configuration uses 90 days. |

The detailed browser client stores a persistent visitor identifier in `localStorage`.
Unique visitors measure distinct recorded identifiers, rather than verified individuals.
Location signals are estimates with source and confidence metadata.
Daily counts measure accepted requests, including repeat requests.

Feedback contact details and subscriber addresses have separate records from analytics.
Review the [collection contract](docs/integrations.md#daily-counts) and [privacy policy](https://loopaware.mprlab.com/privacy/) for data details.

## Run it yourself

LoopAware uses a Go API, SQLite, a static frontend, and a native Expo/React Native operator app.
SQLite is the currently supported database.

The local Docker Compose stack includes these services:

- LoopAware for collection, management APIs, background tasks, and SQLite storage.
- TAuth for Google sign-in and operator sessions.
- Pingwin for email delivery.
- gHTTP for local static files and reverse-proxy routes.

The browser frontend loads shared MPR UI components and other external dependencies from CDNs.
Production separates the GitHub Pages frontend, LoopAware API, and TAuth hostnames.
See [setup and configuration](docs/setup.md) for required inputs and the local startup procedure.
See [deployment](docs/deployment.md) for the Gateway lifecycle, native builds, and publication inputs.

## Documentation and development

| Guide | Contents |
| --- | --- |
| [Setup](docs/setup.md) | Configuration, authentication, local Docker, and frontend hostnames. |
| [Integrations](docs/integrations.md) | Widget, subscriptions, traffic, daily counts, mobile feedback, and LA Sentry. |
| [Dashboards](docs/dashboard.md) | Site access, reports, traffic metrics, and web/mobile controls. |
| [HTTP API](docs/api.md) | Routes, roles, collection limits, and metric definitions. |
| [Development](docs/development.md) | Tests, browser checks, dependency audits, and local test isolation. |
| [Deployment](docs/deployment.md) | Release, publication, activation, signing, and store preparation. |
| [React Native feedback client](clients/react-native/README.md) | Installation and application integration. |

After you install the development prerequisites, use the repository Make targets:

```bash
make test
make lint
make ci
```

These commands validate the source with automated tests.
Hosted deployment, package publication, and mobile store availability require their own operation results.
The manifest selects internal TestFlight and Google Play tracks for the operator app.
The operator app is separate from the React Native feedback client embedded in customer apps.

## License and contact

The repository uses the [PolyForm Noncommercial License 1.0.0](LICENSE).
See the [commercial licensing notice](COMMERCIAL_LICENSE.md) for commercial use and separately licensed materials.
Contact [legal@mprlab.com](mailto:legal@mprlab.com) for commercial terms.
