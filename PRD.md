# LoopAware PRD

## Summary
LoopAware collects customer feedback through a lightweight widget, authenticates operators with Google, and offers a role-aware dashboard for managing sites and messages. It is designed for fast embed, strict origin controls, and a clean operator experience.

## Goals
- Capture in-page feedback with minimal integration work.
- Provide a role-aware dashboard for operators and admins.
- Support email subscriptions (double opt-in) and traffic analytics.
- Enforce strict per-site origin validation across widgets and APIs.
- Deliver real-time dashboard updates for feedback and favicon refreshes.

## Non-goals
- Supporting non-Google identity providers.
- Building a full CRM or ticketing system.
- Replacing customer support platforms.
- Multi-service split of frontend and backend (separate planning item).

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
- Google Identity Services via TAuth.
- Session cookies validate /app and /api routes.
- Roles: admin and user (owner/creator scope).

### Site management
- Create, update, delete sites.
- Persist allowed origins as a list (space/comma separated).
- Support owner reassignment for admins.

### Feedback widget
- /widget.js embeds a bubble UI and posts to /api/feedback.
- Backend validates origin against allowed origins.
- Feedback persists and is streamed via SSE.

### Subscription flow
- /subscribe.js renders a form and posts to /api/subscriptions.
- Double opt-in via /subscriptions/confirm token link.
- Unsubscribe via API or token link.
- Origin validation for all subscription endpoints.

### Traffic pixel
- /pixel.js posts visits to /api/visits.
- Dashboard reads aggregated stats per site.

### Dashboard
- Site list, editor, and widget snippet copy.
- Feedback list with timestamps.
- Subscribers table with export.
- Traffic panel with visit counts and top pages.
- Real-time favicon and feedback updates via SSE.

### Public pages
- /login landing page with sign-in and product overview.
- /privacy static policy page.
- /sitemap.xml for search engine discovery.

## Non-functional requirements
- Security: HttpOnly session cookies, no tokens in localStorage, strict origin validation.
- Reliability: SSE streams should remain unbuffered and stable.
- Performance: widgets must load quickly and avoid blocking the page.
- Compliance: privacy policy and sitemap must be reachable and accurate.

## Success metrics
- Feedback submission success rate.
- Sign-in success on first attempt.
- Email confirmation completion rate.
- Dashboard load time and error-free API calls.

## Dependencies
- TAuth for authentication and session validation.
- Google Identity Services for sign-in UI.
- Pinguin gRPC service for notifications.
- SQLite default storage (pluggable drivers).

## Risks
- Misconfigured allowed origins leading to rejected widget traffic.
- Cross-origin cookie restrictions in misconfigured deployments.
- External dependencies (GIS, TAuth, Pinguin) impacting uptime.
