# LoopAware PRD

## Summary
LoopAware collects customer feedback through a website widget.
Operators use shared `mpr-ui`/TAuth sign-in and a dashboard with site access controls.
The design supports fast integration, strict origin controls, and direct operator workflows.

## Current reference

The [README](README.md) summarizes the current capabilities.
The [setup guide](docs/setup.md) defines the separate production frontend, API, and TAuth origins.
The [integration guide](docs/integrations.md) defines detailed traffic, aggregate counts, and mobile collection.

## Goals
- Capture in-page feedback with minimal integration work.
- Provide a role-aware dashboard for operators and admins.
- Support email subscriptions (double opt-in) and traffic analytics.
- Enforce strict per-site origin validation across widgets and APIs.
- Deliver real-time dashboard updates for feedback and favicon refreshes.

## Non-goals
- Owning identity-provider-specific authentication scaffolding.
- Building a full CRM or ticketing system.
- Replacing customer support platforms.

## Personas
- Operator: owns sites, reviews feedback, manages subscribers.
- Admin: manages all sites, can reassign owners.
- Public visitor: submits feedback, subscribes to updates, tracked via pixel.

## User journeys
1) Create site
- Operator signs in at /login and opens /app.
- Operator creates a site with name, owner email, and allowed origins.
- Dashboard returns widget, subscribe, and pixel snippets.

2) Collect feedback
- Operator embeds /widget.js on an allowed origin.
- Visitor submits feedback via the widget.
- Dashboard receives feedback in real time (SSE) and lists messages.

3) Capture subscribers
- Operator embeds /subscribe.js with a site_id.
- Visitor submits email, receives confirmation, and confirms.
- Operator can export or manage subscribers in the dashboard.

4) Track traffic
- Operator embeds /pixel.js on allowed origins.
- Visits and unique visitors appear in dashboard stats.

## Functional requirements
### Authentication and access
- Shared `mpr-ui`/TAuth sign-in.
- The browser redirects unauthenticated dashboard visitors to /login.
- TAuth verifier-backed session cookies authorize protected /api requests.
- Roles: admin and user (owner/creator scope).

### Site management
- Create, update, delete sites.
- Persist allowed origins as a list (space/comma separated).
- Support owner reassignment for site managers.

### Feedback widget
- /widget.js embeds a bubble UI and posts to /public/feedback.
- Backend validates origin against allowed origins.
- Feedback persists and is streamed via SSE.

### Subscription flow
- /subscribe.js renders a form and posts to /public/subscriptions.
- Double opt-in via /subscriptions/confirm token link.
- Unsubscribe via API or token link.
- Origin validation for subscription capture. Signed tokens authorize confirmation and unsubscribe links.

### Traffic pixel
- /pixel.js sends visits to /public/visits.
- Dashboard reads aggregated stats per site.

### Dashboard
- Site list, editor, and widget snippet copy.
- Feedback list with timestamps.
- Subscribers table with export.
- Traffic panel with visit counts and top pages.
- Real-time favicon and feedback updates via SSE.

### Public pages
- / root landing page with product overview, canonical SEO metadata, and links into the public resource hub.
- /login landing page with shared sign-in and root-canonical product overview.
- /resources crawlable public guides for feedback widgets, mobile feedback, subscribers, analytics, report emails, uptime, access control, LA Sentry, agencies, SaaS teams, and self-hosted deployments.
- /privacy static policy page.
- /sitemap.xml for search engine discovery.

## Non-functional requirements
- Security: HttpOnly session cookies, no tokens in localStorage, strict origin validation.
- Reliability: The design calls for unbuffered, stable SSE streams.
- Performance: widgets must load quickly without page delays.
- Compliance: privacy policy and sitemap must be reachable and accurate.

## Success metrics
- Feedback submission success rate.
- Sign-in success on first attempt.
- Email confirmation completion rate.
- Dashboard load time and error-free API calls.

## Dependencies
- `mpr-ui` for browser sign-in scaffolding.
- TAuth for session issuance and verifier-backed session validation.
- Pingwin gRPC service for notifications.
- SQLite storage.

## Risks
- Misconfigured allowed origins leading to rejected widget traffic.
- Cross-origin cookie restrictions in misconfigured deployments.
- External dependencies (`mpr-ui`, TAuth, Pingwin) impacting uptime.
