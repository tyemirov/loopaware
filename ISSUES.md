# ISSUES

**Append-only section-based log**

Entries record newly discovered requests or changes, with their outcomes. No instructive content lives here. Read @NOTES.md for the process to follow when fixing issues.

Read @AGENTS.md, @AGENTS.DOCKER.md, @AGENTS.FRONTEND.md, @AGENTS.GIT.md, @POLICY.md, @NOTES.md, @README.md and @ISSUES.md. Start working on open issues. Work autonomously and stack up PRs.

Each issue is formatted as `- [ ] [LA-<number>]`. When resolved it becomes `- [x] [LA-<number>]`

## Features (100-199)

- [x] [LA-100] Add new functionality to allow LoopAware gather emails of the customers who want to be getting news from the web-sites. A customer shall be able to embed a simple customizeable form on their page for users to enter their email and subscribe to the news. Plan for end-2end functionality and eliver a plan verified against current codebase. Design captured in docs/LA-100-email-subscriptions.md; implementation pending.
- [x] [LA-101] Add new functionality to allow LoopAware gather statistics of visits on a given website, analogous to Facebook or Google pixel. Plan for end-2end functionality and eliver a plan verified against current codebase. Design captured in docs/LA-101-visit-analytics.md; implementation pending.
- [x] [LA-102] Deliver subscriber domain and schema: add `Subscriber` model/table with smart constructors, uniqueness on (site_id,email), migrations wired into AutoMigrate, and Go tests for validation/invariants. Implemented with `Subscriber` model, validation helpers, AutoMigrate wiring, and tests for constructor invariants and per-site uniqueness.
- [x] [LA-103] Ship public subscription endpoints: `POST /api/subscriptions` plus confirm/unsubscribe routes with origin + rate-limit checks, stable error codes, and integration tests covering happy/edge cases. Implemented routes with edge validation, status transitions, duplicate handling, and coverage in public tests.
- [x] [LA-104] Build embeddable `subscribe.js` and demo page: customizable copy/colors, inline + bubble modes, origin-safe submission to APIs, and browser test asserting end-to-end subscription persistence. Added subscribe.js asset with inline/bubble modes, public demo route, and headless integration test to verify persisted subscriptions.
- [x] [LA-105] Add dashboard subscriber management: owner/admin APIs to list/search/export/update status, UI table/cards with counts and SSE updates, and tests for auth + export contents. Added subscriber counts to site responses, list/export/status APIs with tests, and dashboard UI to view/export/toggle subscribers.
- [x] [LA-106] Wire subscription notifications: optional Pinguin hook on new confirmed subscribers, feature flag/env, and tests using a fake notifier to assert delivery codes. Subscription notifications now reuse Pinguin with a feature flag, plus tests for success/failure and disablement paths.
- [x] [LA-107] Implement visit tracking storage, origin-validated collection endpoint/pixel.js + pixel.gif response, and retention/rollups per LA-101. Delivered SiteVisit model, pixel endpoint, pixel.js asset, and retention/rollup job.
- [x] [LA-108] Add dashboard traffic stats and top-pages reporting: APIs returning totals and top URLs, UI card/table, and tests for auth/aggregation.
- [x] [LA-109] Build visit rollup + retention jobs: scheduler aggregating daily page_views/unique_visitors, pruning raw visits beyond retention, with tests.

## Improvements (200-299)

- [ ] [LA-200] Goal: use the latest version of mpr-ui for the footer and header. Find the mpr-ui documentation under tools/mpr-ui
      Deliverable: document missing DSL/functionality of the mpr-ui to allo GAuth integration with the login in details, including coding suggestions.
      Non-deliverable: code changes
- [x] [LA-201] Separate widgets into three widgets — dashboard now provides distinct feedback, subscribe, and traffic snippets with dedicated copy controls.

1. Feedback widget
2. Subscribe widget
3. Traffic (pixel) widget

- [x] [LA-202] Have separate panes for the retrieval of Subscribers Widget and Traffic Widget underneath Site widget — feedback, subscribers, and traffic snippets now live in individual cards stacked under site details.

1. Site Widget
2. Subscribers Widget
3. Traffic Widget

- [x] [LA-203] Have separate panes for Subscribers and Traffic, not nested inside the feedback messages — dashboard now renders standalone feedback, subscriber, and traffic cards.

1. Feedback messages pane stays the same
2. Subscribers pane after
3. Traffic page after

- [ ] [LA-204] Add subscriber flow integration tests mirroring feedback coverage: stub notifier, submit subscription via widget, assert persisted state and notifier delivery codes.
- [ ] [LA-205] Add widget end-to-end notification tests: exercise embed submission through the widget, verify delivery persistence and notifier calls similar to feedback tests; include race/CI coverage.

## BugFixes (300-399)

- [x] [LA-307] widget test page now uses the current request origin for widget.js so the preview widget renders even when the configured public base URL points elsewhere; added coverage.

## Maintenance (400-499)

- [x] [LA-400] Prepare a short marketing blurb about the LoopAware service. Place it under docs/ . The goal is to place this description in a card on a main site that advertises all mprlab products
- [x] [LA-401] Refresh the LoopAware marketing blurb for the mprlab product catalogue card with concise, card-ready copy under docs/; updated with new two-sentence catalog blurb.
- [ ] [LA-402] Fefactor the Dockerfile multibuild with alpine-based images

## Planning

**Do not work on these, not ready**
