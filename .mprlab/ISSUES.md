# ISSUES

Working backlog for this repository. Keep it current and small. Use @issues-md-format.md for the canonical format.

- Status markers: `[ ]` open, `[!]` blocked (must include a `Blocked:` line), `[x]` closed.
- Hygiene: once a closed issue's consequences are reflected in code/tests and in user-facing docs, remove the entry from this file. Git history remains the record. (Recurring runbooks below are the exception: keep them open.)

## BugFixes

## Improvements

- [ ] [I001] (P1) Replace placeholder-only inputs with labeled fields in the static frontend.
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


## Features

- [x] [F001] (P0) Add a developer Sentry client type and protected monitoring surface.
  ### Summary
  Extend LoopAware with a Sentry-inspired developer monitoring surface owned by LoopAware. Treat `Sentry` as a first-class client type for developers, distinct from the existing feedback widget, subscribe form, and traffic pixel clients. This is not a generic public event endpoint and should not add Sentry as a commercial dependency.

  ### Product Decisions
  - Name the dashboard section `Sentry`.
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
  Add a `Sentry` tab to the existing static dashboard beside Feedback, Subscriptions, and Traffic. The first view should show issue title, level, environment, release, count, last seen, and status.

  The issue detail view should show the latest occurrence stack, request context, tags, and recent occurrences, with actions to resolve, reopen, or ignore an issue.

  ### Alert Plan
  Reuse the existing Pinguin notification path for first-seen and regressed issues. Do not email every occurrence. Add configurable alert policy later for threshold bursts such as `N occurrences in M minutes`.

  ### Developer Client Type
  Add a new `Sentry` client type for developer error monitoring. The client should be configured per site/project with a protected ingest endpoint, credentials, environment, release, and optional tags. It should submit developer error events to LoopAware without sharing secrets through browser-delivered code.

  Start with a small Go client/middleware that PoodleScanner can use: recover panics around HTTP handlers, submit explicit `CaptureError(ctx, err, attrs)` events, include request metadata, and support environment/release configuration.

  Add frontend/browser capture only after the protected-ingest model is clarified.

  ### Deliverables
  - First-class `Sentry` developer client type.
  - Server-side Go client/middleware for protected error capture.
  - Protected `/sentry/errors` backend contract.
  - Migrations/models for developer issues and occurrences.
  - Dashboard `Sentry` tab.
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
  Implemented protected Sentry ingest with per-site token rotation, grouped developer issues and occurrences, authenticated dashboard APIs, the dashboard Sentry tab, a Go client/middleware package, docs, and black-box API/dashboard coverage. `make ci` passed.

- [ ] [F002] (P1) {F001} Add a Node.js Sentry server client.
  ### Summary
  Provide a first-party Node.js package for protected server-side Sentry ingest. This client should target backend runtimes only and must not expose ingest tokens through browser-delivered code.

  ### Deliverables
  - Client configuration for endpoint, site ID, ingest token, environment, release, and default tags.
  - `captureError(error, attrs)` helper that submits the documented `/sentry/errors` payload.
  - Express-compatible middleware for request metadata and thrown error capture.
  - Package README with token handling guidance and a minimal integration example.
  - Black-box integration coverage against the LoopAware ingest API.

- [ ] [F003] (P1) {F001} Add a Python Sentry server client.
  ### Summary
  Provide a first-party Python package for protected server-side Sentry ingest. This client should support common WSGI/ASGI service usage without requiring the commercial Sentry SDK.

  ### Deliverables
  - Client configuration for endpoint, site ID, ingest token, environment, release, and default tags.
  - `capture_error(error, attrs)` helper that submits the documented `/sentry/errors` payload.
  - WSGI and ASGI middleware for request metadata and uncaught exception capture.
  - Package README with token handling guidance and Flask/FastAPI examples.
  - Black-box integration coverage against the LoopAware ingest API.

- [ ] [F004] (P2) {F001} Add a Ruby Sentry server client.
  ### Summary
  Provide a first-party Ruby package for protected server-side Sentry ingest.

  ### Deliverables
  - Client configuration for endpoint, site ID, ingest token, environment, release, and default tags.
  - `capture_error(error, attrs)` helper that submits the documented `/sentry/errors` payload.
  - Rack middleware for request metadata and uncaught exception capture.
  - Package README with token handling guidance and Rails/Rack examples.
  - Black-box integration coverage against the LoopAware ingest API.

- [ ] [F005] (P2) {F001} Add a PHP Sentry server client.
  ### Summary
  Provide a first-party PHP package for protected server-side Sentry ingest.

  ### Deliverables
  - Client configuration for endpoint, site ID, ingest token, environment, release, and default tags.
  - `captureError(Throwable $error, array $attrs = [])` helper that submits the documented `/sentry/errors` payload.
  - PSR-15 middleware for request metadata and uncaught exception capture.
  - Package README with token handling guidance and framework-neutral examples.
  - Black-box integration coverage against the LoopAware ingest API.

- [ ] [F006] (P2) {F001} Add a Java/Kotlin Sentry server client.
  ### Summary
  Provide a first-party JVM package for protected server-side Sentry ingest.

  ### Deliverables
  - Client configuration for endpoint, site ID, ingest token, environment, release, and default tags.
  - `captureError(Throwable error, attrs)` helper that submits the documented `/sentry/errors` payload.
  - Servlet filter support for request metadata and uncaught exception capture.
  - Package README with token handling guidance and Java/Kotlin examples.
  - Black-box integration coverage against the LoopAware ingest API.

- [ ] [F007] (P2) {F001} Add a .NET Sentry server client.
  ### Summary
  Provide a first-party .NET package for protected server-side Sentry ingest.

  ### Deliverables
  - Client configuration for endpoint, site ID, ingest token, environment, release, and default tags.
  - `CaptureError(Exception error, attrs)` helper that submits the documented `/sentry/errors` payload.
  - ASP.NET Core middleware for request metadata and uncaught exception capture.
  - Package README with token handling guidance and minimal API/controller examples.
  - Black-box integration coverage against the LoopAware ingest API.

## Planning
*do not implement yet*

- [ ] [P001] (P1) {F001} Design browser JavaScript Sentry capture without exposed secrets.
  ### Summary
  Define whether and how LoopAware should support browser-side JavaScript error capture for the Sentry surface. Browser code cannot keep per-site ingest tokens private, so this issue must resolve the protection model before any SDK is built.

  ### Questions
  - Should browser capture use signed short-lived envelopes, origin-bound project keys, a relay endpoint, sampling, rate limits, or another non-secret mechanism?
  - Which event fields are safe to collect from browser contexts by default?
  - How should source maps, user identifiers, and privacy-sensitive request context be handled?

  ### Deliverables
  - Proposed browser ingest authentication and abuse-control model.
  - Data minimization rules for browser events.
  - Decision on whether the follow-up implementation should be a standalone browser SDK, a widget extension, or both.
