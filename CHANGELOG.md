# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## Unreleased

### Improvements ⚙️
- Move the GitHub Pages deploy resource into `deploy/app.yml` so gateway Ansible executes the app-owned frontend deployment contract.
- Derive deploy release tags from the app repository `HEAD` automatically instead of accepting operator-supplied deploy revisions.

## [v0.7.21] - 2026-06-13

### Features ✨
- Add per-site team member management with read-only dashboard access for assigned emails.
- Introduce Admin dashboard section for site managers to handle team membership and traffic report recipients.
- Support traffic report recipient modes: manager-only, whole team, and selected team members with validation and multi-recipient dispatch.
- Add API routes for managing site team members and traffic report schedules.

### Improvements ⚙️
- Strengthen email validation to reject display names in team member emails.
- Allow team members to access streams and handle partial traffic report deliveries.
- Prevent retry of partial multi-recipient report deliveries to avoid duplicate emails.
- Update README and issue log with new team member access, admin features, and traffic report scheduling.

### Bug Fixes 🐛
- Fix partial multi-recipient traffic report delivery retry behavior.
- Restore missing-subscriber delete response to `unknown_subscription`.

### Testing 🧪
- Add coverage for traffic report schedule recipient modes.
- Add team member management and read-only access tests for sites.
- Validate email format restrictions for team member invitations.

### Docs 📚
- Update README with per-site team-member access, admin UI, API changes, and traffic report scheduling features.
- Document role-aware dashboard scopes including per-site team-member access.

## [v0.7.20] - 2026-06-12

### Features ✨
- Add Expo-compatible mobile feedback widget support with native feedback button for React Native and Expo apps.
- Introduce mobile app feedback endpoints and API routes for registered native apps.
- Add React Native feedback client with TypeScript types and configuration for type checking.
- Enhance mobile app feedback display and search in the dashboard.
- Reject mobile feedback requests with browser origin headers to prevent invalid submissions.

### Improvements ⚙️
- Run typecheck for React Native client in lint-js target.
- Post-review hardening to reject browser-origin mobile feedback submissions before client validation.

### Bug Fixes 🐛
- _No changes._

### Testing 🧪
- Add rejection test for mobile feedback from browser origins.
- Add tests for mobile app feedback endpoints.

### Docs 📚
- Add mobile feedback API and usage guide to README.md and clients/react-native README.
- Update mobile feedback issue resolution with post-review hardening details.
- Document mobile app registration and usage for feedback submission.

## [v0.7.19] - 2026-06-09

### Features ✨
- Improve visitor location accuracy using edge geo signals and inferred locations.
- Replace visitor timezone reporting with inferred locations combining timezone, locale, network, and edge geo signals.
- Add inferred visitor locations API endpoint with confidence metadata.

### Improvements ⚙️
- Preserve country-only edge geo even for unmapped countries by using a shared unmapped-country anchor.
- Optimize location aggregation by grouping raw location signal tuples in SQL before inference.
- Rename all timezone-related routes, variables, tests, and exports to location to reflect new location inference.
- Enhance CSV export and traffic report emails to include detailed inferred location data.
- Update Makefile to run location map checks instead of timezone map checks.
- Update documentation to describe new visits API endpoints, pixel data details, and traffic data architecture.

### Bug Fixes 🐛
- Fix location distribution to preserve country-only edge geo and optimize aggregation.
- Rename dashboard traffic section from timezone to location for clarity and accuracy.

### Testing 🧪
- Rename and update tests to reflect location distribution instead of timezone distribution.
- Add regression coverage for unmapped country-only edge geo and enhanced location inference.
- Validate with extensive unit, integration, and Playwright/API integration tests.

### Docs 📚
- Update README with new visits API endpoints and pixel data details for location signals.
- Update ARCHITECTURE.md with detailed traffic data flow including edge geo and inferred locations.
- Document changes in dashboard, CSV export, and traffic report emails reflecting location inference.

## [v0.7.18] - 2026-06-09

### Features ✨
- Add traffic interval selector (1 day, 30 days, All) and CSV export button to dashboard.
- Add interval filtering to traffic-related API endpoints.
- Add export endpoint for site visit traffic data.

### Improvements ⚙️
- Update API docs and architecture with interval query parameter for visits endpoints.
- Enhance Timezones map label alignment and visit count display inside bubbles.
- Mark selected-site Traffic intervals and CSV export as completed in ISSUES.md.

### Bug Fixes 🐛
- Sanitize CSV export fields to prevent formula injection.

### Testing 🧪
- Add traffic CSV export and interval selector tests with UI updates.

### Docs 📚
- Update API documentation with new traffic interval filtering and export features.
- Add detailed issue B028 for Timezones label alignment and coverage.

## [v0.7.17] - 2026-06-08

### Features ✨
- _No changes._

### Improvements ⚙️
- Redesigned the Timezones map with a real world land outline and geographic coordinate placement for bubbles.
- Improved device icons in the Devices row graph to clearly differentiate tablet and mobile.
- Added a CI-enforced check for the generated timezone world map path to prevent drift.
- Refined traffic map styles and updated device type icons for better clarity.

### Bug Fixes 🐛
- Ignored route teardown errors in the runTrackedRoute helper to prevent test failures during Playwright context/browser closure.
- Fixed handling of external-asset route fetches cancelled by test teardown to avoid false failures.

### Testing 🧪
- Added timezone world map generation and enhanced traffic dashboard tests with geographic coordinate assertions.
- Expanded dashboard traffic spec to cover device icon distinctions and timezone map layers.
- Added lint-js target to run timezone map consistency checks.

### Docs 📚
- Added detailed issue records documenting timezone map and device icon improvements.

## [v0.7.16] - 2026-06-08

### Features ✨
- _No changes._

### Improvements ⚙️
- Redesigned traffic report tables as grid-based row graphs and maps for Pages, Devices, and Timezones sections.
- Replaced duplicate timezone views with a visit-density map showing bubble sizes proportional to visits.
- Replaced duplicate device views with an icon row graph showing device icons and visit counts.
- Replaced duplicate top-pages views with a ranked path row graph including rank, icon, and visit count.
- Renamed selected-site traffic breakdown section headings to Pages, Devices, and Timezones for clarity.
- Added black-box dashboard test coverage for new traffic report visualizations and labels.

### Bug Fixes 🐛
- Added `Cross-Origin-Opener-Policy: same-origin-allow-popups` header to local, computercat, and test proxy configurations to fix Google Identity popup auth compatibility with edge opener policy.
- Fixed stale four-hour login test by replacing it with black-box browser coverage that simulates idle login and asserts no errors or failed auth responses.
- Enhanced Google Identity stub to seed auto-credential behavior before GIS script request, ensuring zero pre-click initialization and stricter nonce-bound credential validation.

### Testing 🧪
- Improved traffic dashboard tests to cover new map and row graph renderings.
- Enhanced Google Identity stub for more accurate auth flow simulation and testing.
- Added regression coverage for security headers including COOP policy.
- Validated changes with 393 Playwright/API integration specs and `make ci`.

### Docs 📚
- Updated ISSUE.md with recent bugfixes and improvements summaries related to auth flows, traffic report redesign, and security headers.

## [v0.7.15] - 2026-06-05

### Features ✨
- _No changes._

### Improvements ⚙️
- Added Playwright test coverage for login page sign-in after four hours idle to catch stale nonce issues without modifying production code.

### Bug Fixes 🐛
- Fixed login failure after four hours idle on the landing page by verifying nonce refresh and sign-in flow in tests.

### Testing 🧪
- Introduced new integration test simulating four-hour idle on `/login` page to ensure Google sign-in completes successfully.
- Enhanced auth state tests with time manipulation and Google Identity stub validation.

### Docs 📚
- Updated issue tracking documentation with detailed description and resolution of the four-hour login idle bug.

## [v0.7.14] - 2026-06-05

### Features ✨
- Add customizable accent color for feedback bubble widget.
- Add widget accent color support in site configuration API.
- Add widget accent color validation and UI integration tests.

### Improvements ⚙️
- Improve authentication retry and Google identity nonce handling.
- Refactor tests to use Google Identity stub testing API from `mpr-ui`.
- Update widget controls description in README to include accent color.

### Bug Fixes 🐛
- Stabilize browser harness readiness in CI for authentication tests.
- Restore feedback bubble color customization with full support.

### Testing 🧪
- Add Google Identity stub testing API and refactor auth tests.
- Add widget accent color validation and UI integration tests.

### Docs 📚
- Update ISSUES.md with Google Identity stub test improvements.
- Update README.md with widget accent color control description.

## [v0.7.13] - 2026-06-04

### Features ✨
- Add date labels and tick marks to the X-axis of traffic trend charts for better time context.
- Display first, middle, and last date ticks with UTC-formatted labels on selected-site and all-sites traffic trend charts.

### Improvements ⚙️
- Enhance dashboard traffic chart tests to dynamically verify X-axis date labels for 7-day and 30-day trend windows.
- Update SVG renderer to include accessible aria-label describing time axis range and visit scale.

### Bug Fixes 🐛
- _No changes._

### Testing 🧪
- Improve traffic trend chart axis label checks with dynamic date calculations.

### Docs 📚
- Update issue tracker with detailed summary and resolution of adding X-axis time labels on traffic trend charts.

## [v0.7.12] - 2026-06-02

### Features ✨
- _No changes._

### Improvements ⚙️
- Added Playwright test coverage to recover login after long-idle Google nonce expiry, ensuring fresh nonce preparation and successful sign-in flow.

### Bug Fixes 🐛
- Fixed login recovery issue caused by expired Google nonce callback preventing authentication exchange.
- Updated shared `mpr-ui` auth controller to reject expired GIS callback nonces and emit stale nonce events.
- Published `mpr-ui` v3.10.1 with nonce timestamping and improved auth flow robustness.

### Testing 🧪
- Added end-to-end Playwright test simulating long-idle login recovery with nonce expiration and refresh.
- Enhanced black-box browser coverage for header authentication state and login flows.

### Docs 📚
- Documented B019 bug fix and recovery steps in `.mprlab/ISSUES.md`.

## [v0.7.11] - 2026-06-02

### Features ✨
- _No changes._

### Improvements ⚙️
- Added visible count scales and grid lines to traffic trend charts for better readability.
- Updated Playwright tests to use system Chrome in CI, reducing browser installation overhead.
- Added SSE heartbeat frames every 30 seconds to keep dashboard streams alive through gateway read timeouts.

### Bug Fixes 🐛
- Fixed Playwright CI browser setup to prevent premature GitHub Actions job cancellation by installing only Chromium and disabling video capture.
- Resolved dashboard SSE stream closures by emitting periodic heartbeat comments to prevent HTTP/2 protocol errors.

### Testing 🧪
- Added tests verifying SSE heartbeat frames are sent on favicon, feedback, and subscription event streams.
- Updated multiple test files and API handlers for consistency and improved coverage.
- Updated dashboard traffic spec to assert visible count scales on charts.

### Docs 📚
- Updated ISSUES documentation with detailed summaries and resolutions for recent bug fixes and improvements.

## [v0.7.10] - 2026-05-29

### Features ✨
- Added LoopAware-side auth recovery to retry dashboard API requests once after a 401 response by calling TAuth `/auth/refresh` before redirecting to login.
- Extended `cmd/configaudit` to enforce matching LoopAware and TAuth tenant environment variables for session cookie name, JWT signing key, and tenant ID.
- Added Playwright test coverage for stale-first `/api/me` unauthorized response and automatic auth recovery flow.

### Improvements ⚙️
- Updated example environment files to align TAuth JWT signing key placeholders.
- Refined frontend auth request handling to support configurable TAuth logout and refresh paths.
- Improved audit command to detect and warn about LoopAware/TAuth session cookie name mismatches.

### Bug Fixes 🐛
- Fixed production dashboard authentication issue caused by LoopAware API runtime drifting from TAuth tenant cookie-name/signing-key contract.
- Patched `mprlab-gateway` to validate and reject future LoopAware/TAuth session-cookie drift in deployment config.

### Testing 🧪
- Added unit tests for config audit checks on LoopAware and TAuth tenant environment variable consistency.
- Added Playwright tests for login flow retry on transient dashboard API unauthorized responses.

### Docs 📚
- Updated `.mprlab/ISSUES.md` with detailed summary and progress on production dashboard auth recovery and config audit enhancements.
- Improved inline comments and documentation in `cmd/configaudit` for tenant invariant checks.

## [v0.7.9] - 2026-05-25

### Features ✨
- Remove aggregate Top pages sections from all-sites traffic reports to reduce noise and improve report clarity.

### Improvements ⚙️
- Updated portfolio traffic report API, dashboard, and email templates to omit aggregate Top pages data for all-sites reports while retaining individual-site Top pages.
- Added regression tests covering the absence of aggregate Top pages in all-sites reports.
- Cleaned up dashboard and frontend code by removing all-sites Top pages UI elements.

### Bug Fixes 🐛
- _No changes._

### Testing 🧪
- Added tests verifying that all-sites traffic report emails exclude aggregate Top pages.
- Updated API and dashboard tests to confirm removal of Top pages from all-sites reports.

### Docs 📚
- Updated issue tracking documentation to reflect removal of aggregate Top pages from all-sites reports.

## [v0.7.8] - 2026-05-25

### Features ✨
- Moved the account card from the dashboard side column into the Account Settings modal to keep the dashboard side column focused on site selection.
- Added black-box test coverage for the new account card placement in the Account Settings modal.

### Improvements ⚙️
- Updated dashboard readiness helpers to support hidden modal account fields.
- Enhanced test fixtures and specs for dashboard and logout features to reflect UI changes.
- Updated README to describe the Account Settings modal replacing the previous account card.

### Bug Fixes 🐛
- _No changes._

### Testing 🧪
- Added tests verifying the account card appears in the Account Settings modal.
- Added tests ensuring the dashboard side column contains only the Sites card.
- Updated existing dashboard tests to wait for account hydration in the new modal location.

### Docs 📚
- Updated README to reflect the Account Settings modal with avatar, email, role badge, reports, and inactivity controls.

## [v0.7.7] - 2026-05-25

### Features ✨
- Added portfolio traffic report API with full CRUD and scheduling support.
- Introduced portfolio traffic report email template.
- Added separate all-sites traffic dashboard view with portfolio reporting mode.
- Implemented global report-library visual language with saved report definitions and included-site selection.
- Added persisted custom global traffic report definitions scoped to the current user.

### Improvements ⚙️
- Moved all-sites traffic reporting entry into Account Settings for clearer scope separation.
- Split all-sites traffic into a distinct dashboard view, isolating selected-site workspace.
- Updated dashboard graphics for selected-site traffic trends, attribution, engagement, device, and timezone metrics.
- Enhanced scheduling and test-report endpoints to support portfolio reports without overloading per-site schedules.
- Updated environment and configuration files to align with current Pinguin runtime schema and TAuth metadata handling.
- Improved release script to use 'gix sync' instead of 'gix cd'.

### Bug Fixes 🐛
- Fixed global traffic report load failures to preserve error visibility and prevent downstream data clearing.
- Corrected schedule edits to preserve saved selected-site and all-sites timezones.
- Rendered built-in all-sites report name as explicit read-only default to avoid confusion.
- Scoped scheduled report device and timezone totals to the report window for consistent weekly email reporting.
- Restored integration Pinguin startup after config schema drift by updating tenant identity metadata.

### Testing 🧪
- Added black-box API and dashboard coverage for new portfolio reporting surfaces, scheduling, and test report delivery.
- Added regression tests ensuring old visits do not inflate weekly email device/timezone breakdowns.
- Expanded integration tests to cover global report-library creation, selection, and included-site membership changes.

### Docs 📚
- Updated documentation and example environment files to reflect new portfolio traffic report features and configuration changes.
- Revised dashboard UI copy to use "sites" instead of "properties" for portfolio context.

## [v0.7.6] - 2026-05-15

### Features ✨
- _No changes._

### Improvements ⚙️
- _No changes._

### Bug Fixes 🐛
- _No changes._

### Testing 🧪
- _No changes._

### Docs 📚
- Updated ISSUES.md with the latest production dashboard access restoration details after TAuth login issue.

## [v0.7.5] - 2026-05-15

### Features ✨
- _No changes._

### Improvements ⚙️
- Deploy release tags are now automatically derived from the app repository `HEAD`, removing manual revision selection during deploy.

### Bug Fixes 🐛
- _No changes._

### Testing 🧪
- _No changes._

### Docs 📚
- Updated documentation to reflect automatic release tag detection and the removal of manual tag input in deploy scripts.

## [v0.7.4] - 2026-05-15

### Features ✨
- _No changes._

### Improvements ⚙️
- Updated deployment flow to manage GitHub Pages deployment through `deploy/app.yml`, enabling the gateway Ansible to handle backend and Pages deployment as one contract.
- Simplified `make deploy` command by removing explicit GitHub Pages workflow parameters and URLs, enhancing deployment consistency.
- Enhanced deployment script to enforce proper flag usage and streamlined backend and Pages deployment verification.

### Bug Fixes 🐛
- _No changes._

### Testing 🧪
- _No changes._

### Docs 📚
- Updated README to clarify deployment process and the role of `deploy/app.yml` in managing GitHub Pages workflow through the gateway.
- Cleaned up deployment script usage documentation to align with new deployment contract design.

## [v0.7.3] - 2026-05-14

### Features ✨
- _No changes._

### Improvements ⚙️
- Migrate LoopAware authentication to use shared `mpr-ui` sign-in scaffolding with TAuth session issuance and verifier-backed validation.
- Remove direct Google Identity Services scripts and app-owned TAuth bootstrap from LoopAware pages, relying fully on `mpr-ui` for browser auth lifecycle.
- Enforce logout overlay persistence during auth reconciliation and hide main content under logout overlay on dashboard.
- Update CI and browser test coverage to verify single auth controller ownership, correct auth events handling, and session timeout logout behavior.
- Remove local `tools/mpr-ui` code and enforce CDN-only shared UI assets, including mpr-ui scripts and helpers.
- Strengthen session timeout logout flow to redirect only on successful `/auth/logout` responses, maintaining dashboard overlay on failures.
- Fix timeout and stability issues in dashboard allowed-origin tests and autosave validation error visibility.
- Update documentation, marketing, and architecture to reflect the new shared auth boundary and remove Google/TAuth internal handling.

### Bug Fixes 🐛
- Fix header-owned login auth handling and credential exchange bugs by adopting latest shared `mpr-ui` assets and auth scaffolding.
- Address nonce lifecycle bug in `mpr-ui` related to unauthenticated `/me` and `/auth/refresh` probes.
- Restore dashboard allowed-origin test stability and fix autosave race causing stale validation states to hide errors.

### Testing 🧪
- Add black-box browser tests ensuring no direct GIS scripts load while shared auth flow still works, validating single Google auth controller presence.
- Add coverage for logout failure recovery with persistent overlays and session-timeout idle manager restarts.
- Update Playwright tests to cover header auth state changes, logout overlay hardening, and allowed origin flows.

### Docs 📚
- Revise README, PRD, architecture, marketing blurb, and privacy/terms to document delegation of auth scaffolding to shared `mpr-ui`.
- Update authentication flow docs to remove direct TAuth/Google Identity Services bootstrap, clarifying reliance on `mpr-ui` for sign-in lifecycle.
- Add config audit rule blocking presence of local `tools/mpr-ui` symlink or code to enforce CDN-only usage.
- Clarify handling of session cookie issuance via TAuth and role-aware sign-in in key architecture constraints and functional requirements.

## [v0.7.2] - 2026-05-08

### Features ✨
- Replace landing page login CTAs with real Google sign-in controls to improve sign-in flow activation.
- Added black-box browser coverage for landing page login and public page brand navigation.

### Improvements ⚙️
- Enhanced deployment script with flexible gateway directory resolution and backend image verification.
- Added dry-run option to publish script for CI checks without pushing images.
- Release and deployment scripts now have improved environment variable handling and helper resolution.
- Simplified Google sign-in stub to render a clickable button instead of delegating clicks.
- Updated test login flow to click the newly implemented Google sign-in button.
- Refined Makefile defaults and deployment script argument parsing.

### Bug Fixes 🐛
- Removed legacy programmatic click bridge that interfered with Google sign-in.
- Fixed runtime auth bootstrap scope to prevent competing Google sign-in controllers on login page.
- Resolved flaky or broken header-auth-state tests by aligning with new sign-in button.

### Testing 🧪
- Updated Playwright tests to support new `mpr-login-button` Google sign-in interactions.
- Added Playwright coverage for signed-out and signed-in public page brand navigation.
- Verified `make ci` passes with all new changes.

### Docs 📚
- Updated ISSUE tracker with detailed summary of the production landing login fix and test coverage.

## [v0.7.1] - 2026-05-03

### Features ✨
- Introduced scripted release, publish, and deploy workflows with gating for controlled production rollout.
- Added `make` targets for release, publish, and deploy to manage the release lifecycle.
- Implemented a deployment manifest and backend deployment through mprlab-gateway with health verification.

### Improvements ⚙️
- Changed GitHub Pages and Docker image workflows to manual dispatch instead of tag-driven triggers.
- Enhanced release process to ensure release tags are validated and pushed before publishing.
- Updated README with detailed release, publish, and deploy process workflow and instructions.
- Added comprehensive shell scripts for publishing Docker images, releasing versions, and deploying backend with GitHub Pages publishing sequencing.

### Bug Fixes 🐛
- _No changes._

### Testing 🧪
- Added new API helper tests and improved integration test feedback flows.
- Validated release and deployment processes with CI gates before publishing artifacts.

### Docs 📚
- Updated README to clarify the release-to-production sequence and manual workflow dispatching.
- Documented new Makefile targets and deployment manifest usage.

## [v0.7.0] - 2026-05-03

### Features ✨
- Added audience-scoped subscription status API.
- Introduced 'LA Sentry' as a first-class developer monitoring surface with protected server-side ingest endpoints.
- Added browser JavaScript LA Sentry capture with origin-bound ingest to safely capture frontend errors without exposing secrets.

### Improvements ⚙️
- Consolidated LA Sentry client discovery under a new `clients/` directory for Go, Python, and browser clients.
- Removed legacy LA Sentry package wrapper from `pkg/lasentry` for cleaner SDK exposure.
- Fixed subscription confirmation brand navigation to route users appropriately based on authentication state.
- Added black-box integration and dashboard coverage for login, subscription flows, and LA Sentry client.
- Updated landing page to advertise LA Sentry developer monitoring alongside existing features.
- Enhanced SQL query auditing, security, and performance with no regressions detected.

### Bug Fixes 🐛
- Fixed B003 subscription brand navigation to prevent users from being trapped on the confirmation page.
- Updated IP rate limiting to use per-client windows instead of wall-clock buckets to prevent rate-limit boundary issues.

### Testing 🧪
- Added Playwright black-box tests covering login completion, brand navigation on public pages, and LA Sentry ingest API.
- Verified integration tests for LA Sentry Go and Python clients.
- Increased test coverage of public API and authentication flows with new scenarios.

### Docs 📚
- Updated README files for clients, browser, Go, and Python SDKs with usage instructions and examples.
- Documented LA Sentry architecture, client configuration, and ingest contract.
- Enhanced public landing page metadata and feature descriptions to include LA Sentry.
- Updated ISSUES.md with improved status markers, format, and maintenance guidance.

## [v0.6.0] - 2026-05-01

### Features ✨
- Added LA Sentry developer monitoring with protected server-to-server ingest and a dedicated dashboard tab.
- Introduced first-class LA Sentry clients: Go middleware and Python package for error capture.
- Implemented browser JavaScript LA Sentry capture with origin-bound ingest and public CORS.

### Improvements ⚙️
- Improved favicon resolver accuracy and added live favicon integration tests.
- Enhanced rate limiter to store one bounded counter per client IP and prune expired windows.
- Optimized dashboard site listing query batching for feedback, subscriber, visit, and unique-visitor counts.
- Added composite indexes for performance in visit aggregations and Sentry issue listing.
- Normalized LA Sentry client names and added browser and Python Sentry client support.
- Hardened Sentry ingest to retry races, atomically increment counts, and strip query/fragment secrets from request metadata.

### Bug Fixes 🐛
- Fixed favicon discovery to block localhost, private, special-use, link-local, multicast, and unspecified targets before dispatch.
- Addressed issues from LA Sentry review findings.
- Fixed rate limiter to reject new clients once counters reach capacity.

### Testing 🧪
- Added black-box API and dashboard coverage for LA Sentry backend and browser captures.
- Added live favicon integration coverage validating favicons from Google, Wikipedia, GitHub, Apple, Microsoft, and Reddit.
- Added Playwright tests for LA Sentry browser and API integrations.

### Docs 📚
- Added documentation for new LA Sentry developer client types and dashboard features.
- Updated README with usage and integration details for LA Sentry clients and browser harness.
- Documented new API endpoints for LA Sentry error ingest and query.

## [v0.5.8] - 2026-04-23

### Features ✨
- _No changes._

### Improvements ⚙️
- Consolidated `.mprl` folder contents into `.mprlab` and standardized folder structure.
- Updated GitHub Actions workflows to use Node 24 runtime and upgraded action versions for CI, Docker image builds, and Pages deployment.
- Restored MPRL Docker instructions and refined frontend input UX plans in backlog for future updates.

### Bug Fixes 🐛
- _No changes._

### Testing 🧪
- CI workflows now run on Node 24 ensuring fresh compatibility with latest environment.
- Enabled CI triggers on workflow file changes to validate pipeline updates.

### Docs 📚
- Moved documentation including ISSUES and AGENTS files into the new `.mprlab` directory for clarity and standardization.
- Removed deprecated `.mprl/AGENTS.md` document and archived older issue backlog into `.mprlab/ISSUES_ARCHIVE.md`.
- Updated Agents frontend documentation to enforce CDN usage for third-party browser dependencies pinned to specific versions.

## [v0.5.7] - 2026-04-23

### Features ✨
- Added scheduled traffic report emails with daily, weekly, or monthly delivery options.
- Introduced API endpoints to manage traffic report schedules and send test reports.
- Enabled autosave controls for traffic report scheduling in the dashboard.

### Improvements ⚙️
- Scopped traffic report top pages data to the current window for relevance.
- Moved login authentication transitions into the header for better UX.
- Upgraded Go dependency to 1.26 and updated related module dependencies.

### Bug Fixes 🐛
- Fixed stale dashboard inactivity prompts hiding correctly when trusted user activity resumes outside the timeout banner.
- Reset traffic report retry state correctly on save to ensure reliable scheduling.
- Fixed sign-in retry issues and delayed auth profile mutations to improve login stability.

### Testing 🧪
- Added Playwright tests covering dashboard inactivity prompt behavior and traffic report schedule controls.
- Introduced new tests for traffic report scheduling API and email sending functionality.

### Docs 📚
- Updated README with instructions for enabling and configuring traffic report emails.
- Added environment variable documentation for traffic report email feature (`TRAFFIC_REPORT_EMAILS_ENABLED`).

## v0.5.6 (2026-04-21)

## [v0.5.6] - 2026-04-21

### Features ✨
- Start login flow immediately on dashboard call-to-action before redirect on the login page.

### Improvements ⚙️
- Added script to handle dashboard login clicks by triggering the login flow via UI events.
- Enhanced login page links to use the new dashboard login flow trigger attribute.

### Bug Fixes 🐛
- _No changes._

### Testing 🧪
- Added Playwright tests to verify the dashboard CTA triggers login flow correctly before redirecting.
- Included new helper function and test case for dashboard login flow on the login page.

### Docs 📚
- _No changes._

## v0.5.6 (2026-04-21)



## [Unreleased]

### Bug Fixes 🐛
- Hide stale dashboard inactivity prompts when trusted operator activity resumes outside the timeout banner.

### Testing 🧪
- Added Playwright coverage for active dashboard interaction dismissing the inactivity prompt without breaking explicit timeout confirm/dismiss actions.

## [v0.5.5] - 2026-04-21

### Features ✨
- _No changes._

### Improvements ⚙️
- Removed redundant page-level Google Identity bootstrap wiring from the public auth pages and test surfaces so `mpr-ui@latest` remains the single auth bootstrap owner.
- Aligned local development and integration cookie names with the explicit `loopaware_development_*` prefix to avoid collisions with production sessions.

### Bug Fixes 🐛
- Fixed the public login shell so it no longer double-initializes Google Identity, preventing the broken sign-in handoff that surfaced as duplicate GIS initialization warnings and `POST /auth/google` failures.
- Updated public-page auth and SEO assertions to match the current landing-page copy and metadata.

### Testing 🧪
- `make ci` passes, including the full Playwright integration suite covering public auth state, CDN asset loading, and SEO metadata.

### Docs 📚
- Updated README examples to use the development-scoped LoopAware cookie names.

## [v0.5.4] - 2026-04-20

### Features ✨
- Added Google Analytics tracking to all main HTML pages for improved traffic insights.

### Improvements ⚙️
- _No changes._

### Bug Fixes 🐛
- _No changes._

### Testing 🧪
- _No changes._

### Docs 📚
- _No changes._

## [v0.5.3] - 2026-04-19

### Features ✨
- Added a copy snippet button next to widget snippets for improved UX.
- Introduced a public auth waiting screen during sign-in transitions on public pages.

### Improvements ⚙️
- Updated header authentication helpers and strengthened logout hardening flows.
- Enhanced Google Identity and TAuth browser stubs to support nonce-backed credential exchange.
- Refined test coverage for UI states related to header authentication and logout recovery.

### Bug Fixes 🐛
- Fixed an issue where public pages could not complete Google sign-in after logout without a full page reload, improving sign-in recovery without reloads.
- Resolved a bug where the waiting screen on login was not properly shown during authentication handoff.

### Testing 🧪
- Added extensive Playwright tests covering snippet copy buttons UI and functionality.
- Expanded tests for header auth state transitions, login page behavior during sign-in, and logout-to-login recovery paths.
- Improved stub helpers for external assets and TAuth to better simulate authentication flows.

### Docs 📚
- Updated issues documentation with detailed resolution steps and verification commands related to authentication fixes and UI improvements.

## [v0.5.2] - 2026-04-18

### Features ✨
- Add explicit logout state persistence to avoid erroneous auth transition modal reopening after logout.
- Introduce test to verify explicit logout does not reopen auth transition modal on login.

### Improvements ⚙️
- Update integration test script to track and use Docker Compose project names for proper cleanup.
- Modify Makefile to streamline targets and update Docker Compose down command for local environment.
- Enhance header-auth module with explicit logout state tracking via session/local storage and UI sync.
- Improve auth state snapshot normalization considering explicit logout flag.
- Remove logout message from UI for a cleaner user experience.
- Refactor integration test shutdown logic to prevent leftover Docker Compose projects.

### Bug Fixes 🐛
- Fix test-related bugs ensuring stable logout test cases.
- Prevent auth transition modal from appearing after explicit user logout.

### Testing 🧪
- Add robust logout-hardening.spec.js test to check logout overlay behavior and auth modal persistence.
- Refine integration tests and add improved logout transition tracking.

### Docs 📚
- Clarify integration test runner behavior in README, removing outdated instructions about manual test stack teardown.
- Miscellaneous documentation tweaks for consistency with new test and auth flow updates.

## [0.5.1] - 2026-04-18

### Features ✨
- Use MPR UI's built-in auth transition for login-to-dashboard handoff.
- Add dashboard auth transition overlay with visible loading state until authenticated UI finishes loading.
- Dispatch `loopaware:dashboard-ready` event once authenticated dashboard shell is ready.

### Improvements ⚙️
- Retry dispatching dashboard ready event until auth transition overlay hides, ensuring reliable dashboard handoff.
- Update dashboard test helpers to wait for auth transition overlay to disappear before interactions.
- Revert MPR UI frontend asset URLs back to `@latest` after upstream release confirmation.

### Bug Fixes 🐛
- Harden transition release to work when MPR UI boots without readiness helper, preventing dashboard overlay from hanging CI tests.

### Testing 🧪
- Add focused Playwright tests covering auth transition overlay visibility and disappearance during dashboard boot.
- Add regression tests to verify normal authenticated dashboard boot hides the auth transition overlay.
- Enhance existing header auth state tests to cover new transition behavior.

### Docs 📚
- Add detailed ISSUE entries documenting the `auth-transition` feature and dashboard ready event contract.

## [0.5.0] - 2026-04-12

### Features ✨
- Enforced browser security headers at the edge for all compose stacks via proxy middleware.
- Added new `up` and `down` aliases and integration cleanup targets to Makefile for improved developer experience.

### Improvements ⚙️
- Centralized Playwright test harness and normalized specs for better testing consistency.
- Refined runtime configuration and hardened security settings for the API and deployment stacks.
- Migrated frontend runtime configuration from tracked config file to static `web/config.yml` for direct serving.
- Enhanced SEO metadata for public-facing pages.
- Removed Bootstrap JS stubs from test helpers for cleaner test setup.
- Updated Compose proxy services to deliver comprehensive security headers consistently.
- Improved documentation regarding static frontend hosting and integration test workflows.

### Bug Fixes 🐛
- Excluded service files from the build and deploy processes as intended.
- Fixed integration tests teardown behavior with new `test-down` target to avoid stale Docker Compose projects.

### Testing 🧪
- Added middleware tests verifying security headers are set including HSTS on HTTPS requests.
- Added layout tests for dashboard content spacing under sticky header.
- Updated and cleaned integration test scripts and Playwright specs.

### Docs 📚
- Updated setup instructions for static frontend hosting and proxy security header management.
- Documented new Makefile aliases for starting and stopping integration test stacks.
- Clarified configuration file locations and usage for runtime frontend settings.

## [0.4.1] - 2026-04-06

### Features ✨
- _No changes._

### Improvements ⚙️
- Removed extra padding and margin from dashboard main container for tighter layout.

### Bug Fixes 🐛
- _No changes._

### Testing 🧪
- Added layout test to verify dashboard content spacing under the sticky header.

### Docs 📚
- _No changes._

## [v0.4.0] - 2026-04-03

### Features ✨
- Added new API endpoints for device breakdown and timezone distribution statistics.
- Introduced device breakdown and timezone distribution data models and handlers.

### Improvements ⚙️
- Optimized site statistics to avoid unnecessary slice allocation for empty viewport rows.
- Added limit parameters with validation for device breakdown and timezone distribution endpoints.
- Enhanced test coverage for new API endpoints and validation rules.

### Bug Fixes 🐛
- Fixed unnecessary slice allocation in site statistics calculation for empty viewport rows.

### Testing 🧪
- Added comprehensive tests for device breakdown and timezone distribution API endpoints.
- Included authentication and error handling tests for new statistics endpoints.

### Docs 📚
- _No changes._

## [v0.3.0] - 2026-04-03

### Features ✨
- Capture richer visit metadata including screen resolution, viewport, timezone, and page title.
- Refresh public authentication UI and update MPRUI to latest version.
- Add a gHTTP proxy stack serving the local web folder and forwarding backend API requests.

### Improvements ⚙️
- Update docker-compose setup to include a new proxy service for local development.
- Enhance footer layout with improved utility links order: Privacy link before horizontal links and correct link grouping.
- Upgrade CDN asset references to use latest MPRUI versions in tests.

### Bug Fixes 🐛
- Fix timeout banner anchoring to stay properly above the hydrated footer in dashboard UI.

### Testing 🧪
- Refine Playwright tests for allowed origins update checks with polling for persistence.
- Improve session timeout banner tests to verify anchoring relative to footer using layout calculation.
- Update auth header state tests to assert latest CDN URLs for MPR UI assets.
- Add utility functions and refine UI state tests for footer links and authentication UI.

### Docs 📚
- Remove the planning document from the repository.
- Add documentation for the gHTTP proxy stack and its configuration.
- Update README with proxy usage instructions and local compose stack changes.
- Add multiple new LoopAware logo image assets in various sizes and SVG format.

## [v0.2.1] - 2026-04-01

### Features ✨
- Added new Pricing and Terms of Service pages with updated footer links.

### Improvements ⚙️
- Updated README with usage instructions, license details, and badges for CI, license, Go version, and latest release.

### Bug Fixes 🐛
- _No changes._

### Testing 🧪
- _No changes._

### Docs 📚
- Added CONTRIBUTING guide covering license, development workflow, submission process, coding style, and issue reporting.

## [v0.2.0] - 2026-03-21

### Features ✨
- Add sentiment support for public and admin feedback APIs.
- Configure widget feedback inputs and visual capture flow for message input and sentiment buttons.

### Improvements ⚙️
- Enhance widget configurability: allow toggling of message input and sentiment buttons on the feedback widget.
- Update API and dashboard to display sentiment data with feedback messages.
- Restore and refine `.gitignore` for better project file management.

### Bug Fixes 🐛
- _No changes._

### Testing 🧪
- Add tests for widget feedback visibility settings (message input and sentiment toggling).
- Verify sentiment field presence and correctness in feedback API responses.
- Include tests ensuring widget defaults and validation logic for feedback visibility.

### Docs 📚
- Clarify feedback submission requirements to include valid contact plus message and/or sentiment in the README.

## [v0.1.4] - 2026-03-21

### Features ✨
- Add `siteWidgetSiteId` support in frontend runtime config to bootstrap the first-party feedback widget on `/login` and `/app`.
- Internalize API origin mapping in widget and subscribe scripts to avoid exposing `api_origin` in customer snippets.

### Improvements ⚙️
- Restore site widget bootstrap on LoopAware-owned login and dashboard pages.
- Refactor test helpers to generate fresh session cookies dynamically for more reliable authenticated Playwright tests.
- Update dashboard snippets to omit `api_origin` ensuring secure embeds for customers.
- Enhance Playwright coverage for authenticated and unauthenticated header auth states with a new header-auth state machine.

### Bug Fixes 🐛
- Fix stale server-side session cookie reuse causing intermittent `401 unauthorized` errors in authenticated integration tests.
- Resolve customer embed issues requiring customers to copy internal API hosts by internalizing the API origin resolution.

### Testing 🧪
- Expanded Playwright tests to cover unauthenticated `/app` redirect, authenticated `/login` redirect, and dashboard avatar-dropdown UI states.
- Added focused test coverage asserting that customer snippets do not expose `api_origin`.
- Refreshed integration test sessions with userId-based JWT generation for stability after worker restarts.

### Docs 📚
- Documented `siteWidgetSiteId` configuration in `configs/config.frontend.yml` and deployment notes.
- Updated config README to describe frontend service settings including `siteWidgetSiteId` for the widget bootstrap.

## [v0.1.3] - 2026-03-21

### Features ✨
- Enforce CDN-only delivery for all third-party frontend dependencies, removing vendored `mpr-ui` assets.
- Pin third-party browser dependencies (`mpr-ui`, `TAuth`) to specific CDN URLs for reproducible deployments.
- Add new auth bootstrap architectural constraints to prevent tenant bootstrap errors on public pages.

### Improvements ⚙️
- Prevent auth redirect flicker during session recovery for smoother user experience.
- Defer UI bootstrap until TAuth script is ready to avoid race conditions.
- Update `mpr-ui` to version 3.8.2 and remove bundled login widget bootstrap.
- Enhance test helpers with delay options to simulate various authentication bootstrap delays.
- Add browser regression tests to verify pinned CDN asset usage and auth state management during delayed bootstrapping.

### Bug Fixes 🐛
- Fix login page to not bootstrap the landing widget, avoiding unnecessary network requests and console errors.

### Testing 🧪
- Add comprehensive Playwright tests covering:
  - Auth state preservation during TAuth script/load delays.
  - Correct loading of pinned CDN assets on login pages.
  - Proper handling of silent session bootstrap and current user delays.
  - Enforcement of CDN-only frontend dependency usage with no local vendor assets.

### Docs 📚
- Update architecture documentation to specify strict frontend dependency delivery rules (CDN-only).
- Clarify frontend deployment and auth bootstrap constraints.
- Enhance README and issue guidelines with requirements to use pinned CDN URLs for third-party assets.
- Document changes in frontend auth bootstrap strategy and runtime environment configuration.


## [v0.1.2] - 2026-03-20

### Features ✨
- Added a unified header-auth state machine for consistent authenticated header UI across pages.
- Introduced focused Playwright tests to cover authenticated header state, session recovery, and logout overlay.

### Improvements ⚙️
- Synchronized header authentication state with user menu for seamless session recovery on dashboard and static pages.
- Enhanced test stubs with silent bootstrap and delayed auth initialization options.
- Updated gitignore to include vendor UI assets and test specifications for better development workflow.

### Bug Fixes 🐛
- Fixed delayed authentication bootstrap causing static logout overlay display.
- Corrected header-auth state inconsistencies that left the Google sign-in button visible post-session recovery.
- Ensured logout overlay remains visible correctly during manual logout and forced session expiration.

### Testing 🧪
- Added extensive Playwright UI tests for header auth state, dashboard user menu, session timeout handling, and logout hardening.
- Improved end-to-end test fixture setup with avatar defaults and session cookie handling.
- Verified logout overlay behavior and header state synchronization under various authentication scenarios.

### Docs 📚
- Updated issues documentation to reflect fixes for header authentication state and session recovery synchronization issues.

## [v0.1.1] - 2026-03-19

### Added
- Config-audit now rejects unsupported config files and legacy `.env.*` duplicates outside the designated directories.
- Added expanded repository agent and process documentation across Docker, frontend, Git, and issue policy guides.
- Added logout-hardening browser coverage for public pages, token-error flows, and logout failure recovery.

### Changed
- Moved `up.sh` and `down.sh` into `scripts/` and improved their non-TTY handling and error messages.
- Unified tracked config file paths, moved integration compose/env fixtures under `tests/`, and updated GitHub Pages to publish frontend runtime configuration during deploy.
- Improved config-audit to cover multiple compose files while excluding gitignored env files from audit requirements.
- Hardened frontend logout synchronization and public auth overlay handling, and improved compatibility of URL parameter parsing across frontend components.

### Fixed
- UI content now restores after failed logout requests instead of leaving the logout overlay stuck on screen.
- Subscribe query values are preserved correctly during logout hardening and origin sync flows.
- Logout redirect assertions and rapid token-error test flows now behave consistently in integration coverage.

## [v0.1.0] - 2026-02-18

### Added
- Role-aware dashboard with Google authentication via TAuth.
- Feedback widget and public feedback collection endpoint with strict per-site origin validation.
- Subscription widget with double opt-in confirmation flow, unsubscribe flow, and CSV export.
- Traffic pixel with visit and unique visitor metrics, top pages, trend, attribution, and engagement analytics.
- Real-time server-sent events for feedback and favicon refresh updates.
- Multi-origin site configuration support for widget, subscription, and traffic collection.

### Changed
- Release automation now publishes GitHub Pages and Docker images only on pushed version tags matching `vMAJOR.MINOR.PATCH`.

### Fixed
- Top-pages aggregation now merges trailing-slash and non-trailing-slash paths and normalizes all-slash paths to `/`.
- Visit trend aggregation now normalizes SQL day keys to avoid dropping counts for timestamp-like day values.
- WhatsApp in-app browser traffic is no longer misclassified as bot traffic.
- Widget API origin resolution now falls back to HTTPS-aware behavior in proxy deployments that omit `X-Forwarded-Proto`.

[Unreleased]: https://github.com/tyemirov/loopaware/compare/0.5.0...HEAD
[0.5.0]: https://github.com/tyemirov/loopaware/releases/tag/0.5.0
[0.4.1]: https://github.com/tyemirov/loopaware/releases/tag/0.4.1
[v0.2.1]: https://github.com/tyemirov/loopaware/releases/tag/v0.2.1
[v0.2.0]: https://github.com/tyemirov/loopaware/releases/tag/v0.2.0
[v0.1.4]: https://github.com/tyemirov/loopaware/releases/tag/v0.1.4
[v0.1.3]: https://github.com/tyemirov/loopaware/releases/tag/v0.1.3
[v0.1.2]: https://github.com/tyemirov/loopaware/releases/tag/v0.1.2
[v0.1.1]: https://github.com/tyemirov/loopaware/releases/tag/v0.1.1
[v0.1.0]: https://github.com/tyemirov/loopaware/releases/tag/v0.1.0
