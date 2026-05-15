# ISSUES

Entries record newly discovered requests or changes.

Read @AGENTS.md (Workflow section), @POLICY.md, and relevant stack guides before implementing changes.

Format: `- [ ] [B042] (P1) {I007} Title`

- `[ ]` open, `[!]` blocked, `[x]` closed.
- Blocked issues (`[!]`) must include a `Blocked:` line in the body.

## BugFixes

- [x] [B001] (P0) Verify successful login lands on a loaded dashboard.
  ### Summary
  Add black-box browser coverage for the full login completion path: an unauthenticated user starts from `/login`, completes Google/TAuth sign-in, receives a usable session, reaches `/app`, and sees the authenticated dashboard loaded.
  ### Deliverables
  - Add Playwright integration coverage that drives the login page sign-in flow rather than pre-seeding a session before navigation.
  - Assert the final dashboard URL and visible authenticated dashboard state.
  - Fix any auth handoff regression exposed by the new coverage.
  ### Resolution
  Added black-box Playwright coverage for the login CTA completing TAuth exchange, receiving a session cookie, reaching `/app`, waiting for the loaded dashboard, and verifying authenticated header/user state. The TAuth test stub now writes the post-exchange session cookie so the browser test matches the real login handoff. `make ci` passed.
- [x] [B002] (P0) Make IP rate limits independent of wall-clock bucket boundaries.
  ### Summary
  `make ci` exposed an intermittent LA Sentry browser rate-limit failure where requests started near the end of a 30-second wall-clock bucket could split across buckets and avoid the intended per-window limit.
  ### Deliverables
  - Use per-client rate windows that start at the first request in the window.
  - Apply the same boundary-safe behavior to public and browser Sentry rate limits.
  - Verify the black-box integration suite no longer lets the boundary case through.
  ### Resolution
  Replaced wall-clock bucket counters with per-client windows for public API and LA Sentry browser rate limits, updated helper tests, and verified `make test-integration-api` plus `make ci` pass.

- [x] [B003] (P1) Fix subscription confirmation brand navigation.
  ### Summary
  The LoopAware brand link on subscription token pages points to `#top`, so confirming a subscription leaves the user trapped on the confirmation page through the natural header flow.

  ### Deliverables
  - Make subscription token page brand links navigate to LoopAware instead of scrolling the current page.
  - Route unauthenticated users to `/login` and authenticated users to `/app`.
  - Add black-box browser coverage for the public-page brand navigation behavior.

  ### Resolution
  Replaced subscription token page brand `#top` links with auth-aware LoopAware home links, reused shared header auth state to update public-page brand destinations to `/login` or `/app`, added Playwright coverage for signed-out and signed-in public-page brand navigation, and verified `make ci` passes.

- [x] [B004] (P0) Make production landing login use a real Google sign-in control.
  ### Summary
  The production `/login` dashboard CTAs intercept clicks and programmatically re-click the header Google sign-in target. Real Google sign-in is rendered inside a cross-origin iframe, so this delegated click path can lose the user's direct activation and leave the user on the landing page instead of opening the sign-in flow.

  ### Deliverables
  - Replace landing dashboard CTA triggers with first-class `mpr-login-button` controls.
  - Remove the delegated dashboard-login click bridge that programmatically clicks the header Google sign-in target.
  - Add black-box browser coverage that signs in from the landing-page control and reaches the loaded dashboard.
  ### Resolution
  Replaced the landing dashboard CTA anchors with real `mpr-login-button` controls, removed the programmatic dashboard-login click bridge from `web/header-auth.js`, scoped runtime auth bootstrap so the login-page header no longer creates a competing Google controller, updated the Google test stub to expose a clickable rendered sign-in button, added black-box landing-login coverage through the loaded dashboard, and verified `make ci` passes.

- [x] [B005] (P0) Use config-first mpr-ui/TAuth authentication.
  ### Summary
  LoopAware still bootstraps authentication with app-owned `tauth.js` loading, direct TAuth helper globals, manual `tauth-*` attributes, and multiple login controls on `/login`. This interferes with mpr-ui's shared auth lifecycle and can trigger duplicate `/me`/`/auth/refresh` probes plus repeated Google Identity initialization.
  ### Deliverables
  - Serve `/config-ui.yaml` and let `mpr-ui-config.js` apply auth attributes before loading the mpr-ui bundle.
  - Remove direct `tauth.js` loading and app-owned Google/TAuth helper orchestration.
  - Keep LoopAware code limited to public auth events, redirects, and product-specific overlays.
  - Add black-box browser coverage that the login page has a single Google auth controller.
  ### Resolution
  Moved LoopAware auth configuration to `/config-ui.yaml`, switched served pages to the `mpr-ui-config.js` and bundle-marker flow using `mpr-ui@latest`, removed direct `tauth.js`/TAuth helper globals, and kept app code to public mpr-ui auth events plus product redirects/overlays. Fixed the shared `mpr-ui` nonce lifecycle bug that mismatched GIS nonce tokens after unauthenticated `/me`/`/auth/refresh` bootstrap, published the fix upstream, updated black-box coverage for the single auth controller, TAuth credential exchange, and logout failure recovery, and verified `make ci` passes.

- [x] [B006] (P0) Keep the login page Google sign-in in the header actions.
  ### Summary
  The `/login` page needs the visible Google sign-in control in the right side of the header without making LoopAware own Google/TAuth bootstrap or creating a second mpr-ui auth controller.
  ### Deliverables
  - Render the login page Google sign-in as a shared `mpr-ui` control inside the header action area.
  - Keep `/login` on the config-first `mpr-ui@latest` path with no app-owned TAuth bootstrap.
  - Preserve single-controller coverage for `/me`, `/auth/refresh`, and Google Identity initialization.
  ### Resolution
  Kept `/login` on the canonical `<mpr-header data-config-url="/config-ui.yaml">` path so the built-in header Google button remains in the right-side header actions. Fixed and published `mpr-ui` so nested header user menus mirror header auth events/state instead of starting their own profile bootstrap, preserving a single mpr-ui auth owner for `/me`, `/auth/refresh`, and Google Identity initialization. Updated Playwright coverage for header-right placement, TAuth config ownership, credential exchange, and the single auth controller.

- [x] [B007] (P0) Remove LoopAware-owned Google auth scaffolding.
  ### Summary
  LoopAware must not load or inspect Google authentication plumbing directly. Browser authentication scaffolding belongs to `mpr-ui`; session verification belongs to TAuth's verifier. LoopAware should only consume public `mpr-ui` auth lifecycle events and perform product-specific redirects, overlays, and authorization.
  ### Deliverables
  - Remove explicit Google Identity Services script loading from LoopAware-authored HTML pages.
  - Remove LoopAware JavaScript selectors that target the shared header's Google sign-in internals.
  - Keep login redirects driven by documented `mpr-ui` auth lifecycle events.
  - Add black-box coverage proving served LoopAware HTML does not include direct GIS script tags while the shared sign-in flow still works.
  ### Resolution
  Removed direct Google Identity Services script tags from LoopAware-authored auth pages and dashboard preview pages. Replaced the LoopAware header auth click probe against shared Google-control internals with documented `mpr-ui:auth:status-change` and `mpr-ui:header:signin-click` lifecycle handling. Updated README, architecture, PRD, marketing copy, privacy, and terms language so LoopAware describes the auth boundary as shared `mpr-ui`/TAuth sign-in plus TAuth verifier-backed session validation. Added black-box Playwright coverage that fetches each auth page over HTTP and verifies the served HTML does not load GIS directly while the shared sign-in flow still passes. Follow-up coverage now forces delayed authenticated mpr-ui reconciliation after explicit logout so the protected dashboard keeps the logout overlay active until redirect. `make ci` passed.
  ### Changed Files
  `ARCHITECTURE.md`, `PRD.md`, `README.md`, `docs/loopaware-marketing-blurb.md`, `tests/specs/header-auth-state.spec.js`, `tests/specs/logout-hardening.spec.js`, `web/header-auth.js`, shared auth HTML pages under `web/`.

- [x] [B008] (P0) Gate timeout logout redirect on successful response.
  ### Summary
  The session-timeout logout flow redirects to `/login` after any resolved `/auth/logout` fetch response. HTTP 4xx/5xx responses still resolve, so a failed server logout can move users off `/app` while their server session remains valid.
  ### Deliverables
  - Redirect to the landing page only after `/auth/logout` returns a successful response.
  - Keep failed timeout logout attempts on the authenticated dashboard with visible, recoverable UI state.
  - Add black-box browser coverage for the failed logout response path.
  ### Resolution
  Updated the dashboard timeout logout request to throw on non-OK `/auth/logout` responses, recover the dashboard overlay state on failure, and restart the idle manager when the session-timeout flow remains on `/app`. Added black-box browser coverage for a failed session-timeout logout response and verified the full `make ci` gate passes.

- [x] [B009] (P0) Restore dashboard allowed-origin browser coverage stability.
  ### Summary
  Full `make ci` times out in the dashboard allowed-origin browser tests before the suite can complete. The failure appears before the B008 changed path and was reproduced in both the pre-change baseline and post-change CI attempt.
  ### Deliverables
  - Reproduce the `specs/dashboard-allowed-origins.spec.js:88` timeout in isolation.
  - Identify whether the failure belongs to dashboard auth settling, the allowed-origin UI flow, or the Playwright harness setup.
  - Restore the full `make ci` gate.
  ### Resolution
  Moved seeded browser-auth synchronization to the public `MPRUI.testing` helper exposed by the shared header package and released that helper through CDN-hosted mpr-ui. Updated the harness to match the current mpr-ui no-anonymous-probe contract. While restoring the full gate, fixed a follow-on dashboard autosave race so disabling both widget feedback controls shows the validation error immediately and stale autosave success responses no longer hide the invalid state. `make ci` passed.

- [x] [B010] (P0) Enforce CDN-only shared UI assets.
  ### Summary
  LoopAware must never carry a local `tools/mpr-ui` checkout or test-time vendored shared UI bundle; all third-party and shared UI assets must come from CDN-hosted URLs.
  ### Deliverables
  - Remove the local `tools/mpr-ui` symlink from the workspace.
  - Remove local/shared-asset fallback and test-time CDN patching from the Playwright asset harness.
  - Add a CI guard that fails when `tools/mpr-ui` exists.
  - Verify the failing dashboard/auth browser cases pass against CDN-hosted `mpr-ui@latest`.
  ### Resolution
  Removed the `tools/mpr-ui` symlink, removed local mpr-ui reads and test-time asset patching from `tests/helpers/externalAssets.js`, and added a config-audit rule with coverage that rejects `tools/mpr-ui`. Published `mpr-ui` tag `v3.9.7` so `mpr-ui@latest` exposes `MPRUI.testing`, then verified the 27 dashboard/auth cases from the failed run pass against CDN-only assets.

- [!] [B011] (P0) Restore production dashboard access after TAuth login.
  ### Summary
  The production dashboard remains unauthenticated after the shared TAuth login handoff because the LoopAware API runtime can drift from the TAuth tenant cookie-name contract.
  ### Deliverables
  - Verify the live production login/dashboard path and identify the failing layer.
  - Align the deployed LoopAware API session cookie name with the TAuth `loopaware` tenant.
  - Add gateway-side validation so future deploy config cannot mismatch the two values.
  - Verify the local LoopAware suite and focused gateway config checks pass.
  ### Progress
  Identified the live production failure layer and patched `mprlab-gateway`: the LoopAware API env must read TAuth's `app_session_loopaware` cookie and configaudit now rejects future LoopAware/TAuth session-cookie drift. LoopAware `make ci` and gateway `make ci` passed locally.
  Blocked: `make deploy-loopaware-backend` in `mprlab-gateway` stopped at the interactive `Gateway sudo password:` prompt before applying the corrected runtime config to production.

## Improvements

- [ ] [I004] (P1) Consider a design of a current accordion design of different surfaces.
  We may want to have a better split out.
- [x] [I005] (P1) Keep production deploy revision selection automatic.
  ### Summary
  The deploy flow should not ask operators to name or select a revision for Pages/backend deployment. The release workflow owns tagging; deploy consumes the release tag at repository `HEAD`.
  ### Resolution
  Removed the deploy wrapper's manual `--tag` option and `DEPLOY_TAG` override. `make deploy` now derives the v* release tag from `HEAD` when Pages or image verification needs it, and otherwise tells the operator to run the release flow before deploy. Gateway Ansible owns Pages dispatch from the app manifest. Validation passed with `bash -n scripts/deploy.sh`, the deploy no-op dry run, `git diff --check`, and `timeout -k 1200s -s SIGKILL 1200s make ci`.
- [x] [I001] (P1) Advertise LA Sentry on the public landing page.
  ### Summary
  The public landing page currently presents feedback, subscriber capture, and traffic analytics, but omits LA Sentry even though it is now a first-class developer monitoring surface.
  ### Deliverables
  - Update landing-page metadata and visible copy to include LA Sentry developer error monitoring.
  - Add LA Sentry as a first-class feature card alongside the other embeddable surfaces.
  - Update public-page tests that assert landing-page copy.
  ### Resolution
  Updated `/login` landing metadata, hero copy, feature grid, and setup copy to advertise LA Sentry as a first-class developer monitoring surface; updated public-page and auth-state tests; verified `make ci` passes.
- [x] [I002] (P1) Consolidate LA Sentry client discovery under `clients/`.
  ### Summary
  Make the first-party LA Sentry clients discoverable from a dedicated `clients/` entrypoint instead of requiring readers to know that Go, Python, and browser surfaces live in different runtime-oriented folders.
  ### Deliverables
  - Add a client index under `clients/` for Go, Python, and browser usage.
  - Move the Go client implementation under `clients/` so client-facing SDKs live outside the server package namespace.
  - Document why the browser harness remains served from `web/la-sentry.js`.
  - Update repo docs and integration fixtures to prefer the dedicated client locations.
  ### Resolution
  Added `clients/README.md` as the LA Sentry client index, moved the Go client implementation to `clients/go/lasentry`, removed the legacy `pkg/lasentry` package so SDKs are exposed only from `clients/`, added browser and Go client docs under `clients/`, updated README references and the Go integration fixture, and verified `make ci` passes.

- [ ] [I001] (P1) Replace placeholder-only inputs with labeled fields in the static frontend.
  Added `clients/README.md` as the LA Sentry client index, moved the Go client implementation to `clients/go/lasentry` with a `pkg/lasentry` compatibility package, added browser and Go client docs under `clients/`, updated README references and the Go integration fixture, and verified `make ci` passes.
- [ ] [I003] (P1) Replace placeholder-only inputs with labeled fields in the static frontend.
  ### Summary
  Remove placeholder-only UX in the dashboard, widget, and subscribe flows and use explicit labels with specific copy.
  ### Deliverables
  - Update `web/app` pages plus `web/widget.js` and `web/subscribe.js` to render labeled inputs.
  - Remove placeholder text where it is the only accessible label or instruction.
  - Keep draft and empty-state copy specific.
  - Add or update black-box browser coverage for the changed static frontend flows.
  ### Legacy Ref
  - Migrated from `issues.md` issue `LA-426`.


## Maintenance

- [ ] [M001] (P0) Audit and harden SQL queries for security, correctness, and performance.
  ### Summary
  Review all database access paths in the Go backend and confirm SQL behavior is current, secure, and performant. This includes ORM-generated SQL (GORM) and any raw SQL paths used by repository/data-access layers.
  
  ### Analysis
  The backend stack (Go + PostgreSQL-oriented tooling) can accumulate query risk in three areas: security (unsafe interpolation), correctness drift (queries no longer matching current schema/usage), and latency (missing/ineffective indexes, N+1 patterns, over-fetching).
  
  This issue should produce a code-aware SQL audit that:
  - Enumerates all query entry points (repository methods, adapters, migrations, and any raw SQL strings).
  - Verifies parameterization and input handling at DB boundaries (no string-concatenated SQL from user input).
  - Validates query shape against current schema and access patterns.
  - Identifies expensive queries using execution plans and proposes targeted optimizations.
  
  ### Deliverables
  - Query inventory document in issue notes with:
    - Query location (file/function).
    - Query type (ORM-generated or raw SQL).
    - Risk classification (`secure`, `needs-fix`, `needs-optimization`).
  - Implemented fixes for all `needs-fix` security/correctness findings:
    - Parameterized statements/placeholders only.
    - Removal of unsafe dynamic SQL assembly.
    - Updated query logic to match current schema constraints.
  - Performance improvements for high-cost queries:
    - Add/adjust indexes where justified by access pattern.
    - Eliminate obvious N+1 and unnecessary column/row fetches.
    - Use pagination/bounds for unbounded list paths where applicable.
  - Integration test updates (black-box style) covering key data-access flows impacted by query changes.
  - Acceptance criteria:
    - No SQL injection-prone query construction remains.
    - All modified query paths pass existing integration tests.
    - At least one measured plan comparison (`EXPLAIN`/`EXPLAIN ANALYZE`) is captured for each optimized high-cost query, showing improved cost or runtime.
    - Changes are limited to required DB/query paths and documented in the issue with before/after rationale.
  ### 2026-05-01 Audit Notes
  - Security audit found no user-input SQL string concatenation in the Go backend; GORM calls and raw execution paths reviewed in this pass use placeholders or structured query APIs.
  - Dependency checks: `govulncheck ./...` reported 0 reachable vulnerabilities; `npm --prefix tests audit --audit-level=moderate` reported 0 vulnerabilities.
  - Fixed: favicon discovery now blocks localhost, private, special-use, link-local, multicast, and unspecified targets before dispatch; the default resolver transport bypasses environment proxies and rejects non-public DNS resolutions before dialing.
  - Optimized: favicon refresh remains isolated in `pkg/favicon` plus `SiteFaviconManager`; scheduled scans now queue from favicon metadata without selecting cached icon blobs for every site.
  - Added: opt-in live favicon integration coverage validates Google, Wikipedia, GitHub, Apple, Microsoft, and Reddit through the real HTTP resolver via `make test-live-favicons`.
  - Fixed: public feedback/subscription/confirmation rate limiting now stores one bounded counter per client IP, prunes expired windows, and rejects new clients once the active-window counter map is full.
  - Optimized: dashboard site listing now batches feedback, subscriber, visit, and unique-visitor counts for all listed sites instead of issuing four count queries per site.
  - Optimized: added composite indexes for the hot site/time/path/visitor/device/timezone visit aggregations, site-created feedback/subscriber lists, and LA Sentry issue/occurrence list paths.
  - Remaining before closing M001: decide pagination or rollup strategy for unbounded message/subscriber/Sentry issue lists and all-row attribution/device scans, then capture before/after `EXPLAIN` evidence for each additional query rewrite.


## Features

- [x] [F001] (P0) Add a developer LA Sentry client type and protected monitoring surface.
  ### Summary
  Extend LoopAware with a Sentry-inspired developer monitoring surface owned by LoopAware. Treat `LA Sentry` as a first-class client type for developers, distinct from the existing feedback widget, subscribe form, and traffic pixel clients. This is not a generic public event endpoint and should not add Sentry as a commercial dependency.
  ### Product Decisions
  - Name the dashboard section `LA Sentry`.
  - Reserve the `/sentry/*` route namespace for developer monitoring.
  - Use `/sentry/errors` as the protected error ingestion endpoint.
  - Do not add `/public/errors`.
  ### Access Model
  `/sentry/errors` must not be public. The MVP should accept server-to-server submissions authenticated with a per-site/project ingest token or signed request header. The authenticated dashboard remains protected by the existing TAuth session model.
  Browser-side capture is out of MVP unless a non-secret protection model is explicitly designed, because browser scripts cannot keep ingest credentials private.
  ### Data Model
  Add first-class developer monitoring records rather than overloading feedback. Store grouped issues separately from raw occurrences.
  A developer issue should track:
  - Grouping key.
  - Title.
  - Status (`unresolved`, `resolved`, `ignored`).
  - Level.
  - Platform.
  - Environment.
  - Release.
  - First seen.
  - Last seen.
  - Occurrence count.
  A developer error occurrence should store:
  - Raw message.
  - Exception type.
  - Stack frames.
  - Request metadata.
  - User hash.
  - Tags.
  - Extra JSON.
  - Received timestamp.
  ### Ingest Contract
  Accept JSON shaped around error events: `site_id`, `event_id`, `timestamp`, `platform`, `environment`, `release`, `level`, `message`, `exception_type`, `stacktrace`, `request`, `user_hash`, `tags`, and `extra`.
  Validate required fields at the edge, normalize stack frames through smart constructors, compute a stable grouping key from exception type/message/top in-app frame, and reject unknown sites or invalid credentials before persistence.
  ### Dashboard Plan
  Add a `LA Sentry` tab to the existing static dashboard beside Feedback, Subscriptions, and Traffic. The first view should show issue title, level, environment, release, count, last seen, and status.
  The issue detail view should show the latest occurrence stack, request context, tags, and recent occurrences, with actions to resolve, reopen, or ignore an issue.
  ### Alert Plan
  Reuse the existing Pinguin notification path for first-seen and regressed issues. Do not email every occurrence. Add configurable alert policy later for threshold bursts such as `N occurrences in M minutes`.
  ### Developer Client Type
  Add a new `LA Sentry` client type for developer error monitoring. The client should be configured per site/project with a protected ingest endpoint, credentials, environment, release, and optional tags. It should submit developer error events to LoopAware without sharing secrets through browser-delivered code.
  Start with a small Go client/middleware that PoodleScanner can use: recover panics around HTTP handlers, submit explicit `CaptureError(ctx, err, attrs)` events, include request metadata, and support environment/release configuration.
  Add frontend/browser capture only after the protected-ingest model is clarified.
  ### Deliverables
  - First-class `LA Sentry` developer client type.
  - Server-side Go client/middleware for protected error capture.
  - Protected `/sentry/errors` backend contract.
  - Migrations/models for developer issues and occurrences.
  - Dashboard `LA Sentry` tab.
  - Issue grouping.
  - First-seen/regression notifications.
  - Black-box API/dashboard coverage.
  - Follow-up PoodleScanner Go client integration issue if needed.
  ### Docs/Refs
  - `cmd/server/routes.go`
  - `internal/api`
  - `internal/model`
  - `internal/storage/migrations.go`
  - `internal/notifications`
  - `web/app/index.html`
  ### Resolution
  Implemented protected LA Sentry ingest with per-site token rotation, grouped developer issues and occurrences, authenticated dashboard APIs, the dashboard LA Sentry tab, a Go client/middleware package, docs, and black-box API/dashboard coverage. `make ci` passed.
  Post-review hardening now retries concurrent first-occurrence races, atomically increments grouped issue counts, bounds browser rate-limit state, rejects spoofed browser payload URLs, and strips query/fragment secrets from Go middleware request metadata. `make ci` passed.
- [ ] [F002] (P1) {F001} Add a Node.js LA Sentry server client.
  ### Summary
  Provide a first-party Node.js package for protected server-side LA Sentry ingest. This client should target backend runtimes only and must not expose ingest tokens through browser-delivered code.
  ### Deliverables
  - Client configuration for endpoint, site ID, ingest token, environment, release, and default tags.
  - `captureError(error, attrs)` helper that submits the documented `/sentry/errors` payload.
  - Express-compatible middleware for request metadata and thrown error capture.
  - Package README with token handling guidance and a minimal integration example.
  - Black-box integration coverage against the LoopAware ingest API.
- [x] [F003] (P1) {F001} Add a Python LA Sentry server client.
  ### Summary
  Provide a first-party Python package for protected server-side LA Sentry ingest. This client should support common WSGI/ASGI service usage without requiring the commercial Sentry SDK.
  ### Deliverables
  - Client configuration for endpoint, site ID, ingest token, environment, release, and default tags.
  - `capture_error(error, attrs)` helper that submits the documented `/sentry/errors` payload.
  - WSGI and ASGI middleware for request metadata and uncaught exception capture.
  - Package README with token handling guidance and Flask/FastAPI examples.
  - Black-box integration coverage against the LoopAware ingest API.
  ### Resolution
  Added `clients/python/la_sentry` with validated config/capture dataclasses, explicit `capture_error`, WSGI/ASGI middleware, docs, and integration coverage that runs a real Python process against `/sentry/errors`. `make ci` passed.
- [ ] [F004] (P2) {F001} Add a Ruby LA Sentry server client.
  ### Summary
  Provide a first-party Ruby package for protected server-side LA Sentry ingest.
  ### Deliverables
  - Client configuration for endpoint, site ID, ingest token, environment, release, and default tags.
  - `capture_error(error, attrs)` helper that submits the documented `/sentry/errors` payload.
  - Rack middleware for request metadata and uncaught exception capture.
  - Package README with token handling guidance and Rails/Rack examples.
  - Black-box integration coverage against the LoopAware ingest API.
- [ ] [F005] (P2) {F001} Add a PHP LA Sentry server client.
  ### Summary
  Provide a first-party PHP package for protected server-side LA Sentry ingest.
  ### Deliverables
  - Client configuration for endpoint, site ID, ingest token, environment, release, and default tags.
  - `captureError(Throwable $error, array $attrs = [])` helper that submits the documented `/sentry/errors` payload.
  - PSR-15 middleware for request metadata and uncaught exception capture.
  - Package README with token handling guidance and framework-neutral examples.
  - Black-box integration coverage against the LoopAware ingest API.
- [ ] [F006] (P2) {F001} Add a Java/Kotlin LA Sentry server client.
  ### Summary
  Provide a first-party JVM package for protected server-side LA Sentry ingest.
  ### Deliverables
  - Client configuration for endpoint, site ID, ingest token, environment, release, and default tags.
  - `captureError(Throwable error, attrs)` helper that submits the documented `/sentry/errors` payload.
  - Servlet filter support for request metadata and uncaught exception capture.
  - Package README with token handling guidance and Java/Kotlin examples.
  - Black-box integration coverage against the LoopAware ingest API.
- [ ] [F007] (P2) {F001} Add a .NET LA Sentry server client.
  ### Summary
  Provide a first-party .NET package for protected server-side LA Sentry ingest.
  ### Deliverables
  - Client configuration for endpoint, site ID, ingest token, environment, release, and default tags.
  - `CaptureError(Exception error, attrs)` helper that submits the documented `/sentry/errors` payload.
  - ASP.NET Core middleware for request metadata and uncaught exception capture.
  - Package README with token handling guidance and minimal API/controller examples.
  - Black-box integration coverage against the LoopAware ingest API.
- [x] [F008] (P1) {F001} Add browser JavaScript LA Sentry capture with origin-bound ingest.
  ### Summary
  Implement the P001 browser design as a standalone browser harness that captures frontend errors without exposing the protected server-side ingest token.
  ### Product Decisions
  - Use `/sentry/browser-errors` for browser events so `/sentry/errors` remains token-protected server-to-server ingest.
  - Authenticate browser events with the configured site ID plus the site's `allowed_origin` rules, not a browser-visible secret.
  - Keep browser event request metadata minimized to sanitized URL, referrer, and user agent.
  - Keep source map upload and source-map resolution out of scope for this issue.
  ### Deliverables
  - `web/la-sentry.js` browser harness with automatic unhandled error/rejection capture and explicit `LASentry.captureError`.
  - Origin-bound browser ingest endpoint with public CORS preflight and rate limiting.
  - Browser integration page and black-box Playwright coverage.
  - README docs for browser setup and the non-secret protection model.
  ### Resolution
  Added `/sentry/browser-errors` with allowed-origin validation, rate limiting, JavaScript platform normalization, and minimized request metadata. Added `web/la-sentry.js`, a dashboard browser snippet, a browser integration page, docs, and black-box API/browser coverage. `make ci` passed.


## Planning
*do not implement yet*
