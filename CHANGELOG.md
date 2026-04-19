# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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
