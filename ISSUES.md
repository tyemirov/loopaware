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

- [ ] [LA-200] Goal: use the latest version of mpr-ui for the footer and header. Find the mpr-ui documentation under @tools/mpr-ui, @tools/mpr-ui/docs/custom-elements/md
      Deliverable: document missing DSL/functionality of the mpr-ui to allo GAuth integration with the login in details, including coding suggestions. Look into the declarative syntax of the mpr-ui web-components.
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

- [x] [LA-204] Add a dashboard “Test subscribe widget” flow matching the feedback widget experience. Added subscribe-test dashboard button + preview page (inline/bubble controls, SSE status log) with Rod coverage for clicking through and verifying subscriber creation/notifications; wired public + SSE handlers plus template updates.

  1. Introduce a `Test subscribe widget` button alongside the copy button in the subscribe card header; wire new element IDs/config entries plus JS (`sharedPaths` prefix/suffix) so the button opens a per-site preview page just like the existing feedback widget test button.
  2. Implement `/app/sites/:id/subscribe-test` (template + handlers) that renders the dashboard chrome, hosts the subscribe widget in both inline/bubble modes, surfaces accent/CTA/name-field toggles, and posts to the real `/api/subscriptions` endpoint for the selected site.
  3. Show submission status + notifier results on the test page, and add rod-based integration tests that (a) click the dashboard button, (b) interact with the test page, and (c) assert subscribers + notifier calls are recorded.

- [x] [LA-205] Provide a dashboard “Test traffic widget” workflow so pixel.js can be exercised from the UI, mirroring the feedback widget test page.

  1. Add a `Test traffic widget` button to the traffic card header and expose `sharedPaths` entries for a new `/app/sites/:id/traffic-test` page.
  2. Build the traffic test page (dashboard chrome + instructions) that loads `pixel.js` with the selected site ID, lets operators trigger sample hits (URL input + trigger button), and displays live visit totals/top pages pulled from `/api/visits` or SSE.
  3. Extend dashboard/headless integration tests to verify the button opens the page, pixel hits get recorded for the chosen site, and visit counts/top pages update during the session (both normal and `-race` CI runs).

  Traffic test preview page is wired into the dashboard, and the integration harness now exposes `/api/visits` so sample beacons hit the real collector; Rod tests confirm sample visits increment stats.

## BugFixes (300-399)

- [x] [LA-307] widget test page now uses the current request origin for widget.js so the preview widget renders even when the configured public base URL points elsewhere; added coverage.
- [x] [LA-308] Intermittent `TestWidgetIntegrationSubmitsFeedback` failure (headless focus wait around internal/httpapi/widget_integration_test.go:166) observed while preparing LA-204/205; investigate the rod/headless flow so the integration suite stays reliable locally and in CI. Resolved by waiting for panel focus transitions (including the initial field focus) and centralizing the shift-tab key chord to keep Rod interactions stable; headless + race suites now pass consistently.
- [x] [LA-309] 
1. There is no form preview for the subscribe widget on the test page. Add form preview for the subscribe widget on the test page. 
2. It is always supposed to be inline (embedded). Remove Bubble Preview section. 
3. Remove Inline Preview and place the submission form in its place. 
4. The theme toggle switch is no longer operation on the subscribe widget  test page. See @test_subscribe_widget_page.png
   Resolved by embedding the real inline subscribe form directly on the page, removing the unused bubble preview cards, wiring the controls to update the inline form, and restoring the theme toggle via the mpr-ui footer bundle.
- [ ] [LA-310] 
1. The theme toggle switch is no longer operation on the subscribe widget  test page. See @test_traffic_widget_page.png. 
2. The user avatar is hidden and some hideous oval is around it.  See @test_traffic_widget_page.png. 
3. I was expecting much richer information such as IP, country, browser, time of the day etc
4. I clicked record sample visit twice and got 2 unique visitors but I was expecting 1 unqiue visitor

## Maintenance (400-499)

- [x] [LA-400] Prepare a short marketing blurb about the LoopAware service. Place it under docs/ . The goal is to place this description in a card on a main site that advertises all mprlab products
- [x] [LA-401] Refresh the LoopAware marketing blurb for the mprlab product catalogue card with concise, card-ready copy under docs/; updated with new two-sentence catalog blurb.
- [ ] [LA-402] Fefactor the Dockerfile multibuild with alpine-based images

## Planning

**Do not work on these, not ready**
