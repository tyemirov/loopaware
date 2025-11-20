# ISSUES
**Append-only section-based log**

Entries record newly discovered requests or changes, with their outcomes. No instructive content lives here. Read @NOTES.md for the process to follow when fixing issues.

Read @AGENTS.md, @AGENTS.DOCKER.md, @AGENTS.FRONTEND.md, @AGENTS.GIT.md, @POLICY.md, @NOTES.md, @README.md and @ISSUES.md. Start working on open issues. Work autonomously and stack up PRs.

Each issue is formatted as `- [ ] [LA-<number>]`. When resolved it becomes -` [x] [LA-<number>]`

## Features (100-199)

- [ ] [LA-100] Add new functionality to allow LoopAware gather emails of the customers who want to be getting news from the web-sites. A customer shall be able to embed a simple customizeable form on their page for users to enter their email and subscribe to the news. Plan for end-2end functionality and eliver a plan verified against current codebase. Design captured in docs/LA-100-email-subscriptions.md; implementation pending.
- [ ] [LA-101] Add new functionality to allow LoopAware gather statistics of visits on a given website, analogous to Facebook or Google pixel. Plan for end-2end functionality and eliver a plan verified against current codebase. Design captured in docs/LA-101-visit-analytics.md; implementation pending.
- [x] [LA-102] Deliver subscriber domain and schema: add `Subscriber` model/table with smart constructors, uniqueness on (site_id,email), migrations wired into AutoMigrate, and Go tests for validation/invariants. Implemented with `Subscriber` model, validation helpers, AutoMigrate wiring, and tests for constructor invariants and per-site uniqueness.
- [x] [LA-103] Ship public subscription endpoints: `POST /api/subscriptions` plus confirm/unsubscribe routes with origin + rate-limit checks, stable error codes, and integration tests covering happy/edge cases. Implemented routes with edge validation, status transitions, duplicate handling, and coverage in public tests.
- [x] [LA-104] Build embeddable `subscribe.js` and demo page: customizable copy/colors, inline + bubble modes, origin-safe submission to APIs, and browser test asserting end-to-end subscription persistence. Added subscribe.js asset with inline/bubble modes, public demo route, and headless integration test to verify persisted subscriptions.
- [ ] [LA-105] Add dashboard subscriber management: owner/admin APIs to list/search/export/update status, UI table/cards with counts and SSE updates, and tests for auth + export contents.
- [ ] [LA-106] Wire subscription notifications: optional Pinguin hook on new confirmed subscribers, feature flag/env, and tests using a fake notifier to assert delivery codes.
- [ ] [LA-107] Implement visit tracking storage and collection: `SiteVisit` model, origin-validated pixel endpoint (GIF/POST), visitor_id cookie persistence, and Go tests for validation/persistence.
- [ ] [LA-108] Add dashboard traffic stats and top-pages reporting: APIs returning timeseries and top URLs, UI cards/tables, auth checks, and integration tests on aggregates.
- [ ] [LA-109] Build visit rollup + retention jobs: scheduler aggregating daily page_views/unique_visitors, pruning raw visits beyond retention, with tests for rollup math and trimming.

## Improvements (200-299)

## BugFixes (300-399)

- [x] [LA-307] widget test page now uses the current request origin for widget.js so the preview widget renders even when the configured public base URL points elsewhere; added coverage.

## Maintenance (400-499)

- [x] [LA-400] Prepare a short marketing blurb about the LoopAware service. Place it under docs/ . The goal is to place this description in a card on a main site that advertises all mprlab products

## Planning 
**Do not work on these, not ready**
