# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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

[Unreleased]: https://github.com/tyemirov/loopaware/compare/v0.2.0...HEAD
[v0.2.0]: https://github.com/tyemirov/loopaware/releases/tag/v0.2.0
[v0.1.4]: https://github.com/tyemirov/loopaware/releases/tag/v0.1.4
[v0.1.3]: https://github.com/tyemirov/loopaware/releases/tag/v0.1.3
[v0.1.2]: https://github.com/tyemirov/loopaware/releases/tag/v0.1.2
[v0.1.1]: https://github.com/tyemirov/loopaware/releases/tag/v0.1.1
[v0.1.0]: https://github.com/tyemirov/loopaware/releases/tag/v0.1.0
