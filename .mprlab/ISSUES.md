# ISSUES

Entries record newly discovered requests or changes.

Read @AGENTS.md (Workflow section), @POLICY.md, and relevant stack guides before implementing changes.

Completed non-recurring history and resolved dependency references live in `.mprlab/ISSUES_ARCHIVE.md`. Recurring `R` entries stay in this file even when they describe repeated maintenance passes.

Format: `- [ ] [B042] (P1) {I007} Title`

- `[ ]` open, `[-]` taken, `[!]` blocked, `[x]` closed.
- Blocked issues (`[!]`) must include a `Blocked:` line in the body.

## BugFixes

- [!] [B025] (P0) Restore production dashboard access after TAuth login.
  ### Summary
  The production dashboard can remain unauthenticated after the shared TAuth login handoff when the LoopAware API runtime drifts from the TAuth tenant cookie-name/signing-key contract or when the first dashboard API request observes a transient stale auth state before a full page refresh.
  ### Deliverables
  - Verify the live production login/dashboard path and identify the failing layer.
  - Align the deployed LoopAware API session cookie name with the TAuth `loopaware` tenant.
  - Add gateway-side validation so future deploy config cannot mismatch the two values.
  - Verify the local LoopAware suite and focused gateway config checks pass.
  ### Progress
  Identified the live production cookie contract on 2026-05-27: TAuth's `loopaware` tenant clears `app_session_loopaware` and `app_refresh_loopaware` with `Domain=mprlab.com`, `SameSite=None`, and `Secure`, while LoopAware static production config still points dashboard API traffic at `https://loopaware-api.mprlab.com` and TAuth traffic at `https://tauth-api.mprlab.com`.
  Added LoopAware-side auth recovery so a post-login dashboard API 401 calls TAuth `/auth/refresh` with the configured tenant and retries the API request once before redirecting to `/login`. Added black-box Playwright coverage for the stale-first-`/api/me` path that previously required a manual full page refresh. Extended `cmd/configaudit` so LoopAware `TAUTH_TENANT_ID`, `TAUTH_JWT_SIGNING_KEY`, and `TAUTH_SESSION_COOKIE_NAME` must match the matching TAuth tenant env values, and aligned the example env placeholders that the new audit exposed as drift. `make ci` passed.
  Follow-up 2026-06-02: isolated the long-idle login button symptom to an expired GIS nonce callback in shared `mpr-ui` and resolved it in B019 by publishing `mpr-ui` `v3.10.1`; B011 remains blocked on the separate production runtime-config deploy/verification item below.
  Follow-up 2026-07-02: B032 removed the exact configaudit value-matching rule. LoopAware configaudit now validates the runtime config schema, required keys, placeholder coverage, and required non-empty values, while concrete TAuth and Pinguin secret/cookie values remain operator-owned deployment inputs.
  Blocked: applying and verifying the corrected production runtime config still requires the `mprlab-gateway` deploy step that previously stopped at the interactive `Gateway sudo password:` prompt.
  ### Changed Files
  `cmd/configaudit/main.go`, `cmd/configaudit/main_test.go`, `configs/.env.loopaware.example`, `configs/.env.loopaware.computercat.example`, `tests/specs/header-auth-state.spec.js`, `web/app/index.html`, `.mprlab/ISSUES.md`.
- [!] [B042] (P0) Make the release, publish, and deploy lifecycle fail before durable external mutation.
  ### Summary
  The prior deployment dry-run pass was insufficient operational proof, and the top-level lifecycle had no phase-correct readiness gates. `make publish` could durably mutate GitHub, GHCR, and App Store Connect before discovering that Google Play was disabled or inaccessible; Android `--dry-run` validated only local artifacts and did not authenticate to the Android Publisher API. The deploy dry run could also be weakened by skip flags, and an explicitly selected stale release tag was not required to match repository `HEAD`.
  ### Deliverables
  - Add phase-correct `release-dry-run`, `publish-dry-run`, and `deploy-dry-run` gates; do not claim that unpublished artifacts can pass exact published-deployment verification.
  - Make `make publish` run the same publication preflight before its first external mutation.
  - Add an authenticated transient Google Play edit/track probe whose successful preflight requires confirmed deletion and that catches a disabled API, missing ADC scope, or missing release authority without uploading a bundle or changing a live track.
  - Validate iOS upload inputs, Android signing/artifact inputs, GitHub publication state, container tooling/auth, and npm publication identity before mutation.
  - Reject deploy dry-run skip flags and release tags that do not identify the exact local and remote default-branch commit.
  - Add black-box regressions for ordering, no-mutation behavior, and the previously late Google Play failure.
  ### Progress
  Added phase-correct `release-dry-run`, `publish-dry-run`, and strengthened `deploy-dry-run` gates. Release validation runs the same full CI command as the real release and every real artifact builder in a disposable directory, requires the exact nine-file payload set, locks mobile artifacts to the staging timestamp, reconstructs mobile/container/Pages and React Native inputs from the exact source commit, strips inherited Expo public configuration, and records/verifies the canonical mobile runtime, redirect, and signing identity. It parses the shared env file as strict allowlisted data instead of executable Bash, requires local default-branch and stable-tag state to match origin, and recognizes only the exact already-prepared release on a repeat invocation. Lifecycle goals pin Make to `/bin/sh`, normalize supported inputs as literal values, require canonical Bash 4+, and reject Make no-execute/error-ignore/touch/question modes, shell/startup-hook overrides, exported Bash functions/options, Python/Node startup-path overrides, Docker environment overrides, and raw argument fragments. Git routing requires exactly one canonical origin fetch URL, rejects `remote.origin.pushurl` and `GH_REPO`, verifies that effective fetch/push URLs remain canonical after Git URL rewriting, and permits only an empty `GH_HOST` or `github.com`. Docker-backed integration rejects inherited product destinations, env files, Compose names, remote Docker contexts, and Docker TLS inputs before forcing its own localhost/test stack; release/publication container boundaries independently require a local `unix://` or `npipe://` Docker endpoint.
  Publication validation reverifies each prepared artifact at its mutation boundary, enforces canonical GitHub/GHCR/App Store/Play/npm destinations, proves that the exact container archive loads before GitHub mutation, proves GHCR write authority with confirmed transient-session cleanup, validates the IPA through App Store Connect, and performs an unchanged Play track `PUT` inside a transient edit with confirmed cleanup. The Play check lists existing bundles and rejects used/non-monotonic version codes and active rollouts; real publication replaces the internal track with one exact completed release containing the new version code, commits with `ERROR_IF_IN_REVIEW`, and post-verifies the committed AAB hash and exact single-release track state in a second edit. The edit probe requires a dedicated automation identity because creating an edit can invalidate another edit for that app and identity. Existing npm packages prove token write authority through a verified idempotent public-visibility write; an unbootstrapped package fails closed. npm exact-version integrity, public visibility, and `latest` are post-verified, and a downgrade is rejected before the write probe. GitHub Release metadata/assets, including the container descriptor, and versioned GHCR references are immutable; exact reruns preserve them, missing GitHub assets or a missing version index can be completed from verified content, and any content/digest mismatch or extra image platform fails closed. Container publication rejects an existing malformed or multiplatform `latest` index before rewrite, captures the registry push digest, builds version/`latest` indexes only from `image@digest`, and verifies version, platform, and latest manifests against that digest.
  One Git-common lifecycle lock serializes the phases, and one manifest digest is held through preflight, every provider stage, and the final completion-attestation stage. Store uploads run last, legacy Apple credential aliases are rejected, and ambiguous or partial cross-store success emits an explicit no-blind-retry failure. Only after every provider succeeds does publication upload and redownload-verify deterministic `publication.json`, bound to the manifest digest and completed stages; deploy requires that exact attestation. A completed publish rerun is verification-only, while attestation-upload recovery is an explicit attestation-only action after provider inspection so single-use store uploads are not repeated. Deploy validation rejects partial flags, repository impostors, weakened selections, and mutable provenance; it verifies a once-downloaded release manifest/publication attestation/container descriptor/Pages archive against the source and registry config, pins the registry digest, and validates the app-owned inventory, runtime assets, Compose render, and Ansible task bundle before production contact. Black-box tests cover incomplete inventory, source/runtime/timestamp drift, real and dry-run mode isolation, lock/manifest drift, Make/env/startup/function injection, remote branch/tag and effective Git destination drift, local Docker isolation, prepared-release idempotence, publication completion/recovery, GitHub/GHCR immutability, npm downgrade/latest behavior, ordering, cleanup failure, disabled Play API, Play write denial, used and non-monotonic Play codes, active rollouts, exact Play track replacement and review handling, post-commit verification failure, container load/push provenance, post-preflight checkout mutation, canonical iOS/npm behavior, and the non-deploying app-owned Ansible preflight.
  ### Validation
  Current shell/Node/Python syntax checks, mobile config/typecheck checks, `git diff --check`, and the aggregate `make release-workflow-check` pass. A fresh `make ci` passed config audit, builds, mobile/API boundaries, every focused lifecycle contract, the static workflow validator, test TypeScript checks, Go vet, Go unit tests, and Go race tests, then exited 126 when the managed Codex sandbox denied the integration launcher's `ps` call. An isolated `bash -x` run proved the launcher had accepted the local Unix Docker socket and failed at that first `ps`, before Docker Compose or any product test started. Direct invocations of `make release-dry-run`, `make publish-dry-run`, and `make deploy-dry-run` each failed at the lifecycle lock because this managed workspace intentionally keeps `.git` read-only; the serializer correctly requires a writable Git common directory. No release, publication, provider artifact, or production deployment was attempted.
  Blocked: lifecycle closure requires these LoopAware changes merged into the LoopAware default branch, a new provenance-labeled release prepared and published from that source, and all three phase gates passing in order for that same release. Until then, the complete release-to-production lifecycle is not operationally proven.
  ### Changed Files
  `PLAN.md`, `.dockerignore`, `.mprlab/ISSUES.md`, `Makefile`, `README.md`, `configs/.env.loopaware.computercat.example`, `configs/.env.loopaware.example`, `mobile/scripts/build-android-bundle.mjs`, `mobile/scripts/build-ios-archive.mjs`, `mobile/scripts/publish-android-play.mjs`, `mobile/scripts/submit-ios.mjs`, `mobile/scripts/validate-mobile-config.mjs`, `scripts/deploy.sh`, `scripts/publish-mobile.sh`, `scripts/publish-preflight.sh`, `scripts/publish-react-native.sh`, `scripts/publish-release.sh`, `scripts/publish.sh`, `scripts/release.sh`, `scripts/release-preflight.sh`, `scripts/release/deploy_pages_artifact.sh`, `scripts/release/docker_identity.sh`, `scripts/release/load_release_env.sh`, `scripts/release/parse_release_env.py`, `scripts/release/prepare_container_artifact.sh`, `scripts/release/prepare_release.sh`, `scripts/release/publish_container_artifacts.sh`, `scripts/release/publish_release.sh`, `scripts/release/record_publication.sh`, `scripts/release/release_helper.py`, `scripts/release/repository_identity.sh`, `scripts/release/run_lifecycle.sh`, `scripts/release/verify_staged_artifacts.py`, `scripts/release/with_lifecycle_lock.sh`, `scripts/test-deploy-dry-run.sh`, `scripts/test-ios-npm-publication.sh`, `scripts/test-lifecycle-orchestration.sh`, `scripts/test-publish-preflight.sh`, `scripts/test-release-tooling.sh`, `scripts/test-staged-release-artifacts.sh`, `scripts/validate-release-workflow.mjs`, and `tests/scripts/run-integration.sh`.


## Improvements

- [!] [I035] (P1) Adopt the schema-v3 sibling-gateway lifecycle.
  Goal:
  Make `.mprlab/deploy/resources.yml` the only production lifecycle declaration for LoopAware.
  Requirements:
  - Declare the backend, retained data, runtime capability, public route, health check, Pages site, npm package, mobile application, and TAuth tenant through current typed resources.
  - Keep private bytes in the ignored `.mprlab/deploy/.env` input and bind them only through `private_values`.
  - Remove app-owned Ansible, production Compose, release, publication, Pages activation, and deployment implementations.
  - Delegate the exact zero-argument `make release`, `make publish`, and `make deploy` phases to `../mprlab-gateway`.
  Validation:
  Run the full LoopAware CI gate and non-mutating gateway plans for all three lifecycle phases.
  Blocked: completing the gateway-only cutover still depends on coordinated changes in the sibling `../mprlab-gateway` repository and a follow-up owner decision for the post-B081 mobile publication boundary.
  Historical Resolution:
  Replaced the schema-1 dispatch stub with the complete schema-v3 resource graph
  and delegated the three production lifecycle targets to the exact sibling
  gateway. The original schema-v3 implementation introduced an EAS contract,
  advanced the npm client to `0.7.52`, excluded the private deployment input
  from Git and Docker contexts, and preserved local Compose plus mobile
  development, native identity, API-boundary, and config validation.
  Validation Results:
  The clean pre-change `make ci` baseline passed all 458 Playwright/API scenarios. Focused `make mobile-check`, `make config-audit`, and `go test ./cmd/configaudit` passed. Sealed gateway `plan-app-release`, `plan-app-publish`, and `plan-app-deploy` validation passed against the candidate committed bytes and a secret-free private-input fixture. Final `make ci` passed all Go, JavaScript, mobile, package, race, config-audit, and 458 Playwright/API scenarios after restoring the still-current local mobile invariants. Prepared the ignored mode-`0600` deployment input from the existing private LoopAware and TAuth values, verified the six exact nonempty bindings, and did not log secret bytes. No release, publication, or production deployment was run.
  Current Contract:
  B081 removes the EAS dependency introduced by the original schema-v3 change.
  Expo remains the local native-project generator; signed IPA/AAB construction
  and direct store publication are repository-owned local operations.
  Changed Files:
  `.dockerignore`, `.github/workflows/ci.yml`, `.gitignore`, `.mprlab/ISSUES.md`, `.mprlab/deploy/resources.yml`, `.mprlab/deploy/ansible/**`, `.mprlab/deploy/docker-compose.yml`, `CHANGELOG.md`, `Makefile`, `README.md`, `clients/react-native/package.json`, `clients/react-native/package-lock.json`, `cmd/configaudit/main.go`, `configs/.env.loopaware.example`, `configs/.env.loopaware.computercat.example`, `mobile/eas.json`, `mobile/package.json`, `mobile/package-lock.json`, `mobile/scripts/validate-mobile-config.mjs`, removed mobile production build/publish scripts, and removed app-owned lifecycle scripts under `scripts/` and `scripts/release/`.
- [ ] [I023] (P1) Consider a design of a current accordion design of different surfaces.
  We may want to have a better split out.
- [ ] [I027] (P1) {I028} Replace placeholder-only inputs with labeled fields in the static frontend.
  Duplicate: `web/app/index.html` still contains placeholder-only inputs, so keep I028 as the canonical implementation issue and do not execute I027 separately. Retain I027 only as a cross-reference until lifecycle ownership can remove the duplicate ID.
- [ ] [I028] (P1) Replace placeholder-only inputs with labeled fields in the static frontend.
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

- [ ] [M400R] (P2) Backlog hygiene and archive.
  Goal:
  Keep the issue tracker reliable, readable, and focused on active work while preserving resolved history in the appropriate archive.
  Requirements:
  - Cadence: run weekly during active development and before each release cut.
  - Validate section names, identifier prefixes, recurrence suffixes, priority markers, dependencies, and duplicate IDs against the current `issues-md-format.md`.
  - Reconcile stale statuses, duplicate issues, broken references, obsolete instructions, and entries filed under the wrong section.
  - Move completed non-recurring history to the repository issue archive or durable documentation when the active tracker becomes noisy.
  - Keep active, blocked, planning, and recurring entries visible in `ISSUES.md`.
  Deliverables:
  - Normalized `ISSUES.md` structure and statuses.
  - Updated issue archive or docs when completed entries are removed from the active tracker.
  - A short `Last run:` note summarizing the cleanup and any follow-up issues filed.
  Validation:
  - Re-read `ISSUES.md` after edits and confirm every issue is under the right section with a unique section-aware ID.
  - Confirm recurring entries remain open and keep the `R` suffix.
  - Confirm no active, blocked, recurring, or planning work was archived.
  Last run: 2026-08-09.
- [ ] [M401R] (P2) Polish open issues.
  Goal:
  Keep unresolved work executable by making each open issue concrete, ordered, and testable.
  Requirements:
  - Cadence: run weekly during active development and before handing a repo to automated execution.
  - Review every unresolved non-recurring issue for missing context, dependencies, repro steps, acceptance criteria, and validation expectations.
  - Make priorities concrete and ensure each open issue has actionable deliverables.
  - Merge duplicate open issues or add explicit dependency links when separate entries must remain.
  - Do not close or implement issues as part of this polish pass unless that work is separately requested.
  Deliverables:
  - Open issues with enough detail for a person or agent to execute without rediscovery.
  - New or updated dependency markers where ordering matters.
  - A short `Last run:` note listing the number of issues polished and any blockers found.
  Validation:
  - Sample the open entries after the pass and confirm each has clear next actions and validation expectations.
  - Confirm no recurring runbook was marked complete.
  - Confirm duplicates were merged or explicitly cross-referenced.
- [ ] [M402R] (P2) Architecture and policy review.
  Goal:
  Catch architecture, policy, and workflow drift before it becomes hidden maintenance debt.
  Requirements:
  - Cadence: run monthly, before large refactors, and after major framework or runtime changes.
  - Review the codebase, docs, and workflow against `AGENTS.md`, `POLICY.md`, stack guides, and the current architecture notes.
  - Look for drift from forward-only contracts, edge-validation boundaries, smart-constructor usage, testing policy, and module ownership.
  - Record findings as new Maintenance issues with concrete scope, priority, and validation.
  - Close the pass with a no-action note only when the review finds no actionable drift.
  Deliverables:
  - New Maintenance issues for each actionable architecture or policy drift finding.
  - Updated notes on areas reviewed and areas intentionally left unchanged.
  - A short `Last run:` note with the review scope and outcome.
  Validation:
  - Confirm every finding is represented as an issue with owner-readable context and validation criteria.
  - Confirm no implementation changes were mixed into the review runbook unless separately requested.
  - Confirm all recurring runbooks remain open.
- [ ] [M403R] (P1) Dependency and security audit.
  Goal:
  Keep third-party dependencies, runtime versions, and security-sensitive configuration within the current supported contract.
  Requirements:
  - Cadence: run weekly for active apps and before each release cut.
  - Inspect package managers, lockfiles, language toolchains, container bases, and generated clients for known vulnerabilities or stale direct dependencies.
  - Review auth, secret, CORS, CSP, SQL, network, and permission-sensitive configuration for drift from the current contract.
  - Prefer current supported dependencies; do not add compatibility shims for obsolete dependency behavior.
  - File separate Maintenance or BugFix issues for each actionable vulnerability, unsupported runtime, or security-contract gap.
  Deliverables:
  - Documented audit commands or data sources used for the pass.
  - Updated issues for each actionable dependency or security finding.
  - A short `Last run:` note with clean result or follow-up issue IDs.
  Validation:
  - Rerun the repository-native audit, lint, or dependency checks used for the pass.
  - Confirm every finding is either filed, fixed under a separate issue, or explicitly marked not applicable with evidence.
  - Confirm no secrets or private payloads were written into the tracker.
  Last run: 2026-08-03. Baseline and final `make ci` runs passed; the final gate included vulnerability scans, policy audits, race coverage, and all 452 browser/API scenarios. Source review, live-safe exploit reproduction, exact-version extended CodeQL analysis, released-image inspection, and authenticated GitHub control queries produced BugFixes B069-B078. B069-B077 are resolved. B078's local fixes and evidence dispositions are complete, while 11 live alerts against the unchanged remote `master` commit remain open until the corrected source is pushed and default setup reruns. Dependabot and secret scanning report zero open alerts. No secret values were written into the tracker, and no release, publication, or production deployment was run.
- [ ] [M404R] (P1) CI, release, and artifact health.
  Goal:
  Keep the repository's validation, release, publication, and generated artifact surfaces trustworthy.
  Requirements:
  - Cadence: run before every release, publish, or deploy, and weekly for critical services.
  - Verify repository-native CI, lint, format, coverage, release, publish, Docker image, Pages, and artifact workflows still match the documented contract.
  - Check generated artifacts, release tags, published images, and Pages outputs for source-to-public drift.
  - File concrete follow-up issues for failing gates, stale artifacts, missing release prerequisites, or undocumented workflow changes.
  - Do not perform production deployment from this runbook unless the operator explicitly requests that deployment.
  Deliverables:
  - Recorded gate status and artifact surfaces inspected.
  - Follow-up issues for each reproducible CI, release, publish, or artifact drift problem.
  - A short `Last run:` note with commands run and any skipped surfaces.
  Validation:
  - Use repository-native `make` targets or documented release helpers for checks.
  - Confirm release and deployment ownership boundaries remain separate.
  - Confirm public or published artifacts match the intended source revision when that surface is inspected.
- [ ] [M405R] (P1) Code contract and static hygiene.
  Goal:
  Keep source contracts explicit, current, and statically guarded against policy drift.
  Requirements:
  - Cadence: run monthly and before large refactors.
  - Scan for dead code, unused exports, duplicated literals, silent fallbacks, legacy aliases, compatibility reads, and zero-but-invalid domain states.
  - Check static analysis, coverage, schema, and contract guards that are supposed to prevent drift.
  - File focused Maintenance issues for each concrete violation instead of broad cleanup placeholders.
  - Keep the current canonical contract only; do not preserve obsolete behavior unless a product requirement explicitly says so.
  Deliverables:
  - Issue entries for each actionable static hygiene or contract violation.
  - Notes on static tools, searches, and contract guards used during the pass.
  - A short `Last run:` note with clean result or follow-up issue IDs.
  Validation:
  - Rerun the relevant static checks, contract tests, or repository searches used to identify drift.
  - Confirm every finding has a narrow follow-up issue and does not duplicate existing backlog work.
  - Confirm no implementation changes were mixed into the audit unless separately requested.
- [ ] [M406R] (P1) Production drift and health.
  Goal:
  Detect when production, public, or scheduled runtime state has drifted from the intended repository contract.
  Requirements:
  - Cadence: run weekly for deployed services and after each publish or deploy.
  - Compare current source, runtime configuration, published images, public routes, scheduled jobs, and health checks for drift.
  - Inspect real operator-facing surfaces rather than assuming merged source is deployed.
  - File follow-up issues for stale images, stale Pages output, missing routes, failed monitors, invalid production config, or undocumented runtime differences.
  - Stop before production deploy or destructive operator actions unless the operator explicitly requests them.
  Deliverables:
  - Recorded source revision, public artifact, route, image, or health surfaces inspected.
  - Follow-up issues for each source-to-runtime drift finding.
  - A short `Last run:` note with evidence links or commands used.
  Validation:
  - Verify inspected production or public surfaces directly where access is available.
  - Confirm any deploy-required finding is filed with the exact publish/deploy boundary and owner.
  - Confirm no production state was changed by the audit unless explicitly requested.
- [ ] [M407R] (P2) Documentation and runbook hygiene.
  Goal:
  Keep durable documentation and runbooks aligned with the current behavior users and operators actually rely on.
  Requirements:
  - Cadence: run before release cuts and after merge bursts that change user-facing or operator-facing behavior.
  - Review README, ARCHITECTURE, PRD, CHANGELOG, docs, runbooks, setup guides, and local workflow notes for stale behavior or missing new contracts.
  - Update docs when closed issues changed durable behavior, public APIs, operator workflows, release semantics, or deployment expectations.
  - Remove or rewrite stale instructions instead of preserving obsolete alternatives.
  - File separate issues for documentation gaps that require product or implementation decisions.
  Deliverables:
  - Updated documentation or filed follow-up issues for each gap.
  - A short `Last run:` note listing docs inspected and changes made.
  - Cross-references from archived issue history to durable docs when useful.
  Validation:
  - Check links, command names, paths, and public contract descriptions touched by the pass.
  - Confirm docs describe the current canonical path only.
  - Confirm issue archive and active tracker references remain consistent.
- [ ] [M001R] (P0) Audit and harden SQL queries for security, correctness, and performance.
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

- [ ] [F002] (P1) {F001} Add a Node.js LA Sentry server client.
  ### Summary
  Provide a first-party Node.js package for protected server-side LA Sentry ingest. This client should target backend runtimes only and must not expose ingest tokens through browser-delivered code.
  ### Deliverables
  - Client configuration for endpoint, site ID, ingest token, environment, release, and default tags.
  - `captureError(error, attrs)` helper that submits the documented `/sentry/errors` payload.
  - Express-compatible middleware for request metadata and thrown error capture.
  - Package README with token handling guidance and a minimal integration example.
  - Black-box integration coverage against the LoopAware ingest API.
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


## Planning
*do not implement yet*

