# ISSUES

Entries record newly discovered requests or changes.

Read @AGENTS.md (Workflow section), @POLICY.md, and relevant stack guides before implementing changes.

Format: `- [ ] [B042] (P1) {I007} Title`

- `[ ]` open, `[!]` blocked, `[x]` closed.
- Blocked issues (`[!]`) must include a `Blocked:` line in the body.

## BugFixes

- [x] [B001] (P1) Stabilize seeded dashboard auth after login redirects.
  ### Summary
  A release run timed out in `dashboard-allowed-origins.spec.js` because `openDashboard()` reached `/login` while waiting for the MPR UI testing auth helper, then waited for dashboard account fields that cannot exist on the landing page.
  ### Deliverables
  - Keep authenticated dashboard helpers on the requested protected page after seeded MPR UI auth succeeds.
  - Preserve the existing MPR UI testing helper contract instead of using private auth globals.
  - Verify the allowed-origins browser coverage and the relevant lint/typecheck gate.
  ### Resolution
  Updated `openAuthenticatedPage()` so seeded MPR UI auth recovers when a protected-page boot redirects to `/login` before the testing helper can authenticate the header. The helper now clears the app logout marker left by that test-only redirect, reopens the original authenticated path, and retries seeded MPR UI auth before dashboard readiness checks run. Validation passed with `make lint-js`, `make test-integration` with 425 Playwright/API integration specs, and final `make ci` with 425 Playwright/API integration specs.
  ### Changed Files
  `PLAN.md`, `.mprlab/ISSUES.md`, `tests/helpers/fixtures.js`.
- [x] [B002] (P1) Ignore external-asset route fetches cancelled by test teardown.
  ### Summary
  The Playwright external asset stub can still be handling a same-origin document route when the final browser context closes, causing an otherwise-complete integration run to fail with `route.fetch: Target page, context or browser has been closed`.
  ### Deliverables
  - Treat route errors caused by Playwright page/context/browser teardown as non-actionable cleanup.
  - Continue surfacing real route failures while the page is still active.
  - Verify the formerly failing widget test page coverage and the full CI gate.
  ### Resolution
  Added a narrow teardown classifier around tracked external-asset route handling so Playwright page/context/browser closure during test cleanup no longer fails an otherwise complete run, while non-teardown route errors still propagate. Focused validation passed with `make lint-js` and `env LOOPAWARE_BASE_URL=http://localhost:8090 npm --prefix tests run test -- specs/widget-test-page.spec.js`. Full validation passed with `make ci`, including 396 Playwright/API integration specs.
  ### Changed Files
  `PLAN.md`, `tests/helpers/externalAssets.js`, `.mprlab/ISSUES.md`.
- [x] [B003] (P1) Keep Timezones labels aligned with geographic markers.
  ### Summary
  The selected-site Timezones map offsets labels far enough from their projected markers that Los Angeles can read as though it belongs around New York, and visit counts compete with the geographic label text instead of living inside the bubbles.
  ### Deliverables
  - Render visit counts inside the Timezones circles.
  - Keep geographic labels centered on their projected marker positions while preserving readable label boxes.
  - Add dashboard coverage for Los Angeles marker placement, bubble-count text, and label alignment.
  ### Resolution
  Updated the Timezones SVG renderer so each bubble renders its visit count centered inside the circle, while the geographic label box stays centered on the projected marker and only moves vertically for collision avoidance. Added dashboard coverage for Los Angeles projected placement, in-circle count text, and label-to-marker x-axis alignment. Validation passed with baseline `make ci`, `make lint-js`, final `make ci`, and a local visual dashboard check using Los Angeles 75 / New York 39 / Unknown 46 seeded traffic.
  ### Changed Files
  `PLAN.md`, `web/app/index.html`, `tests/specs/dashboard-traffic.spec.js`, `.mprlab/ISSUES.md`.
- [x] [B004] (P2) Show native feedback context in the operator mobile app.
  ### Summary
  The operator mobile app checks feedback rows for `source_kind: "mobile"`, but the backend returns native feedback with the canonical `source_kind: "mobile_app"`, causing native feedback rows to appear as Web widget feedback.
  ### Deliverables
  - Align the operator app feedback display condition with the backend mobile feedback source kind.
  - Keep the mobile feedback message type constrained to the backend source-kind contract.
  - Verify the mobile TypeScript gate and full CI pass.
  ### Resolution
  Updated the operator mobile app feedback row detail condition to use the backend `mobile_app` source kind and narrowed `FeedbackMessage.source_kind` to the current backend source-kind union. Validation passed with baseline `make ci`, focused `make mobile-check`, and final `make ci` with 425 Playwright/API integration specs.
  ### Changed Files
  `PLAN.md`, `.mprlab/ISSUES.md`, `mobile/App.tsx`, `mobile/src/types.ts`.
- [x] [B005] (P2) Proxy LA Sentry routes in the local gHTTP template.
  ### Summary
  The local gHTTP env template routes `/public/` and `/api/` to the LoopAware API but omits `/sentry/`, so copied local stacks send `/sentry/errors` and `/sentry/browser-errors` to the static server instead of the backend LA Sentry handlers.
  ### Deliverables
  - Add the `/sentry/` proxy route to `configs/.env.ghttp.example`.
  - Keep the route target aligned with the local compose API service name.
  - Verify config audit and the full CI gate pass.
  ### Resolution
  Added `/sentry/=http://loopaware:8080` to the local gHTTP env template so copied local stacks route LA Sentry server and browser ingest requests to the LoopAware API, matching the existing test and computercat proxy templates. Validation passed with `make config-audit` and final `make ci` with 425 Playwright/API integration specs.
  ### Changed Files
  `PLAN.md`, `.mprlab/ISSUES.md`, `configs/.env.ghttp.example`.
- [x] [B006] (P0) Replace incorrect four-hour login test with console-clean stale-idle coverage.
  ### Summary
  The existing four-hour login regression asserts an internal prepared-nonce refresh strategy, so it can pass while the user still sees login failure after leaving `/login` open for several hours.
  ### Deliverables
  - Remove the implementation-specific four-hour login tests that inspect refreshed Google Identity nonces.
  - Add black-box browser coverage that loads `/login` with stale auth-restore state, emulates four hours of page idleness, then signs in through the visible header control.
  - Assert the flow produces no console warnings/errors, no hidden auth error events, and no failed auth boundary responses.
  ### Resolution
  Removed the nonce-refresh-specific four-hour login tests and replaced them with one browser scenario that seeds stale TAuth restore state, advances the page clock by four hours, signs in through the header Google control, and asserts the corrected session/nonce contract plus zero console warnings/errors.
  Tightened the Google and TAuth test stubs so `/auth/google` succeeds only when the stub Google credential is bound to the submitted `nonce_token`, matching the stricter shared-auth/TAuth security contract.
  After `mpr-ui@latest` resolved to `3.10.4`, the focused stale-idle regression and canonical auth-state suite passed: the stale restore path uses `/auth/session`, performs no legacy `/me` or `/auth/refresh` anonymous probes, performs no background nonce/GIS initialization work, and initializes GIS once with the click-time nonce.
  The LoopAware Google Identity stub helper now seeds auto-credential behavior before the GIS script is requested so the test can assert zero pre-click initialization without forcing the old background-load contract. The diagnostic treats only navigation-cancelled `/auth/session` checks as non-actionable while still failing on console problems, 4xx auth responses, hidden auth error events, legacy probes, repeated nonces, or missing nonce-bound credential exchange.
  Final validation passed with `make ci` on 2026-06-08, including the stale-idle auth coverage and 393 Playwright/API integration specs.
  ### Changed Files
  `.mprlab/ISSUES.md`, `PLAN.md`, `tests/helpers/externalAssets.js`, `tests/helpers/tauthStub.js`, `tests/specs/header-auth-state.spec.js`.
- [x] [B007] (P0) Make Google popup auth compatible with edge opener policy.
  ### Summary
  The long-idle login console trace includes a Google Identity popup warning: `Cross-Origin-Opener-Policy policy would block the window.postMessage call`. LoopAware’s tracked proxy hardening headers do not pin a COOP policy, so a live edge that applies strict `same-origin` opener isolation can break or warn on the GIS popup’s opener communication.
  ### Deliverables
  - Set the LoopAware proxy COOP policy to `same-origin-allow-popups` across local, computercat, and test stacks.
  - Add regression coverage proving static and proxied responses carry that popup-compatible policy.
  - Keep the auth failure path visible through the shared auth events rather than relying on browser console noise.
  ### Resolution
  Added `Cross-Origin-Opener-Policy: same-origin-allow-popups` to the local, computercat, and test gHTTP proxy header configuration so GIS popup opener communication is explicitly allowed instead of inheriting a stricter edge default. Updated the security-header regression to assert the COOP value for both static frontend documents and proxied API responses. Tests: `make lint-js`; `LOOPAWARE_TEST_SUITE=test make test-integration` (278 passed); `make ci` (394 integration specs passed).
  ### Changed Files
  `.mprlab/ISSUES.md`, `PLAN.md`, `docker-compose.yml`, `docker-compose.computercat.yml`, `tests/docker-compose.yml`, `tests/specs/security-headers.spec.js`.
- [x] [B008] (P0) Cover landing-page login after four idle hours.
  ### Summary
  A user can leave the `/login` landing page open for hours, return, and find that the visible Google sign-in no longer completes login.
  ### Deliverables
  - Add black-box browser coverage that loads `/login`, emulates four hours passing after page load, then attempts the visible header Google sign-in.
  - Keep this change test-only; do not edit production application code.
  - Verify focused header auth coverage and report whether the scenario passes or reproduces the bug.
  ### Resolution
  Added `tests/specs/header-auth-state.spec.js` coverage for a single visible header sign-in attempt after `/login` has stayed loaded for four emulated hours. The test initially reproduced the bug against the previously released shared auth library. The systemic fix shipped in `mpr-ui` `v3.10.3`, jsDelivr `mpr-ui@latest` now resolves to `x-jsd-version: 3.10.3`, and the LoopAware scenario advances the Playwright browser clock so the shared prepared-nonce refresh timers execute before the single sign-in click. Production LoopAware code remains untouched. Tests: `make lint-js`; `make test-integration-all` (394 passed); `make ci` (394 integration tests passed).
  ### Changed Files
  `.mprlab/ISSUES.md`, `PLAN.md`, `tests/specs/header-auth-state.spec.js`.
- [x] [B009] (P1) Use `mpr-ui` Google Identity testing helper instead of stub globals.
  ### Summary
  LoopAware auth specs still mutate and inspect the local Google Identity stub global directly, even though this behavior belongs behind `mpr-ui`'s public test-only integration contract.
  ### Deliverables
  - Implement the `google.accounts.id.__mprUiTesting` adapter in the LoopAware GIS route stub.
  - Route all auto-credential, initialized nonce, and initialize-count test access through `MPRUI.testing.googleIdentity`.
  - Verify focused auth coverage and the full `make ci` gate after the supporting `mpr-ui` helper is available from CDN.
  ### Resolution
  Released `mpr-ui` `v3.10.2` with the `MPRUI.testing.googleIdentity` adapter for test-only Google Identity drivers, verified jsDelivr `mpr-ui@latest` resolves to `x-jsd-version: 3.10.2`, and routed LoopAware's GIS route stub plus auth specs through that public helper instead of app-owned stub globals. Focused auth coverage passed with 83 specs, then final `make ci` passed with 393 Playwright/API specs.
  ### Changed Files
  `.mprlab/ISSUES.md`, `PLAN.md`, `tests/helpers/externalAssets.js`, `tests/specs/header-auth-state.spec.js`, `tests/specs/logout-hardening.spec.js`.
- [x] [B010] (P1) Share Google Identity stub readiness across auth specs.
  ### Summary
  GitHub Actions still fails in `logout-hardening.spec.js` because that spec has a second local `enableAutoGoogleCredentialOnClick` helper that mutates the Google Identity stub before the stub has installed its initialized nonce state.
  ### Deliverables
  - Move the Google Identity initialized-nonce wait into a shared test asset helper.
  - Use the shared wait in both header auth and logout hardening sign-in helpers before mutating stub state.
  - Verify focused logout hardening coverage plus the full `make ci` gate.
  ### Resolution
  Added shared `waitForGoogleIdentityStubInitialized` coverage support in the external asset helper, then reused it from both header auth and logout hardening credential helpers before setting `autoCredentialOnClick`. Focused `logout-hardening` plus `header-auth-state` coverage passed, and final `make ci` passed with 393 Playwright/API specs.
  ### Changed Files
  `.mprlab/ISSUES.md`, `PLAN.md`, `tests/helpers/externalAssets.js`, `tests/specs/header-auth-state.spec.js`, `tests/specs/logout-hardening.spec.js`.
- [x] [B011] (P1) Stabilize auth browser harness readiness in CI.
  ### Summary
  GitHub Actions intermittently fails while opening authenticated dashboard pages because seeded `mpr-ui` test authentication can evaluate during a transient navigation, and the long-idle Google nonce regression can mutate the Google Identity stub before the stub state has been installed.
  ### Deliverables
  - Retry seeded `mpr-ui` browser-session authentication only across transient navigation/context swaps.
  - Wait for the Google Identity stub and initialized nonce config before long-idle login tests mutate or read stub state.
  - Verify focused dashboard allowed-origin and header-auth coverage plus the full `make ci` gate.
  ### Resolution
  Made the seeded `mpr-ui` browser authentication helper retry only transient Playwright navigation/context-loss errors between readiness and `page.evaluate`, and made header auth tests wait for the Google Identity stub's initialized nonce config before toggling auto credential behavior or reading nonce state. Focused coverage for `dashboard-allowed-origins` and `header-auth-state` passed, then final `make ci` passed with 393 Playwright/API specs.
  ### Changed Files
  `.mprlab/ISSUES.md`, `PLAN.md`, `tests/helpers/fixtures.js`, `tests/specs/header-auth-state.spec.js`.
- [x] [B012] (P1) Restore feedback bubble color customization.
  ### Summary
  Operators can still adjust feedback bubble placement and feedback input visibility, but the dashboard and widget test page no longer expose a way to customize the feedback bubble color.
  ### Deliverables
  - Add a persisted widget accent color setting with edge validation.
  - Expose the color control in the dashboard Feedback widget card and widget test page.
  - Return the saved accent through public widget configuration and apply it to the rendered bubble.
  - Add black-box API, dashboard, and widget coverage.
  ### Resolution
  Added `widget_accent_color` as a persisted site setting with lowercase `#rrggbb` edge validation, migration backfill, admin/public API exposure, dashboard autosave support, widget-test page editing and preview, and runtime widget application to the feedback bubble and Send button. Added API, dashboard, widget runtime, and widget-test page coverage. Baseline and final `make ci` passed.
  ### Changed Files
  `.mprlab/ISSUES.md`, `README.md`, `internal/api/admin.go`, `internal/api/public.go`, `internal/api/public_additional_test.go`, `internal/api/public_test.go`, `internal/model/models.go`, `internal/storage/database.go`, `internal/storage/database_test.go`, `internal/storage/migrations.go`, `tests/specs/api-admin.spec.js`, `tests/specs/api-public.spec.js`, `tests/specs/dashboard-labels.spec.js`, `tests/specs/dashboard-site-actions.spec.js`, `tests/specs/widget-integration.spec.js`, `tests/specs/widget-test-page.spec.js`, `web/app/index.html`, `web/app/widget-test/index.html`, `web/widget.js`.
- [x] [B013] (P0) Recover login after long-idle Google nonce expiry.
  ### Summary
  Leaving a public LoopAware page open long enough for the prepared Google/TAuth nonce to expire can make the next header sign-in popup complete visually while the credential exchange stays unauthenticated.
  ### Deliverables
  - Reproduce the long-idle login path through black-box browser coverage.
  - Fix the shared CDN-hosted `mpr-ui` auth controller so expired GIS callback nonces cannot reach `/auth/google`.
  - Verify LoopAware recovers by preparing a fresh nonce and completing the next sign-in.
  ### Resolution
  Published `mpr-ui` `v3.10.1`, which timestamps prepared GIS nonces, rejects expired callback nonces before TAuth credential exchange, emits `mpr-ui.auth.stale_nonce`, and primes a fresh nonce for the next sign-in attempt. Added LoopAware Playwright coverage that advances the page clock past nonce freshness, verifies the stale click does not call `/auth/google`, then verifies the next click exchanges the refreshed nonce and reaches `/app`. Verified jsDelivr `mpr-ui@latest` resolves to `x-jsd-version: 3.10.1`, then ran `make test-integration-all` and `make ci`.
  ### Changed Files
  `tests/specs/header-auth-state.spec.js`, `.mprlab/ISSUES.md`.
- [x] [B014] (P0) Restore GitHub Actions browser setup before CI timeout.
  ### Summary
  The GitHub Actions `test` job cancels before `make ci` starts because `npm --prefix tests run install:browsers` spends the 15-minute job budget installing every Playwright browser and Linux dependency even though the suite runs Chromium-only.
  ### Deliverables
  - Install only Chromium for Playwright browser coverage.
  - Keep CI dependency installation locked to the tracked package lockfile.
  - Preserve the local `make ci` gate and verify the full suite still passes.
  ### Resolution
  Changed GitHub Actions to install test dependencies with `npm ci`, run Playwright against the runner's system Chrome instead of downloading bundled Chromium, disable Playwright video capture when a system browser channel is selected so cached Playwright ffmpeg is not required, increased the job timeout to 30 minutes, and kept the integration-script fallback aligned with the configured browser channel. Verified `npm --prefix tests ci`, the system-Chrome Playwright configuration, and full local `make ci` with 383 Playwright/API tests passing.
  ### Changed Files
  `.github/workflows/ci.yml`, `tests/playwright.config.js`, `tests/package.json`, `tests/scripts/run-integration.sh`, `.mprlab/ISSUES.md`.
- [x] [B015] (P0) Verify successful login lands on a loaded dashboard.
  ### Summary
  Add black-box browser coverage for the full login completion path: an unauthenticated user starts from `/login`, completes Google/TAuth sign-in, receives a usable session, reaches `/app`, and sees the authenticated dashboard loaded.
  ### Deliverables
  - Add Playwright integration coverage that drives the login page sign-in flow rather than pre-seeding a session before navigation.
  - Assert the final dashboard URL and visible authenticated dashboard state.
  - Fix any auth handoff regression exposed by the new coverage.
  ### Resolution
  Added black-box Playwright coverage for the login CTA completing TAuth exchange, receiving a session cookie, reaching `/app`, waiting for the loaded dashboard, and verifying authenticated header/user state. The TAuth test stub now writes the post-exchange session cookie so the browser test matches the real login handoff. `make ci` passed.
- [x] [B016] (P0) Make IP rate limits independent of wall-clock bucket boundaries.
  ### Summary
  `make ci` exposed an intermittent LA Sentry browser rate-limit failure where requests started near the end of a 30-second wall-clock bucket could split across buckets and avoid the intended per-window limit.
  ### Deliverables
  - Use per-client rate windows that start at the first request in the window.
  - Apply the same boundary-safe behavior to public and browser Sentry rate limits.
  - Verify the black-box integration suite no longer lets the boundary case through.
  ### Resolution
  Replaced wall-clock bucket counters with per-client windows for public API and LA Sentry browser rate limits, updated helper tests, and verified `make test-integration-api` plus `make ci` pass.
- [x] [B017] (P1) Fix subscription confirmation brand navigation.
  ### Summary
  The LoopAware brand link on subscription token pages points to `#top`, so confirming a subscription leaves the user trapped on the confirmation page through the natural header flow.
  ### Deliverables
  - Make subscription token page brand links navigate to LoopAware instead of scrolling the current page.
  - Route unauthenticated users to `/login` and authenticated users to `/app`.
  - Add black-box browser coverage for the public-page brand navigation behavior.
  ### Resolution
  Replaced subscription token page brand `#top` links with auth-aware LoopAware home links, reused shared header auth state to update public-page brand destinations to `/login` or `/app`, added Playwright coverage for signed-out and signed-in public-page brand navigation, and verified `make ci` passes.
- [x] [B018] (P0) Make production landing login use a real Google sign-in control.
  ### Summary
  The production `/login` dashboard CTAs intercept clicks and programmatically re-click the header Google sign-in target. Real Google sign-in is rendered inside a cross-origin iframe, so this delegated click path can lose the user's direct activation and leave the user on the landing page instead of opening the sign-in flow.
  ### Deliverables
  - Replace landing dashboard CTA triggers with first-class `mpr-login-button` controls.
  - Remove the delegated dashboard-login click bridge that programmatically clicks the header Google sign-in target.
  - Add black-box browser coverage that signs in from the landing-page control and reaches the loaded dashboard.
  ### Resolution
  Replaced the landing dashboard CTA anchors with real `mpr-login-button` controls, removed the programmatic dashboard-login click bridge from `web/header-auth.js`, scoped runtime auth bootstrap so the login-page header no longer creates a competing Google controller, updated the Google test stub to expose a clickable rendered sign-in button, added black-box landing-login coverage through the loaded dashboard, and verified `make ci` passes.
- [x] [B019] (P0) Use config-first mpr-ui/TAuth authentication.
  ### Summary
  LoopAware still bootstraps authentication with app-owned `tauth.js` loading, direct TAuth helper globals, manual `tauth-*` attributes, and multiple login controls on `/login`. This interferes with mpr-ui's shared auth lifecycle and can trigger duplicate `/me`/`/auth/refresh` probes plus repeated Google Identity initialization.
  ### Deliverables
  - Serve `/config-ui.yaml` and let `mpr-ui-config.js` apply auth attributes before loading the mpr-ui bundle.
  - Remove direct `tauth.js` loading and app-owned Google/TAuth helper orchestration.
  - Keep LoopAware code limited to public auth events, redirects, and product-specific overlays.
  - Add black-box browser coverage that the login page has a single Google auth controller.
  ### Resolution
  Moved LoopAware auth configuration to `/config-ui.yaml`, switched served pages to the `mpr-ui-config.js` and bundle-marker flow using `mpr-ui@latest`, removed direct `tauth.js`/TAuth helper globals, and kept app code to public mpr-ui auth events plus product redirects/overlays. Fixed the shared `mpr-ui` nonce lifecycle bug that mismatched GIS nonce tokens after unauthenticated `/me`/`/auth/refresh` bootstrap, published the fix upstream, updated black-box coverage for the single auth controller, TAuth credential exchange, and logout failure recovery, and verified `make ci` passes.
- [x] [B020] (P0) Keep the login page Google sign-in in the header actions.
  ### Summary
  The `/login` page needs the visible Google sign-in control in the right side of the header without making LoopAware own Google/TAuth bootstrap or creating a second mpr-ui auth controller.
  ### Deliverables
  - Render the login page Google sign-in as a shared `mpr-ui` control inside the header action area.
  - Keep `/login` on the config-first `mpr-ui@latest` path with no app-owned TAuth bootstrap.
  - Preserve single-controller coverage for `/me`, `/auth/refresh`, and Google Identity initialization.
  ### Resolution
  Kept `/login` on the canonical `<mpr-header data-config-url="/config-ui.yaml">` path so the built-in header Google button remains in the right-side header actions. Fixed and published `mpr-ui` so nested header user menus mirror header auth events/state instead of starting their own profile bootstrap, preserving a single mpr-ui auth owner for `/me`, `/auth/refresh`, and Google Identity initialization. Updated Playwright coverage for header-right placement, TAuth config ownership, credential exchange, and the single auth controller.
- [x] [B021] (P0) Remove LoopAware-owned Google auth scaffolding.
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
- [x] [B022] (P0) Gate timeout logout redirect on successful response.
  ### Summary
  The session-timeout logout flow redirects to `/login` after any resolved `/auth/logout` fetch response. HTTP 4xx/5xx responses still resolve, so a failed server logout can move users off `/app` while their server session remains valid.
  ### Deliverables
  - Redirect to the landing page only after `/auth/logout` returns a successful response.
  - Keep failed timeout logout attempts on the authenticated dashboard with visible, recoverable UI state.
  - Add black-box browser coverage for the failed logout response path.
  ### Resolution
  Updated the dashboard timeout logout request to throw on non-OK `/auth/logout` responses, recover the dashboard overlay state on failure, and restart the idle manager when the session-timeout flow remains on `/app`. Added black-box browser coverage for a failed session-timeout logout response and verified the full `make ci` gate passes.
- [x] [B023] (P0) Restore dashboard allowed-origin browser coverage stability.
  ### Summary
  Full `make ci` times out in the dashboard allowed-origin browser tests before the suite can complete. The failure appears before the B008 changed path and was reproduced in both the pre-change baseline and post-change CI attempt.
  ### Deliverables
  - Reproduce the `specs/dashboard-allowed-origins.spec.js:88` timeout in isolation.
  - Identify whether the failure belongs to dashboard auth settling, the allowed-origin UI flow, or the Playwright harness setup.
  - Restore the full `make ci` gate.
  ### Resolution
  Moved seeded browser-auth synchronization to the public `MPRUI.testing` helper exposed by the shared header package and released that helper through CDN-hosted mpr-ui. Updated the harness to match the current mpr-ui no-anonymous-probe contract. While restoring the full gate, fixed a follow-on dashboard autosave race so disabling both widget feedback controls shows the validation error immediately and stale autosave success responses no longer hide the invalid state. `make ci` passed.
- [x] [B024] (P0) Enforce CDN-only shared UI assets.
  ### Summary
  LoopAware must never carry a local `tools/mpr-ui` checkout or test-time vendored shared UI bundle; all third-party and shared UI assets must come from CDN-hosted URLs.
  ### Deliverables
  - Remove the local `tools/mpr-ui` symlink from the workspace.
  - Remove local/shared-asset fallback and test-time CDN patching from the Playwright asset harness.
  - Add a CI guard that fails when `tools/mpr-ui` exists.
  - Verify the failing dashboard/auth browser cases pass against CDN-hosted `mpr-ui@latest`.
  ### Resolution
  Removed the `tools/mpr-ui` symlink, removed local mpr-ui reads and test-time asset patching from `tests/helpers/externalAssets.js`, and added a config-audit rule with coverage that rejects `tools/mpr-ui`. Published `mpr-ui` tag `v3.9.7` so `mpr-ui@latest` exposes `MPRUI.testing`, then verified the 27 dashboard/auth cases from the failed run pass against CDN-only assets.
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
- [x] [B026] (P0) Restore integration Pinguin startup after config schema drift.
  ### Summary
  Baseline `make ci` no longer reaches the API and browser integration suites because the Pinguin service in `tests/docker-compose.yml` rejects LoopAware's bundled Pinguin config. The runtime now treats TAuth identity metadata as a shared-shell concern and fails on the stale `tenants[].identity` shape.
  ### Deliverables
  - Update the LoopAware-owned Pinguin config fixture to match the current Pinguin runtime schema.
  - Keep TAuth provider metadata in the TAuth/shared-shell config path, not inside Pinguin tenant config.
  - Preserve email notification test wiring for the LoopAware tenant.
  - Verify the API integration slice starts successfully before proceeding with reporting work.
  ### Resolution
  Removed stale Pinguin tenant identity metadata from the LoopAware Pinguin config, added the current disabled SMTP submission/forwarding section defaults to the tracked test and example envs, and removed the obsolete config-audit Google-provider invariant. Verified the API integration slice passes with 108 tests and the full `make ci` gate passes with 358 integration tests.
- [x] [B027] (P0) Scope scheduled report device and timezone totals to the report window.
  ### Summary
  Weekly traffic report emails can show a seven-day page-view total while Devices and Timezones list larger all-time visit totals, making the report internally inconsistent.
  ### Deliverables
  - Use the same report-window filter for scheduled email Devices and Timezones as the headline counts and Top pages.
  - Keep dashboard all-time device/timezone breakdown endpoints unchanged.
  - Add regression coverage proving old visits do not inflate the weekly email breakdown sections.
  ### Resolution
  Added window-scoped device and timezone statistics methods that share the VisitTrend UTC day window. Scheduled traffic report emails now use those windowed breakdowns while dashboard device/timezone endpoints keep their existing all-time behavior. Added regression coverage proving old visits do not appear in weekly email Devices, Timezones, Top pages, or headline totals. `make ci` passed.
  ### Changed Files
  `internal/api/site_stats.go`, `internal/api/traffic_report_schedule.go`, `internal/api/traffic_report_schedule_test.go`, `internal/api/admin_helpers_test.go`, `internal/api/admin_test.go`, `.mprlab/ISSUES.md`.
- [x] [B028] (P1) Fix global traffic report review findings.
  ### Summary
  Review found that the global traffic report page can hide report-definition load failures, schedule edits can overwrite saved timezones with the browser timezone, and the locked default report name appears like a broken editable field.
  ### Deliverables
  - Preserve report-definition load errors without loading downstream report data that clears the visible failure.
  - Preserve saved selected-site and all-sites schedule timezones when editing other schedule fields.
  - Render the built-in all-sites report name as an explicit read-only default state while keeping custom reports nameable.
  - Add black-box dashboard coverage for the corrected behaviors.
  ### Resolution
  Report-definition load failures now stop the all-sites reporting load chain so downstream stats cannot clear the visible error. Traffic report schedule responses expose whether they are persisted, letting the dashboard seed unsaved defaults from the browser timezone while preserving saved selected-site and all-sites timezones on later edits. The built-in all-sites report now renders as an explicit read-only default report while custom reports keep the editable name input. Added black-box API and dashboard coverage for these paths. `make ci` passed.
  ### Changed Files
  `internal/api/portfolio_traffic_report.go`, `internal/api/traffic_report_schedule.go`, `tests/specs/api-admin.spec.js`, `tests/specs/dashboard-traffic.spec.js`, `web/app/index.html`, `.mprlab/ISSUES.md`.
- [x] [B029] (P1) Remove aggregate top-pages sections from all-sites traffic reports.
  ### Summary
  All-sites traffic reports rank bare URL paths across unrelated properties, which makes the Top pages section noisy and misleading for aggregate reports even though it remains useful for individual-site reports.
  ### Deliverables
  - Remove top-pages data and sections from all-sites report API/dashboard/email output.
  - Keep individual-site traffic report Top pages unchanged.
  - Add regression coverage proving all-sites reports summarize sites without aggregate top-pages output.
  ### Resolution
  Removed aggregate Top pages output from portfolio traffic report API responses, dashboard rendering, and scheduled email templates while keeping selected-site Top pages unchanged. Added API, dashboard, element, and email-template regression coverage proving all-sites reports omit aggregate top-pages output. `make ci` passed.
  ### Changed Files
  `internal/api/portfolio_traffic_report.go`, `internal/api/templates/portfolio_traffic_report_email.txt`, `internal/api/portfolio_traffic_report_test.go`, `tests/specs/api-admin.spec.js`, `tests/specs/dashboard-elements.spec.js`, `tests/specs/dashboard-traffic.spec.js`, `web/app/index.html`, `.mprlab/ISSUES.md`.
- [x] [B030] (P1) Keep dashboard SSE streams alive through gateway read timeouts.
  ### Summary
  Production dashboard SSE streams for favicon and feedback updates can sit idle longer than the gateway upstream read timeout, causing Chrome to report `ERR_HTTP2_PROTOCOL_ERROR` when the proxy closes the stream.
  ### Deliverables
  - Emit backend SSE comment heartbeats more frequently than the gateway 80s read timeout.
  - Cover favicon, feedback, and subscription test SSE streams with handler tests that prove heartbeat frames are written.
  - Verify backend stream coverage without relying on slow production-duration timers.
  ### Resolution
  Added 30s `: heartbeat` comment frames to the favicon, feedback, and subscription test SSE loops, with focused handler coverage using shortened test intervals. `go test ./internal/api` and `go test ./...` pass.
  The I012 chart-scale assertion blocker was removed by targeting the rendered SVG text labels directly; focused `dashboard-traffic.spec.js` coverage passed. Full `make ci` now passes with 383 Playwright/API tests.
  ### Changed Files
  `internal/api/admin.go`, `internal/api/site_subscribe_test_handlers.go`, `internal/api/stream_handlers_test.go`, `.mprlab/ISSUES.md`.
- [x] [B031] (P1) Ignore disposed external-asset responses during browser teardown.
  ### Summary
  Baseline `make ci` failed in the widget test page browser coverage after the spec completed because the external-asset route handler attempted to read a same-origin document response that Playwright had already disposed during page/context teardown.
  ### Deliverables
  - Treat disposed route responses caused by Playwright teardown as non-actionable cleanup.
  - Continue surfacing real external-asset route failures while the page remains active.
  - Verify the formerly failing widget test page coverage and the full CI gate.
  ### Resolution
  Extended the external asset route teardown classifier to include Playwright's disposed response error after baseline `make ci` failed in the completed widget test page cleanup path. Final validation passed with `make ci`, including 430 Playwright/API integration specs.
  ### Changed Files
  `PLAN.md`, `.mprlab/ISSUES.md`, `tests/helpers/externalAssets.js`.
- [x] [B032] (P0) Remove exact cross-service value checks from configaudit.
  ### Summary
  `make deploy` failed during its local `make ci` preflight because `cmd/configaudit` still enforced `loopaware.auth.tauth.jwt_signing_key must match tauth.TAUTH_TENANT_JWT_SIGNING_KEY_LOOPAWARE`, even though deployment config validation should verify schemas and required values without hardcoding the exact operator-owned TAuth/Pinguin values.
  ### Deliverables
  - Remove configaudit equality checks for LoopAware/TAuth JWT signing key, session cookie name, tenant ID, Pinguin auth token, and Pinguin/TAuth signing key values.
  - Keep placeholder coverage, runtime YAML loading, schema checks, and required non-empty value validation.
  - Add coverage proving cross-service operator value drift is accepted by configaudit.
  - Verify the deploy-blocking configaudit command and the full LoopAware gate.
  ### Resolution
  Removed the cross-service invariant comparison pass from `cmd/configaudit`; the audit still renders and validates `configs/config.loopaware.yml` through `serverconfig.LoadWithLookup`, so missing placeholders and empty required runtime fields remain failures. Added configaudit coverage that intentionally gives LoopAware, TAuth, and Pinguin different concrete signing key, session cookie, and token values while expecting the audit to pass. Validation passed with focused configaudit tests, `go run ./cmd/configaudit`, and final `make ci`.
  ### Changed Files
  `PLAN.md`, `.mprlab/ISSUES.md`, `CHANGELOG.md`, `cmd/configaudit/main.go`, `cmd/configaudit/main_test.go`, `cmd/configaudit/main_additional_test.go`.
- [x] [B033] (P0) Align the deploy entrypoint with `.mprlab/deploy` artifacts.
  ### Summary
  The deploy metadata migration moved app-owned deployment resource artifacts under `.mprlab/deploy/`, but `make deploy`, `scripts/deploy.sh`, and README still defaulted to `deploy/app.yml`. That made the deployment entrypoint incompatible with the current governance path and left the old `deploy/ansible/resources.yml` location in the tracked tree.
  ### Deliverables
  - Move the app deploy manifest to `.mprlab/deploy/app.yml`.
  - Keep the resource manifest under `.mprlab/deploy/resources.yml`.
  - Update the Makefile, deploy wrapper, README, and changelog to describe the current deployment artifact path.
  - Verify a non-deploying `make deploy` invocation resolves the `.mprlab/deploy/app.yml` manifest.
  ### Resolution
  Moved `deploy/app.yml` to `.mprlab/deploy/app.yml`, kept the app-owned resource manifest under `.mprlab/deploy/resources.yml`, and updated the default `APP_MANIFEST`, deploy script help/defaults, and README deploy docs to point at `.mprlab/deploy/app.yml`. Validation passed with `bash -n scripts/deploy.sh`, a non-deploying `make deploy DEPLOY_ARGS="--skip-ci --skip-image-verify --skip-backend --skip-pages"`, `make config-audit`, `git diff --check`, and final `make ci`.
  ### Changed Files
  `PLAN.md`, `.mprlab/ISSUES.md`, `.mprlab/deploy/app.yml`, `.mprlab/deploy/resources.yml`, `CHANGELOG.md`, `Makefile`, `README.md`, `scripts/deploy.sh`, `deploy/app.yml`, `deploy/ansible/resources.yml`.
- [x] [B034] (P1) Show trailing 24-hour traffic for the 1-day interval.
  ### Summary
  The dashboard 1-day traffic interval can show no trend, source, and engagement data even when visits occurred within the last 24 hours, because the interval is sourced from a day-bucket query instead of a rolling 24-hour window.
  ### Deliverables
  - Treat `interval=1day` as the past 24 hours across selected-site traffic endpoints and CSV export.
  - Preserve the 30-day and all-time interval semantics.
  - Add black-box API coverage proving visits inside the trailing 24-hour window appear and older visits do not.
  ### Resolution
  Replaced the selected-site and all-sites `1day` traffic window with a trailing 24-hour cutoff while preserving UTC day-bucket behavior for longer intervals and all-time reporting. Added API coverage for 23-hour versus 25-hour visits across dashboard stats, top pages, attribution, and trend output. Review follow-up removed the generated Playwright console log from the branch and changed scheduled traffic report email totals to read aggregate page-view and unique-visitor counts, with coverage for split daily trend buckets. Validation passed with focused `go test`, full `go test ./internal/api`, `git diff --check`, and final `make ci`.
  ### Changed Files
  `PLAN.md`, `.mprlab/ISSUES.md`, `README.md`, `.playwright-cli/console-2026-07-03T21-41-36-761Z.log`, `internal/api/admin.go`, `internal/api/admin_test.go`, `internal/api/portfolio_traffic_report.go`, `internal/api/site_stats.go`, `internal/api/site_stats_additional_test.go`, `internal/api/traffic_report_schedule.go`, `internal/api/traffic_report_schedule_test.go`.
- [x] [B035] (P0) Recover release publishing when the current tag image is missing.
  ### Summary
  Running `make release && make publish` from a clean `master` already tagged as `v0.7.39` stops in release after the full CI gate because there are no changelog notes for `v0.7.40`, while `make deploy` later fails because `ghcr.io/tyemirov/loopaware:v0.7.39` is missing. The release cycle should make the existing-tag publish repair path explicit and fail deploy before expensive CI when the required image is absent.
  ### Deliverables
  - Make `make release` a successful no-op when `HEAD` is already covered by the current release tag, so chained `make release && make publish` can repair the image for that existing tag.
  - Make deploy image verification report a stable `make publish` recovery message when the release image is not published.
  - Verify deploy image presence before rerunning deployment CI.
  - Add a CI guard for the release/publish/deploy workflow contract and update the README runbook.
  ### Resolution
  Made `make release` exit successfully before CI when `HEAD` is already covered by the current release tag, so `make release && make publish` can repair Docker images for that existing tag instead of attempting an empty release. Moved deploy image verification before the deployment CI gate and wrapped missing registry image inspection failures with an explicit `make publish` recovery message. Added `release-workflow-check` to the Makefile lint path to guard release idempotency, deploy image verification ordering, and README recovery documentation. Validation passed with baseline `make ci`, `bash -n scripts/release.sh scripts/publish.sh scripts/deploy.sh`, `make release-workflow-check`, a missing-image deploy probe using `make deploy DEPLOY_ARGS="--skip-ci --image ghcr.io/tyemirov/loopaware-nonexistent-b035"`, `git diff --check`, `make lint-js`, and final `make ci`.

  Current contract note: Superseded by B039 and B042. Release always prepares a new canonical artifact set; publication repair and provider mutation belong to the separate publish phase.
  ### Changed Files
  `PLAN.md`, `.mprlab/ISSUES.md`, `Makefile`, `README.md`, `scripts/deploy.sh`, `scripts/release.sh`, `scripts/validate-release-workflow.mjs`.
- [x] [B036] (P0) Fail iOS store submit before archive when App Store Connect app id is missing.
  ### Summary
  `make release` can spend the full iOS archive/export path and only then fail in `submit-ios` with `iOS upload requires MOBILE_IOS_ASC_APP_ID or LOOPAWARE_MOBILE_IOS_ASC_APP_ID`, because the upload identity is validated after `build-ios` completes.
  ### Deliverables
  - Validate App Store Connect upload identity and credentials before `make submit-ios` builds the IPA.
  - Validate the same iOS upload inputs before `make release` creates a repository release when iOS upload is enabled.
  - Keep `make build-ios` usable as a build-only artifact command.
  - Add a mobile config guard for the pre-archive submit preflight.
  - Update the mobile publishing runbook and verify the missing-id path fails before archive work.
  ### Resolution
  Added `submit-ios-preflight` to validate App Store Connect upload identity and credentials without requiring a built IPA, changed `submit-ios` to run that preflight before invoking `build-ios`, and made `make release` run the same iOS preflight before `make ci` and release creation when iOS upload is enabled. Kept `make build-ios` as a build-only artifact path. Updated mobile config validation and README release/mobile publishing docs to lock in the pre-archive and pre-release upload-input checks. Validation passed with baseline `make ci`, `bash -n scripts/release.sh scripts/publish.sh scripts/deploy.sh`, `node --check mobile/scripts/submit-ios.mjs`, `npm --prefix mobile run validate-config`, missing-id `make submit-ios` preflight proof that did not reach `build-ios`, positive `submit-ios.mjs --preflight-only` proof with a dummy numeric ASC app id, a fake-helper `scripts/release.sh` proof that stopped at `submit-ios-preflight` before `make ci`, `git diff --check`, `make lint-js`, and final `make ci`.

  Current contract note: Superseded by B039 and B042. Release builds the exact IPA; App Store Connect validation and upload authority are publication-phase checks.
  ### Changed Files
  `PLAN.md`, `.mprlab/ISSUES.md`, `Makefile`, `README.md`, `scripts/release.sh`, `mobile/scripts/submit-ios.mjs`, `mobile/scripts/validate-mobile-config.mjs`.
- [x] [B037] (P0) Load the repository env before release mobile store preflight.
  ### Summary
  After B036 moved iOS upload validation before archive work, `make release` still failed with a missing App Store Connect app id because the release path did not load the local `configs/.env.loopaware` file before running the mobile store preflight.
  ### Deliverables
  - Load the canonical local LoopAware env file before mobile store preflight and submit work in `make release`.
  - Keep the already-tagged release no-op path ahead of store env requirements so `make release && make publish` still repairs existing release images.
  - Export the loaded env to recursive `make submit-ios` and `make submit-android` invocations.
  - Add guards and docs for the release env contract.
  ### Resolution
  Added `RELEASE_ENV_FILE` with the canonical default `configs/.env.loopaware`, changed `make release` to pass that env file to `scripts/release.sh`, and made the release script parse and export the dotenv entries before native mobile store preflight and submit work. The env load remains after the existing already-tagged no-op check, so `make release && make publish` can still repair existing release images without requiring store upload inputs. Added the LoopAware App Store Connect app id to the tracked LoopAware env templates, merged the config-directory README guidance into the root README, removed `configs/README.md`, and guarded the release env contract in both release workflow and mobile config validation. Also added the non-secret app id to the ignored local `configs/.env.loopaware` on this machine. Validation passed with baseline `make ci`, `bash -n scripts/release.sh scripts/publish.sh scripts/deploy.sh`, `make release-workflow-check`, `npm --prefix mobile run validate-config`, `git diff --check`, release env harness proofs for local and temporary `RELEASE_ENV_FILE` loading before iOS preflight, `make lint-js`, and final `make ci`.

  Current contract note: Superseded by B039 and B042. The env file still supplies build/publication credentials, but store authority and upload no longer run inside release.
  ### Changed Files
  `PLAN.md`, `.mprlab/ISSUES.md`, `CHANGELOG.md`, `Makefile`, `README.md`, `docs/LA-116-split-frontend-backend.md`, `configs/README.md`, `configs/.env.loopaware.example`, `configs/.env.loopaware.computercat.example`, `scripts/release.sh`, `scripts/validate-release-workflow.mjs`, `mobile/scripts/validate-mobile-config.mjs`.
- [x] [B038] (P0) Keep release workflow validation inside the repository checkout.
  ### Summary
  GitHub Actions fails in `release-workflow-check` because `scripts/validate-release-workflow.mjs` reads `../agentSkills/gitrelease/scripts/prepare_release.sh`, a machine-local sibling path that is absent from the LoopAware checkout.
  ### Deliverables
  - Remove direct reads of shared Git Release implementation files from LoopAware CI.
  - Validate the tracked release and prepared-release publish adapter contracts at the repository boundary.
  - Keep missing shared release tooling fail-fast at the operator entrypoints without adding fallbacks or CI-only skips.
  - Verify the focused release workflow check, JavaScript lint path, and full CI gate.
  ### Resolution
  Removed the validator's direct read of the machine-local Git Release pipeline and replaced it with repository-owned contract checks for both `scripts/release.sh` and `scripts/publish-release.sh`. The checks now require each adapter's explicit pipeline override, canonical shared pipeline name, executable fail-fast gate, and argument-forwarding `exec`, while the shared Git Release package remains responsible for validating its own pipeline internals. No fallback, conditional skip, duplicate helper, or CI-only checkout was added. Validation passed with `bash -n scripts/release.sh scripts/publish-release.sh scripts/publish-mobile.sh scripts/publish-react-native.sh scripts/deploy.sh`, direct validator execution, `make release-workflow-check`, `git diff --check`, `make lint-js`, and final `make ci` with 454 Playwright/API integration specs.
  ### Changed Files
  `PLAN.md`, `.mprlab/ISSUES.md`, `scripts/validate-release-workflow.mjs`.
- [x] [B039] (P1) Make release behavior reproducible from the LoopAware checkout.
  Goal:
  Remove the mutable sibling repository from every release, publication, and Pages artifact path.

  Requirements:
  Keep preparation, publication, mobile-package verification, Pages activation, and container publication on the current repository-owned Git Release contract without changing production deploy or store-upload ownership.

  Deliverables:
  - Vendor the current container-and-Pages Git Release executables under `scripts/release/`.
  - Route the Makefile and every release wrapper exclusively through the owned directory.
  - Reject the former sibling path in the release validator.
  - Exercise the owned Pages builder and deployer with distinct source and release commits, an empty `.nojekyll`, source-marker acceptance, and release-commit-marker rejection.

  Validation:
  - `make release-pages-contract-check`
  - `make release-workflow-check`
  - `make test`
  - `make lint`
  - `make ci`

  ### Resolution
  Moved release preparation, container publication, mobile-package verification, and Pages artifact preparation/deployment into the LoopAware checkout and removed the mutable sibling-tool path from the release adapters and validator. Repaired the repository-owned Pages deployer so manifest parsing completes under the supported macOS Bash, added bounded black-box coverage for source-versus-release commit provenance and `.nojekyll`, and added a verification-only Pages artifact path that performs no branch mutation. Validation passed with `make release-pages-contract-check`, `make release-workflow-check`, `make test`, `make lint`, and final `make ci` with 454 Playwright/API integration specs.
  ### Changed Files
  `PLAN.md`, `.mprlab/ISSUES.md`, `Makefile`, `scripts/release/prepare_container_artifact.sh`, `scripts/release/prepare_pages_artifact.sh`, `scripts/release/prepare_release.sh`, `scripts/release/publish_container_artifacts.sh`, `scripts/release/publish_release.sh`, `scripts/release/release_helper.py`, `scripts/release/deploy_pages_artifact.sh`, `scripts/test-release-tooling.sh`, `scripts/validate-release-workflow.mjs`.

- [x] [B040] (P0) Make deployment readiness provable before production mutation.
  ### Summary
  `make deploy` reached independent late failures for a missing cross-repo env value and malformed gateway Ansible YAML, while the existing local preflight could still pass with mismatched LoopAware-to-Pinguin credentials, incomplete target dependency closure, silently dropped runtime assets, and an unreadable repository-owned Pages deployer. At discovery time, there was no single non-deploying command that exercised the exact published release and gateway target contract.
  ### Deliverables
  - Add one `make deploy-dry-run` entrypoint that validates the exact release tag, published container digests, published Pages artifact, and exact gateway LoopAware backend target without prompting for credentials or contacting production hosts.
  - Validate cross-repo Pinguin and TAuth identities, rendered Compose/Caddy config, selected runtime assets, and complete target dependency closure.
  - Make the real deploy validate its Pages artifact before any backend mutation.
  - Add black-box/static regressions proving the dry run cannot push Pages, prompt for sudo, or execute a remote Ansible play.
  - Document the guarantees and the remote-only checks that remain part of the real deploy/verify sequence.

  ### Resolution
  Replaced the mutable sibling-gateway handoff with a LoopAware-owned deployment controller, ignored operator inventory, production Compose file, and four-phase Ansible task bundle, all under `.mprlab/deploy/`. The dry run preserves the exact release, publication, immutable-image, Pages, and repository-state checks, then validates the app-owned inventory, private runtime env, config audit, and production Compose render without SSH or production contact. The real path reruns local validation before asking for become credentials, proves remote capacity and the shared network/volume/dependencies, verifies exact LoopAware-to-TAuth/Pinguin identities plus authenticated read-only canaries, recreates only `loopaware-api` at the verified digest, and verifies the running image, Pinguin connectivity, and public `/healthz` response before Pages activation. The gateway aggregate can consume the declared task bundle, but app deployment no longer locates, validates, locks, or executes gateway source.
  ### Layout Correction
  The first implementation incorrectly placed the new Compose and Ansible assets in a root `deploy/` directory even though LoopAware's canonical app-owned deployment boundary is `.mprlab/deploy/`. Moved the complete Compose/Ansible tree and ignored operator inventory under `.mprlab/deploy/`, corrected repository and staged-remote relative paths, updated every active consumer, and added a validator rejection for any root `deploy/` tree.
  ### Validation
  Passed shell/Python syntax checks, `git diff --check`, `make config-audit`, `make release-workflow-check`, `make test-unit`, `make deploy-dry-run-contract-check`, the pinned app-owned Ansible local preflight and deploy-playbook syntax check, the gateway generic app-resource validator with every relocated `.mprlab/deploy/ansible/tasks/*` entrypoint, live read-only TAuth/Pinguin dependency canaries, `make test-integration-api` with all 133 API tests, and final `make ci` with all 455 integration tests after the layout correction. No production apply or Pages mutation was run.
  ### Changed Files
  `PLAN.md`, `.gitignore`, `.mprlab/ISSUES.md`, `.mprlab/deploy/resources.yml`, `.mprlab/deploy/docker-compose.yml`, `.mprlab/deploy/ansible/ansible.cfg`, `.mprlab/deploy/ansible/inventory/hosts.yml.example`, `.mprlab/deploy/ansible/playbooks/deploy.yml`, `.mprlab/deploy/ansible/playbooks/preflight-local.yml`, `.mprlab/deploy/ansible/tasks/deploy.yml`, `.mprlab/deploy/ansible/tasks/preflight.yml`, `.mprlab/deploy/ansible/tasks/resolve-image.yml`, `.mprlab/deploy/ansible/tasks/validate.yml`, `.mprlab/deploy/ansible/tasks/verify.yml`, `Makefile`, `README.md`, `cmd/configaudit/main.go`, `cmd/server/main.go`, `cmd/server/routes.go`, `configs/.env.ghttp.computercat.example`, `configs/.env.ghttp.example`, `scripts/deploy.sh`, `scripts/run-app-ansible-deploy.sh`, `scripts/test-deploy-dry-run.sh`, `scripts/validate-release-workflow.mjs`, `scripts/verify-loopaware-dependency-contract.py`, `tests/configs/ghttp.env`, and `tests/specs/api-public.spec.js`.

- [x] [B041] (P1) Make logout redirect observation race-free in the browser harness.
  ### Summary
  Final `make ci` intermittently timed out in `logout-hardening.spec.js` after the page had already reached `/login`, because the shared helper registered its redirect waiter after a fast navigation and converted two expired waiters into an `AggregateError` without rechecking the current URL.
  ### Deliverables
  - Accept an already-completed login redirect before registering asynchronous waiters.
  - Recheck the current URL when both overlay/URL waiters expire so a completed navigation is not reported as a failure.
  - Verify the focused logout hardening path and final `make ci`.

  ### Resolution
  Changed the shared logout observer to accept a completed `/login` navigation before registering waiters and to recheck the current URL after both asynchronous signals expire. The focused 14-test logout hardening suite passed, followed by final `make ci` with all 454 Playwright/API integration specs.
  ### Changed Files
  `PLAN.md`, `.mprlab/ISSUES.md`, `tests/helpers/fixtures.js`.

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

- [x] [B043] (P1) Remove the undeclared `uv` dependency from release workflow validation.
  ### Summary
  PR #277's `Go CI / test` job reaches `make release-pages-contract-check` and fails with `/usr/bin/env: 'uv': No such file or directory`. The repository-owned release helper declares no third-party Python dependencies but uses `uv run --script` as its executable boundary, while the canonical GitHub Actions job installs Python but not `uv`.
  ### Deliverables
  - Make the release helper execute directly with the available Python 3 runtime.
  - Remove obsolete `UV_CACHE_DIR` plumbing from release, publish, preflight, and contract-test callers.
  - Add a regression that rejects reintroducing a `uv` runtime dependency.
  - Pin the GitHub Actions Python version and ensure changes under `scripts/**` trigger the workflow.
  - Pass the exact failed Pages contract, the aggregate release workflow check, and final `make ci`.

  ### Resolution
  Replaced the dependency-free release helper's `uv run --script` boundary with direct Python 3 execution and removed every obsolete `UV_CACHE_DIR` assignment from lifecycle callers and contract fixtures. GitHub Actions now provisions Python 3.11 explicitly before `make ci`, and both push and pull-request filters include `scripts/**` so release-tool-only changes cannot bypass CI. The release tooling test and static workflow validator reject a restored `uv` boundary or missing Python/path-filter contract.

  ### Validation
  `make release-pages-contract-check` passed with a deliberately failing `uv` executable first in `PATH`. `make release-workflow-check`, Python/shell/Node syntax checks, workflow YAML parsing, `git diff --check`, and final `make ci` passed; the final integration run completed all 454 Playwright/API tests.

  ### Changed Files
  `PLAN.md`, `.github/workflows/ci.yml`, `.mprlab/ISSUES.md`, `scripts/publish-mobile.sh`, `scripts/publish-react-native.sh`, `scripts/release-preflight.sh`, `scripts/release/prepare_release.sh`, `scripts/release/publish_container_artifacts.sh`, `scripts/release/publish_release.sh`, `scripts/release/release_helper.py`, `scripts/test-release-tooling.sh`, `scripts/test-staged-release-artifacts.sh`, and `scripts/validate-release-workflow.mjs`.

- [x] [B044] (P1) Make the lifecycle Bash-function rejection probe portable to Linux CI.
  ### Summary
  PR #277's `lifecycle-orchestration-contract-check` passes on macOS but fails silently on Ubuntu when it expects `/bin/sh` to preserve an injected `BASH_FUNC_git%%` environment entry. Ubuntu's `dash` removes that non-POSIX key before `run_lifecycle.sh` can inspect it; the injected function does not execute, but the test incorrectly requires the runner's rejection message.
  ### Deliverables
  - Deliver the hostile exported-function key to `run_lifecycle.sh` through a shell boundary that preserves it on both macOS and Ubuntu.
  - Keep the production `/bin/sh` launcher probe and the exact rejection/no-execution assertions.
  - Pass the focused lifecycle contract in Ubuntu and locally, the aggregate release workflow check, and final `make ci`.
  ### Resolution
  Kept the production `/bin/sh` execution probe intact and routed only the hostile exported-function rejection probe through Bash with startup files disabled. The injected `BASH_FUNC_git%%` key now reaches `run_lifecycle.sh` consistently on macOS and Ubuntu, so the test still requires the exact fail-closed diagnostic and proves the injected `git` function never executes instead of depending on whether the platform's `/bin/sh` silently removes the key first.
  ### Validation
  `make lifecycle-orchestration-contract-check` passed locally and the complete contract script passed in a clean Ubuntu 24.04 container. `make release-workflow-check`, `bash -n scripts/test-lifecycle-orchestration.sh`, `git diff --check`, and final `make ci` passed; the final integration run completed all 454 Playwright/API specs.
  ### Changed Files
  `PLAN.md`, `.mprlab/ISSUES.md`, and `scripts/test-lifecycle-orchestration.sh`.

- [x] [B045] (P0) Make remote release-tag verification portable to the production Mac.
  Goal:
  Prevent `make release` from failing before preparation when macOS BSD `awk` parses the remote annotated-tag resolver.

  Requirements:
  Keep remote default-branch and stable-tag identity verification fail-closed. Do not retry release or publication while repairing the non-mutating preflight path.

  Deliverables:
  - Replace the parser-ambiguous `awk` ternary with the repository's canonical explicit peeled-tag selection.
  - Exercise a pushed annotated stable tag through the real repository identity boundary.
  - Run the non-mutating release workflow contract on macOS CI with canonical Bash and core utilities so production-Mac portability cannot remain Ubuntu-only.
  - Keep the workflow contract statically enforced by the repository validator.

  Resolution:
  Replaced the parser-ambiguous remote annotated-tag ternary with the repository's explicit peeled-tag `if/else` selection. The lifecycle fixture now pushes an annotated stable tag before repository identity validation, so macOS BSD `awk` executes the previously skipped path. GitHub Actions now has a separate non-mutating macOS 15 release-contract job with pinned Node and Python plus canonical Homebrew Bash/coreutils, and the repository validator scopes and enforces that complete job contract.

  Validation:
  The original expression reproduced the reported BSD `awk` syntax error while the prior lifecycle target and baseline `make ci` passed, proving the false-negative. After the fix, `make lifecycle-orchestration-contract-check` passed on macOS, the same script passed in a clean Ubuntu 24.04 container, workflow YAML parsing and shell syntax checks passed, `make release-workflow-check` passed all six focused suites plus static validation, and final `make ci` passed with all 454 Playwright/API specs. No release, publication, or deployment target was invoked.

  Changed Files:
  `PLAN.md`, `.github/workflows/ci.yml`, `.mprlab/ISSUES.md`, `scripts/release/repository_identity.sh`, `scripts/test-lifecycle-orchestration.sh`, and `scripts/validate-release-workflow.mjs`.

- [x] [B046] (P0) Surface iOS artifact preparation failures at the failing builder.
  Goal:
  Prevent `make release` and `make release-dry-run` from masking a failed iOS archive step and reporting only a missing-payload inventory error after later artifact builders succeed.

  Requirements:
  Keep the exact canonical nine-file payload gate. Make CocoaPods preparation deterministic and noninteractive, and stop artifact preparation at the first failed builder. Do not allow npx to download undeclared tooling or add a yes/fallback path.

  Deliverables:
  - Lock `pod-install@1.1.0` in the mobile package contract.
  - Include locked development tooling when the production-mode archive copy runs `npm ci`.
  - Keep Expo prebuild and CocoaPods installation on local `npx --no-install` executables.
  - Make `mobile-release-artifacts` stop immediately on an iOS failure without invoking Android.
  - Extend mobile and release workflow validation so the missing-package contract cannot return.

  Resolution:
  Locked `pod-install@1.1.0` in the mobile package and lockfile contract, changed the disposable production-mode archive install to `npm ci --include=dev`, and kept Expo prebuild and CocoaPods installation on local `npx --no-install` executables with closed stdin. Made the mobile release-artifact recipe fail immediately when the iOS builder fails, before Android can run, and added package, lockfile, subprocess, and fail-fast regression checks to the mobile and release validators.

  Validation:
  A clean `NODE_ENV=production npm ci --include=dev` install exposed the locked local `pod-install@1.1.0` executable. `make build-ios` completed Expo prebuild, CocoaPods installation, signed Xcode archive, App Store export, and IPA validation without publishing. `make mobile-check`, `make staged-release-contract-check`, and `make release-workflow-check` passed. Final `make ci` passed with all 454 Playwright/API integration specs.

  Changed Files:
  `PLAN.md`, `.mprlab/ISSUES.md`, `Makefile`, `mobile/package.json`, `mobile/package-lock.json`, `mobile/scripts/build-ios-archive.mjs`, `mobile/scripts/validate-mobile-config.mjs`, `scripts/test-staged-release-artifacts.sh`, and `scripts/validate-release-workflow.mjs`.

- [x] [B047] (P0) Keep container archive image identity aligned with the staged descriptor.
  Goal:
  Prevent `make release` from failing staged artifact verification when Docker reports an inspect image ID that differs from the saved archive config digest.

  Requirements:
  Keep one canonical container identity contract: the prepared descriptor, staged verifier, publish preflight, and registry checks must use the saved archive config digest. Do not add Docker-version fallbacks or compatibility aliases.

  Deliverables:
  - Write the staged container descriptor image ID from the saved archive config digest.
  - Verify loaded publish images through the saved image archive identity rather than Docker's inspect-only ID.
  - Add release fixture coverage for Docker inspect IDs that differ from saved archive config digests.
  - Verify the focused release workflow gate and full CI.

  Resolution:
  Added a repository release helper that reads the canonical image identity from a Docker archive's config JSON digest, then changed container artifact preparation to write that saved-archive identity into `container.json`. Publish preflight and publish now verify loaded local images by saving the loaded tag back to an archive and comparing the same config digest, while retaining platform checks through Docker inspect.

  Validation:
  Reproduced the production-machine mismatch from the failed staged artifact: Docker inspect reported `sha256:8954...` while the saved archive config digest was `sha256:d220...`. Added fake-Docker release fixtures for both preparation and publish where inspect IDs intentionally differ from archive config digests. `make staged-release-contract-check`, `make publish-preflight-contract-check`, `make release-workflow-check`, and final `make ci` passed with all 454 Playwright/API integration specs.

  Changed Files:
  `PLAN.md`, `.mprlab/ISSUES.md`, `scripts/release/container_archive_image_id.py`, `scripts/release/prepare_container_artifact.sh`, `scripts/release/publish_container_artifacts.sh`, `scripts/test-publish-preflight.sh`, `scripts/test-release-tooling.sh`, `scripts/test-staged-release-artifacts.sh`, and `scripts/validate-release-workflow.mjs`.

- [x] [B048] (P0) Allow attestation sidecars on the mutable container `latest` tag.
  Goal:
  Prevent `make publish` from failing preflight when the existing mutable `ghcr.io/tyemirov/loopaware:latest` index contains one deployable `linux/amd64` manifest plus Docker's `unknown/unknown` attestation sidecar.

  Requirements:
  Keep immutable version and versioned platform tags strict. Only relax the mutable `latest` preflight enough to accept attestations that point at the single deployable `linux/amd64` digest; do not accept multiple deployable platforms as the canonical publish state.

  Deliverables:
  - Treat Docker attestation manifests on `latest` as sidecars rather than deployable platform entries.
  - Preserve the single `linux/amd64` deployable image invariant for existing mutable `latest`.
  - Add publish preflight fixture coverage for the real registry shape.
  - Verify focused release workflow checks and full CI.

  Resolution:
  Changed the mutable `latest` preflight parser so `unknown/unknown` Docker attestation manifests are accepted only when they declare `vnd.docker.reference.type=attestation-manifest` and point at the single deployable `linux/amd64` digest. Existing immutable version and versioned platform tag checks still require exact prepared image identity, and an extra deployable platform on `latest` still fails.

  Validation:
  Verified the live `ghcr.io/tyemirov/loopaware:latest` index contains one `linux/amd64` manifest plus one `unknown/unknown` attestation manifest. Added publish-preflight fixture coverage for that shape and retained the failure case for a real extra `linux/arm64` deployable manifest. `make publish-preflight-contract-check`, `make release-workflow-check`, and final `make ci` passed with all 454 Playwright/API integration specs.

  Changed Files:
  `PLAN.md`, `.mprlab/ISSUES.md`, `scripts/release/publish_container_artifacts.sh`, `scripts/test-publish-preflight.sh`, and `scripts/validate-release-workflow.mjs`.

- [x] [B049] (P0) Make seeded dashboard authentication atomic with header readiness.
  Goal:
  Prevent the black-box integration suite from intermittently failing with `loopaware.header_missing` when the dashboard redirects between the helper's header-readiness check and its seeded MPR UI authentication call.

  Requirements:
  Keep authentication at the browser boundary and preserve real navigation behavior. Perform readiness detection and seeded authentication in one browser evaluation so navigation cannot remove the header between those operations. Do not increase timeouts or add blind delays.

  Deliverables:
  - Replace the split readiness/evaluation sequence with one observable browser readiness operation that authenticates the current header.
  - Remove retry constants and delay logic made obsolete by the atomic operation.
  - Exercise the change through the black-box dashboard integration suite and final CI.

  Validation:
  Reproduce the baseline `loopaware.header_missing` failure, then run the dashboard integration surface and final `make ci`.

  Resolution:
  Replaced the split header-readiness wait and seeded authentication evaluation with one browser-side readiness operation that authenticates the currently bound MPR UI header. Removed the obsolete retry count, delay, and navigation-error string matching so dashboard redirects remain real while the test boundary no longer observes a stale header reference.

  Validation Results:
  Baseline `make ci` and a canonical integration rerun reproduced `loopaware.header_missing` in two different dashboard tests. After the fix, `make lint` passed and `make test-integration-all` passed all 454 Playwright/API integration specs.

  Changed Files:
  `PLAN.md`, `.mprlab/ISSUES.md`, and `tests/helpers/fixtures.js`.

- [x] [B050] (P0) Make remote release refs authoritative and transactional.
  Goal:
  Prevent a local-only or replayed release tag from blocking preparation, and prevent publication from advancing `origin/master` without publishing the matching release tag.

  Requirements:
  Treat remote stable tags as the durable release source of truth. Local tags and `.git/mprlab-release` state are transient preparation state. Preserve only the one exact pending local release tag directly above the current remote default branch; discard other unpublished local stable tags and force local stable tags to the remote targets. Publish the release commit and tag in one atomic Git transaction. Do not add compatibility reads or preserve the aborted untagged release shape.

  Deliverables:
  - Synchronize stable local tags from `origin` before release-state validation.
  - Delete stale or unpublished local stable tags except the exact pending prepared release at `HEAD`.
  - Push the default branch and release tag through one `git push --atomic` operation.
  - Add contract coverage proving a tag rejection cannot advance the remote default branch.
  - Remove the aborted untagged `v0.7.44` CHANGELOG section as a bounded forward migration.
  - Document the remote-authoritative transactional contract.

  Validation:
  Run lifecycle orchestration, release tooling, static release workflow validation, and final `make ci`.

  Resolution:
  Release-state validation now reads the remote default branch and stable tags first, deletes unpublished local stable tags except the one exact pending prepared release directly above the remote default branch, and force-synchronizes all remote stable tags in one fetch. Prepared local tags and `.git/mprlab-release` remain transient inputs. Publication now sends every required default-branch and tag refspec through one `git push --atomic`, so a rejected tag cannot leave the remote branch advanced. Removed the aborted untagged `v0.7.44` CHANGELOG section so the next release rebuilds that version from the current remote source.

  Validation Results:
  Added lifecycle fixtures for missing, stale, wrong-target, and exact-pending local tags. Added a bare-repository pre-receive rejection that proves the default branch remains unchanged when the tag is rejected, plus a success fixture requiring both refs in one receive transaction. `make lifecycle-orchestration-contract-check`, `make release-pages-contract-check`, and `make release-workflow-check` passed. After restarting Docker Desktop to recover its containerd metadata store, final `make ci` passed all release contracts, Go race checks, and 454 Playwright/API integration specs.

  Changed Files:
  `PLAN.md`, `.mprlab/ISSUES.md`, `CHANGELOG.md`, `README.md`, `scripts/release/prepare_release.sh`, `scripts/release/release_helper.py`, `scripts/release/repository_identity.sh`, `scripts/test-lifecycle-orchestration.sh`, `scripts/test-release-tooling.sh`, and `scripts/validate-release-workflow.mjs`.

- [x] [B051] (P0) Fail closed while building the React Native release artifact.
  Goal:
  Prevent the exact-commit React Native artifact builder from reporting success and staging a package without compiled outputs after its clean dependency install fails.

  Requirements:
  Run package commands from inside the disposable exact-commit checkout and stop at the first failed command. Keep the canonical scoped package identity and lockfile; do not add aliases, fallbacks, or compatibility handling for temporary directory names.

  Deliverables:
  - Run the clean install, typecheck, build, verification, and final pack from the disposable package checkout.
  - Make the artifact recipe fail immediately when any package command fails.
  - Add black-box contract coverage proving a failed clean install cannot invoke later package stages or create a tarball.
  - Strengthen static release validation for the fail-fast working-directory contract.

  Validation:
  Reproduce the original false-success boundary, run the real exact-commit package artifact target, focused staged/release workflow checks, and final `make ci`.

  Resolution:
  Made the exact-commit React Native artifact recipe fail immediately and run its clean install, typecheck, build, verification, and pack commands from inside the disposable package checkout. The static release validator now enforces that fail-fast working-directory contract and rejects the broken temporary-directory `npm --prefix` form. Added a black-box staged-artifact fixture that fails the disposable clean install and proves no later npm command runs and no package is created.

  Validation Results:
  Reproduced the original target returning success while staging a three-file package without `dist/`. The new regression failed against that behavior and passed after the repair. A real exact-commit artifact build produced the canonical five-file package with `dist/index.js` and `dist/index.d.ts`. `make staged-release-contract-check`, `make release-workflow-check`, `git diff --check`, and final `make ci` passed; the final integration run completed all 454 Playwright/API specs.

  Changed Files:
  `PLAN.md`, `.mprlab/ISSUES.md`, `Makefile`, `scripts/test-staged-release-artifacts.sh`, and `scripts/validate-release-workflow.mjs`.

- [x] [B052] (P0) Follow the GHCR Bearer challenge during push-authority preflight.
  Goal:
  Let `make publish` prove GHCR write authority with valid GitHub Package credentials instead of failing when the registry correctly returns its initial authentication challenge.

  Requirements:
  Require the canonical GHCR Registry v2 Bearer challenge, exchange the configured GitHub credentials for the exact repository `pull,push` scope, and use only that scoped Bearer token for the temporary upload session and cleanup. Reject unexpected realms, services, scopes, redirects, or direct Basic upload authorization; do not add alternate authentication paths.

  Deliverables:
  - Parse and validate the GHCR `WWW-Authenticate` challenge returned by the upload endpoint.
  - Exchange the GitHub username and token for a repository-scoped registry Bearer token.
  - Create and delete the preflight upload session with Bearer authorization while keeping credentials out of command arguments and logs.
  - Add black-box and static contract coverage for the challenge, token exchange, authenticated upload creation, cleanup, and rejection boundaries.

  Validation:
  Reproduce the original HTTP 401 boundary, run the focused publish-preflight and release workflow checks, and run final `make ci`.

  Resolution:
  Changed the GHCR write-authority preflight to start without credentials, require the registry's canonical Bearer challenge, and validate its trusted realm, service, and repository `pull` scope. The preflight now exchanges GitHub credentials for the explicit repository `pull,push` scope, uses only the returned Bearer token to create and delete the temporary upload session, and rejects upload locations outside the exact GHCR repository path. GitHub credentials and the scoped registry token stay in curl configuration input instead of command arguments or logs.

  Validation Results:
  Reproduced GHCR's live HTTP 401 challenge and confirmed it advertises `https://ghcr.io/token`, service `ghcr.io`, and repository `pull` scope. Added a four-request black-box contract for the unauthenticated challenge, authenticated token exchange, Bearer upload creation, and Bearer cleanup, plus rejection coverage for an untrusted realm and off-registry upload location. `make publish-preflight-contract-check`, `make release-workflow-check`, `git diff --check`, and final `make ci` passed; the final integration run completed all 454 Playwright/API specs. An earlier final-CI attempt was invalidated by an aborted Docker Desktop VM filesystem journal; after restarting Docker Desktop and removing that failed run's isolated stack, the clean rerun passed.

  Changed Files:
  `PLAN.md`, `.mprlab/ISSUES.md`, `scripts/release/publish_container_artifacts.sh`, `scripts/test-publish-preflight.sh`, and `scripts/validate-release-workflow.mjs`.

- [x] [B053] (P0) Rebuild the release-preflight fixes on clean history and parse GHCR challenges semantically.
  Goal:
  Keep the aborted local `v0.7.44` release commit out of the proposed change while accepting valid GHCR Bearer challenges regardless of authentication-parameter order or optional whitespace.

  Requirements:
  Build the replacement branch directly from `origin/master` without rewriting history. Parse exactly one Bearer challenge into the canonical `realm`, `service`, and `scope` parameters, reject missing, duplicate, or unknown parameters, and keep exact value validation. Do not preserve the positional parser or the aborted changelog entry.

  Deliverables:
  - Reapply the B051 and B052 implementation without the aborted release commit or `v0.7.44` changelog section.
  - Parse Bearer authentication parameters independently of their order and optional whitespace.
  - Reject malformed challenges and challenges with missing, duplicate, or unknown parameters.
  - Add black-box coverage for reordered valid parameters and duplicate-parameter rejection.
  - Update static release validation for the semantic parsing contract.

  Validation:
  Run the focused staged-artifact, publish-preflight, and release-workflow contracts, verify the branch ancestry/diff excludes `CHANGELOG.md`, and run final `make ci`.

  Resolution:
  Rebuilt the release-preflight changes on `bugfix/B053-rebuild-release-preflight` directly from `origin/master`, leaving the aborted local `v0.7.44` release commit and changelog section outside the proposed history. Reapplied the fail-fast exact-commit React Native package build and the GHCR Bearer token exchange. Replaced the positional GHCR challenge regex with strict authentication-parameter parsing that accepts canonical parameters in any order with optional whitespace, requires exactly `realm`, `service`, and `scope`, rejects missing, duplicate, unknown, or malformed parameters, and preserves exact trusted-value validation.

  Validation Results:
  Baseline `make ci` passed before implementation. `make staged-release-contract-check publish-preflight-contract-check release-workflow-check` passed with black-box coverage for canonical and reordered challenges plus untrusted, missing, duplicate, and unknown parameter rejection. The ancestry audit reported `aborted-release-ancestor=no`, the diff audit reported `changelog-diff=no`, and `git diff --check` passed. Final `make ci` passed all release contracts, Go race checks, and 454 Playwright/API integration specs.

  Changed Files:
  `PLAN.md`, `.mprlab/ISSUES.md`, `Makefile`, `scripts/release/publish_container_artifacts.sh`, `scripts/test-publish-preflight.sh`, `scripts/test-staged-release-artifacts.sh`, and `scripts/validate-release-workflow.mjs`.

- [x] [B054] (P0) Accept GHCR's current repository-scoped upload session location.
  Goal:
  Let `make publish` complete its non-pushing GHCR authority check when the registry returns its current singular `blobs/upload/<session>` location after creating the temporary upload session.

  Requirements:
  Treat the returned location as a security boundary: require HTTPS on `ghcr.io`, the exact target repository, the current singular upload namespace, and one opaque session path segment. Reject cross-origin, wrong-repository, malformed, query-bearing, or obsolete plural locations. Keep the upload-creation request on the Registry v2 `blobs/uploads/` endpoint and use the validated returned location verbatim for cleanup.

  Deliverables:
  - Parse and validate the upload-session location semantically instead of comparing it with the obsolete plural response prefix.
  - Add black-box coverage for GHCR's live absolute location shape and rejection of untrusted and obsolete locations.
  - Update the static release-workflow contract for the singular returned-location namespace.

  Validation:
  Reproduce the reported absolute GHCR location in the publish-preflight fixture, run the focused publish-preflight and release-workflow checks, and run final `make ci`.

  Resolution:
  Replaced the obsolete plural-prefix comparison with strict semantic validation of GHCR's current absolute upload-session location. The preflight now requires HTTPS on the exact `ghcr.io` authority, the target repository's singular `blobs/upload/` namespace, and one safe opaque session segment, then sends the cleanup DELETE to that validated URL. Relative, cross-origin, wrong-repository, query-bearing, and obsolete plural response locations are rejected; the Registry v2 upload-creation request remains on `blobs/uploads/`.

  Validation Results:
  Reproduced the reported `https://ghcr.io/v2/tyemirov/loopaware/blobs/upload/16.<uuid>` response in the black-box fixture and verified exact-URL cleanup plus rejection of untrusted-origin, wrong-repository, obsolete-plural, and query-bearing locations. `bash -n`, `git diff --check`, `node scripts/validate-release-workflow.mjs`, and `make publish-preflight-contract-check release-workflow-check` passed. The initial clean baseline completed 454 specs and hit a transient browser-context teardown timeout on the final spec; final `make ci` passed all release contracts, Go race checks, and all 455 Playwright/API integration specs.

  Changed Files:
  `PLAN.md`, `.mprlab/ISSUES.md`, `scripts/release/publish_container_artifacts.sh`, `scripts/test-publish-preflight.sh`, and `scripts/validate-release-workflow.mjs`.

- [x] [B055] (P0) Recover release preparation from stale unpublished changelog sections.
  Goal:
  Let `make release` reuse the next remote-authoritative version when an earlier failed publication left release commits and duplicate changelog sections on the default branch without publishing the corresponding remote tag.

  Requirements:
  Keep remote stable tags authoritative for version selection. For the one selected unpublished version, require generated notes to identify that exact version, delete every stale changelog section for it, and insert one regenerated canonical section. Do not preserve conflicting sections, bump around unpublished state, or add alternate compatibility paths. Exclude release-bookkeeping commits from generated release notes.

  Deliverables:
  - Require `insert-changelog` to receive and validate the selected version.
  - Replace all stale sections for that exact version while preserving other release sections.
  - Omit `Release vMAJOR.MINOR.PATCH` bookkeeping subjects from generated notes.
  - Add black-box coverage reproducing multiple stale headings with different dates and content.
  - Update the static release-workflow contract for the canonical recovery path.

  Validation:
  Reproduce the duplicate unpublished-version headings in a repository fixture, run the focused release workflow checks, and run final `make ci`.

  Resolution:
  Kept remote stable tags authoritative, so the unpublished `v0.7.44` version remains the next release instead of being skipped. The release workflow now passes that exact selected version into changelog insertion, requires the generated notes heading to match it, removes every existing canonical section for that unpublished version, and inserts one regenerated section while preserving `Unreleased` and all other releases. Release-bookkeeping commit subjects are excluded from generated notes.

  Validation Results:
  Baseline `make ci` passed all 455 Playwright/API integration specs. The black-box release fixture reproduced two stale `v1.2.3` headings with different dates and content, replaced them with one canonical section, preserved `Unreleased` and `v1.2.2`, omitted the stale text and `Release v1.2.3` commit subject, and rejected a mismatched requested version. Applying the helper to a disposable clone of the actual repository replaced both stale `v0.7.44` sections with one and produced zero release-bookkeeping bullets. `make release-workflow-check`, `git diff --check`, and final `make ci` passed; final CI included all release contracts, Go race checks, and all 455 integration specs.

  Changed Files:
  `PLAN.md`, `.mprlab/ISSUES.md`, `scripts/release/prepare_release.sh`, `scripts/release/release_helper.py`, `scripts/test-release-tooling.sh`, and `scripts/validate-release-workflow.mjs`.

- [x] [B056] (P0) Use the standard Docker flow for GHCR publication.
  Goal:
  Publish LoopAware containers through GHCR's supported Docker client path without maintaining a parallel Registry v2 implementation.

  Requirements:
  Authenticate with `docker login ghcr.io --username ... --password-stdin` during container publication preflight and publication. Let the existing `docker push` perform the real push-authority check during publication. Delete the manual Bearer challenge, token exchange, upload-session creation, upload-location parsing, and cleanup logic without preserving a compatibility path.

  Deliverables:
  - Remove all direct GHCR Registry API calls from container publication.
  - Use one standard Docker login path for preflight and publication.
  - Keep preflight non-publishing while validating prepared archive loadability and Docker authentication.
  - Cover successful login, rejected login, non-pushing preflight, and normal Docker push in the black-box publication fixture.
  - Update the static release-workflow contract to reject manual Registry API logic.

  Validation:
  Run the focused publish-preflight and release-workflow checks, verify the obsolete Registry API strings are absent, and run final `make ci`.

  Resolution:
  Deleted the custom GHCR Registry v2 implementation in full. Container preflight and publication now share one canonical `docker login ghcr.io --username ... --password-stdin` path; preflight stops after artifact, remote-state, and authentication checks, while publication proves push authority through the existing `docker push`. The fixture no longer simulates Bearer challenges, token exchange, upload sessions, response locations, or cleanup.

  Validation Results:
  Baseline `make ci` passed all release contracts, Go race checks, and 455 Playwright/API integration specs. The focused black-box fixture passed successful Docker login, rejected Docker login, non-pushing preflight, prepared archive loadability, and normal Docker push coverage. `bash -n`, `git diff --check`, `node scripts/validate-release-workflow.mjs`, and `make publish-preflight-contract-check release-workflow-check` passed. Final `make ci` passed all release contracts, Go race checks, and all 455 integration specs. No real registry login or publication was run.

  Changed Files:
  `PLAN.md`, `.mprlab/ISSUES.md`, `scripts/release/publish_container_artifacts.sh`, `scripts/test-publish-preflight.sh`, and `scripts/validate-release-workflow.mjs`.

- [x] [B057] (P0) Validate the exact iOS artifact with App Store Connect API-key authentication.
  Goal:
  Restore `make publish` by removing the unsupported App Store Connect provider-list operation from the iOS publication preflight.

  Requirements:
  Use one canonical API-key-authenticated `altool --validate-app` operation against the prepared IPA before upload. Delete the partial credential-only preflight and its `--preflight-only` mode instead of preserving an alias, fallback, or compatibility path.

  Deliverables:
  - Make mobile publication preflight validate the exact prepared IPA without calling `altool --list-providers`.
  - Make the lower-level iOS submission target validate the exact artifact before uploading it.
  - Add black-box and static release-contract coverage that rejects provider listing and requires exact IPA validation.
  - Document the canonical App Store Connect preflight boundary.

  Validation:
  Run the focused iOS/npm publication contract, mobile config checks, aggregate release workflow checks, a live non-uploading App Store Connect validation against the prepared IPA, and final `make ci`.

  Resolution:
  Removed the credential-only iOS preflight and its `altool --list-providers` call. Mobile publication and the lower-level `make submit-ios` path now use the existing hash-verified exact-IPA `altool --validate-app` operation before upload. Updated the black-box fixtures and static validators to require that canonical operation and reject restoration of provider listing or the partial iOS `--preflight-only` mode.

  Validation Results:
  Focused iOS/npm publication, mobile, publish-preflight, and aggregate release-workflow checks passed; final `make ci` passed all release contracts, Go race checks, and 455 Playwright/API integration specs. PR #289 passed both required GitHub checks and merged to `master`. A clean `make release && make publish` run prepared the signed `v0.7.44` IPA, and the live API-key-authenticated `altool --validate-app` operation passed with verified App Store Connect authority. Publication then stopped at the subsequent npm authority preflight recorded as B058; no store artifact was uploaded.

  Changed Files:
  `PLAN.md`, `.mprlab/ISSUES.md`, `Makefile`, `README.md`, `mobile/scripts/submit-ios.mjs`, `mobile/scripts/validate-mobile-config.mjs`, `scripts/publish-mobile.sh`, `scripts/test-ios-npm-publication.sh`, `scripts/test-publish-preflight.sh`, and `scripts/validate-release-workflow.mjs`.

- [x] [B058] (P0) Prove npm publication authority at the prepared package boundary.
  Goal:
  Restore `make publish` after iOS validation by replacing the broad npm user-package listing with a package-scoped write-authority check for `@loopaware/react-native`.

  Requirements:
  Use the existing idempotent `npm access set status=public @loopaware/react-native` operation as the canonical package-scoped authority proof. Delete the `npm whoami` plus `npm access list packages <user>` path instead of preserving a fallback, alias, or second authority check.

  Deliverables:
  - Remove the broad npm identity package-list query from publication preflight.
  - Keep the existing package existence, visibility, exact integrity, monotonic version, dry-run, and post-publication checks.
  - Add black-box and static contract coverage that rejects restoration of user package listing and proves write denial fails before publication.
  - Document the package-scoped npm authority boundary.

  Validation:
  Run the focused iOS/npm publication and release-workflow checks, verify the live package-scoped authority command with the configured token, run final `make ci`, and successfully run `make release && make publish` from clean canonical `master`.

  Resolution:
  Removed the broad `npm whoami` plus `npm access list packages <user>` sequence. The existing idempotent package-scoped public-status write is now the single authority proof, while the existing package existence, visibility, exact integrity, monotonic version, dry-run, and post-publication checks remain in place. Updated the black-box fixture to fail before publication when that write is denied, and updated the static contract to reject restoration of either broad identity operation.

  Validation Results:
  The live package-scoped `npm access set status=public @loopaware/react-native` command succeeded with the configured granular token and returned the canonical public package status. Focused iOS/npm publication, publish-preflight, and aggregate release-workflow checks passed. Final `make ci` passed all release contracts, Go race checks, and all 455 Playwright/API integration specs. PR #290 passed both required GitHub checks and merged to `master`. A clean `make release && make publish` run passed the live npm authority preflight; the existing `@loopaware/react-native@0.1.0` registry integrity matched the prepared tarball, the package was public, and `latest` already pointed to `0.1.0`. Publication then stopped at the subsequent GHCR manifest-shape verifier recorded as B059; no npm or mobile store upload was required or performed.

  Changed Files:
  `PLAN.md`, `.mprlab/ISSUES.md`, `README.md`, `scripts/publish-react-native.sh`, `scripts/test-ios-npm-publication.sh`, and `scripts/validate-release-workflow.mjs`.

- [x] [B059] (P0) Push the canonical single-platform manifest to GHCR.
  Goal:
  Restore container publication by ensuring the versioned `linux/amd64` tag contains the deployable image manifest that the immutable registry verifier expects, not BuildKit's enclosing OCI index and attestation sidecar.

  Requirements:
  Use the standard Docker platform-specific push operation for the prepared `linux/amd64` image. Keep one strict registry shape and delete any need to accept an OCI index at the versioned platform tag; do not add a fallback or dual-shape verifier.

  Deliverables:
  - Push the versioned platform tag with explicit `--platform linux/amd64` selection.
  - Preserve exact prepared config-digest verification before and after the push.
  - Add black-box and static release-contract coverage that requires the platform-specific Docker push.
  - Document why the platform tag excludes the enclosing BuildKit index and attestations.

  Validation:
  Run focused publish-preflight and release-workflow checks, run final `make ci`, inspect the live GHCR platform tag shape after publication, and successfully run `make release && make publish` with a fresh release identity.

  Resolution:
  Replaced the ordinary versioned platform push with `docker push --platform linux/amd64`, so GHCR receives the prepared deployable image manifest instead of BuildKit's enclosing OCI index and attestation sidecar. Kept the existing strict single-manifest config-digest verifier as the only accepted platform-tag contract. The black-box fixture rejects a push without the exact platform selection, and the static workflow validator requires it.

  Validation Results:
  Live `v0.7.44` publication reproduced the noncanonical index failure before the fix. Focused publish-preflight and aggregate release-workflow checks passed, followed by `make ci` with all release contracts, Go race checks, and 455 Playwright/API integration specs. PR #291 passed both required GitHub checks and merged to `master`. Live `v0.7.45` proved the corrected platform push and index creation. The final `make release && make publish` command completed successfully for `v0.7.46`; live GHCR inspection confirmed `v0.7.46-linux-amd64` is one OCI image manifest with prepared config `sha256:e54ec04cbe53414365cd0ba8d4be62fc1048a7073516ad7f4da9a40405a53b0c`, while `v0.7.46` and `latest` each contain exactly one `linux/amd64` entry at deployable digest `sha256:8ea85bdda40ef7160aff6e1c307ec8e510d3e8df9f423cd06219187f48e69bfe`.

  Changed Files:
  `PLAN.md`, `.mprlab/ISSUES.md`, `README.md`, `scripts/release/publish_container_artifacts.sh`, `scripts/test-publish-preflight.sh`, and `scripts/validate-release-workflow.mjs`.

- [x] [B060] (P0) Replace the Google Play internal track with one completed release.
  Goal:
  Complete mobile publication by sending the canonical Google Play track shape instead of appending a second completed release that the Android Publisher API rejects.

  Requirements:
  Publish one exact completed internal-track release containing the new prepared version code. Reject active or manual rollout states before upload, but do not retain obsolete completed release objects, metadata, or version codes in the replacement track.

  Deliverables:
  - Replace the existing completed release with the exact new completed release in the Play track update.
  - Post-verify that the committed track contains exactly that one release and version code.
  - Add black-box and static release-contract coverage that rejects completed-release append behavior.
  - Correct the release runbook and prior lifecycle contract note to describe replacement semantics.

  Validation:
  Run the focused publish-preflight and mobile configuration checks, run final `make ci`, inspect the live Play internal track, and successfully run `make release && make publish` with a fresh release identity because iOS build `206389225` was already accepted before the `v0.7.45` Android failure.

  Resolution:
  Replaced the internal-track update payload with one canonical completed release containing only the prepared version code. Existing completed release objects are validated but not retained; active or manual rollouts still fail before upload. Post-publication verification now requires the committed track to contain exactly the new completed release. Black-box coverage proves the outgoing payload omits the old release and rejects a provider response that retains it, while the static workflow validator rejects restoration of append behavior.

  Validation Results:
  Live `v0.7.45` publication reproduced `Only one completed release is allowed` after App Store Connect accepted build `206389225`; a transient inspection confirmed Google Play remained unchanged at version code `205912845`. Focused publish-preflight, mobile config, and aggregate release-workflow checks passed, followed by `make ci` with all release contracts, Go race checks, and all 455 Playwright/API integration specs. PR #292 passed the macOS release-contract and full test checks and merged to `master`. The exact `make release && make publish` command then completed successfully for `v0.7.46`: App Store Connect accepted fresh build `206391477` with delivery UUID `d3eba9e4-842b-48e0-b01d-7e9a4f2147b4`, Google Play committed edit `06256342907665238563`, and a new transient inspection confirmed the internal track contains exactly one completed `2026.7.16` release at version code `206391477` with AAB hash `9d5bede44a267eeba0c6f8afc2d245bb58b5a474662c773704b3e453e710f25c`. The immutable `publication.json` attestation records every provider stage complete.

  Changed Files:
  `PLAN.md`, `.mprlab/ISSUES.md`, `README.md`, `mobile/scripts/publish-android-play.mjs`, `scripts/test-publish-preflight.sh`, and `scripts/validate-release-workflow.mjs`.

- [x] [B061] (P0) Replace the raw Ansible become prompt with the gateway credential boundary.
  Goal:
  Keep `make deploy` on the canonical app-owned deployment controller while removing the unexplained LoopAware-only `BECOME password:` prompt.

  Requirements:
  Prompt explicitly for `Gateway sudo password:` only when the operator has not supplied the canonical local password file. Pass the credential through a temporary private file, remove it after Ansible exits, and never restore `--ask-become-pass`.

  Deliverables:
  - Replace Ansible's raw interactive become flag with the same explicit gateway sudo-password boundary used by the aggregate controller.
  - Preserve the non-interactive `LOOPAWARE_ANSIBLE_BECOME_PASSWORD_FILE` contract.
  - Add black-box coverage for prompt text, password-file handoff, cleanup, and the absence of `--ask-become-pass`.
  - Add a static release-contract rejection for the raw prompt path and document the operator-facing credential name.

  Validation:
  Run the focused deploy dry-run contract, aggregate release-workflow check, shell syntax and diff checks, then final `make ci`. Do not run production deployment.

  Resolution:
  Removed LoopAware's explicit `--ask-become-pass` handoff, which exposed Ansible's unexplained `BECOME password:` text. The app-owned controller now reads an interactive credential through the explicit `Gateway sudo password:` boundary, passes it through a private temporary password file, deletes that file on exit, and preserves the canonical non-interactive password-file input. Black-box coverage proves the exact prompt, password-file handoff and cleanup, while the static workflow contract rejects restoration of the raw Ansible prompt path.

  Validation Results:
  `bash -n scripts/run-app-ansible-deploy.sh scripts/test-deploy-dry-run.sh`, `git diff --check`, `make deploy-dry-run-contract-check`, and `make release-workflow-check` passed. Final `make ci` passed config audit, builds, mobile and package checks, release and deployment contracts, Go tests and race tests, and all 455 Playwright/API integration specs. No production deployment or Pages activation was executed.

  Changed Files:
  `PLAN.md`, `.mprlab/ISSUES.md`, `README.md`, `scripts/run-app-ansible-deploy.sh`, `scripts/test-deploy-dry-run.sh`, and `scripts/validate-release-workflow.mjs`.

- [x] [B062] (P0) Recover an untagged release commit idempotently.
  Goal:
  Let `make release && make publish` converge when the remote default branch already contains the exact canonical release commit but the matching stable tag and prepared manifest were never created.

  Requirements:
  Treat the one exact `Release vMAJOR.MINOR.PATCH` commit as pending release state only when it directly follows the source commit, changes only `CHANGELOG.md`, and its changelog is the canonical transformation for the remotely selected next version. Rebuild artifacts from the source parent, create the missing local tag and manifest without creating a second release commit, and reject every conflicting shape. Keep remote stable tags authoritative and add no fallback or compatibility path.

  Deliverables:
  - Recognize and validate the exact untagged release-commit state before artifact preparation.
  - Preserve source-parent provenance in real and dry-run artifact builds.
  - Reuse the existing release commit and create only the missing pending tag and manifest.
  - Add black-box coverage for recovery, source provenance, and a second idempotent invocation.
  - Update the static release contract and operator documentation.

  Validation:
  Reproduce a remote default branch containing `Release v1.2.3` with only remote tag `v1.2.2`, run the real release preparation boundary twice, run focused release workflow checks, and run final `make ci`. Do not publish providers or deploy production.

  Resolution:
  Release preparation now recognizes only the exact untagged commit for the remotely selected next version: it must have one source parent, change only `CHANGELOG.md`, and equal the canonical changelog transformation generated from that parent. Recovery rebuilds all payloads from the source parent, reuses the existing release commit, and creates the missing annotated tag and manifest without a second release commit. Release dry-run carries the same source provenance, remote-tag synchronization preserves an exact pending local tag even when its commit already reached the remote default branch, and deploy now directs an untagged operator through both release and publish phases.

  Validation Results:
  The pre-change `make ci` baseline passed all lifecycle gates and 455 integration specs. The black-box release fixture reproduced remote `master` at untagged `Release v1.2.3` with remote tag `v1.2.2`, proved dry-run selected the source parent, rejected conflicting changelog state, recovered the tag and nine-payload manifest without moving `HEAD`, and made the second invocation verification-only. The lifecycle fixture proved an exact pending local tag survives when its commit is pushed separately. Shell syntax checks, `git diff --check`, static release validation, focused `make staged-release-contract-check`, `make lifecycle-orchestration-contract-check`, aggregate `make release-workflow-check`, and final `make ci` passed; final CI included Go race checks and all 455 Playwright/API integration specs. No provider publication, Pages activation, or production deployment was run.

  Changed Files:
  `PLAN.md`, `.mprlab/ISSUES.md`, `README.md`, `scripts/deploy.sh`, `scripts/release-preflight.sh`, `scripts/release/prepare_release.sh`, `scripts/release/repository_identity.sh`, `scripts/test-lifecycle-orchestration.sh`, `scripts/test-staged-release-artifacts.sh`, and `scripts/validate-release-workflow.mjs`.

- [x] [B063] (P0) Make protected dashboard authentication readiness event-driven.
  Goal:
  Prevent a cold or delayed initial `mpr-ui` load from redirecting an authenticated dashboard test to `/login` and leaving dashboard hydration waiting on elements that cannot appear there.

  Requirements:
  Drive protected-page authentication settling from the public `mpr-ui` bootstrap status transition. Remove the fixed three-second settle timer and the test harness's two-attempt login recovery. Do not increase timeouts, add blind delays, or add a fallback path.

  Deliverables:
  - Add black-box browser coverage that delays the initial `mpr-ui` bundle beyond the former settle interval and still reaches a hydrated authenticated dashboard.
  - Treat the terminal `mpr-ui` bootstrap status as the protected-page readiness signal.
  - Require seeded dashboard authentication to complete on the requested path before dashboard hydration begins.
  - Delete the obsolete login-redirect retry and explicit-logout cleanup from the shared fixture.

  Validation:
  Reproduce the cold-load boundary through the new browser scenario, run the focused browser integration gate, and run final `make ci` without changing the Playwright timeout.

  Resolution:
  Removed the fixed protected-page authentication settle timer and made the terminal public `mpr-ui` bootstrap status the redirect decision boundary. Authenticated browser fixtures now seed the canonical TAuth restore hint and require the authenticated requested route before dashboard hydration; the obsolete login retry and explicit-logout cleanup paths were deleted. Reload coverage now waits for the current dashboard bootstrap instead of starting a competing navigation.

  Validation Results:
  - Before the fix, the new cold-bootstrap regression reproduced the 90-second dashboard account hydration timeout while the other 455 browser scenarios passed.
  - `make test-integration`: 456 passed in 4.3 minutes.
  - `make ci`: passed, including 456 browser scenarios in 3.4 minutes.

  Changed Files:
  `PLAN.md`, `.mprlab/ISSUES.md`, `web/header-auth.js`, `tests/helpers/fixtures.js`, `tests/specs/header-auth-state.spec.js`, and `tests/specs/dashboard-auto-logout.spec.js`.

- [x] [B064] (P1) Restore authentication before inspecting protected-page assets.
  Goal:
  Ensure dashboard and dashboard-preview asset checks validate the requested protected page instead of racing an authentication redirect to `/login`.

  Requirements:
  Protected asset inspection must restore the seeded TAuth session and reach the authenticated requested route before asserting CDN assets. Do not add retries, blind waits, fallback inspection paths, or timeout increases.

  Deliverables:
  - Make the protected asset helper use the canonical authenticated-page restoration contract.
  - Add black-box assertions that the dashboard and each protected preview remain on their requested routes while assets are inspected.
  - Reproduce the pre-fix redirect race deterministically and prove the corrected helper cannot validate `/login` by mistake.

  Validation:
  Run the focused protected asset scenarios, the full browser integration gate, and final `make ci`.

  Resolution:
  Split seeded authentication readiness into the mpr-ui session boundary and the LoopAware dashboard binding. Protected asset inspection now always restores the seeded TAuth session, waits for authenticated mpr-ui state on the exact requested route, and only then inspects CDN assets. Removed the obsolete session-restoration opt-out.

  Validation Results:
  - The deterministic pre-fix regression failed all four protected asset cases: `/app` redirected to `/login`, while the three preview pages remained unauthenticated; the other 452 browser scenarios passed.
  - `make lint-js`: passed.
  - `make test-integration`: passed all 456 browser scenarios in 3.9 minutes.
  - `make ci`: passed all release and lifecycle contracts, Go tests with race detection, and 456 browser scenarios in 3.6 minutes; the four protected asset cases completed in 285–372 milliseconds.

  Changed Files:
  `PLAN.md`, `.mprlab/ISSUES.md`, `tests/helpers/fixtures.js`, and `tests/specs/header-auth-state.spec.js`.

  Follow-up Review (2026-07-18):
  The first resolution restored correct protected-route behavior but constructed mpr-ui's private `tauth.restore.v1:*` storage key directly. mpr-ui v3.11.1 already publishes `MPRUI.testing.authenticate(host, profile)` for browser suites that seed backend sessions; that helper drives the mounted controller, emits the normal auth lifecycle, and owns restore-hint persistence.

  Follow-up Requirements:
  - Seed authenticated browser state exclusively through `MPRUI.testing.authenticate()`.
  - Delete LoopAware's restore-key prefix, value, and local-storage construction.
  - Preserve exact requested-route and authenticated-state assertions so protected asset checks cannot inspect `/login`.

  Follow-up Resolution:
  Replaced LoopAware's private restore-key construction with the public `MPRUI.testing.authenticate()` browser fixture. The helper mounts mpr-ui on a stable `/privacy` page in the target browser context, lets mpr-ui own its normal authenticated lifecycle and restore-hint persistence, then closes that fixture page before navigating the still-cold target page. Protected checks continue to require authenticated mpr-ui state on the exact requested route. The long-idle login scenario now seeds its stale restore state through the same public fixture.

  Follow-up Validation Results:
  - Before the follow-up fix, the new lifecycle regression received authenticated event paths `['/app']` instead of an mpr-ui fixture event on `/privacy`; the other 456 browser scenarios passed.
  - `make lint-js`: passed after asserting the public `data-mpr-auth-status` host state instead of assuming an internal helper return shape.
  - `make test-integration`: passed all 457 browser scenarios in 4.0 minutes.
  - `make ci`: passed all release and lifecycle contracts, JavaScript type checks, Go tests with race detection, and all 457 browser scenarios in 4.2 minutes.
  - After merging current `master` with B065, `make ci` passed the updated release and lifecycle contracts, JavaScript type checks, Go tests with race detection, and all 457 browser scenarios in 4.3 minutes.

  Follow-up Changed Files:
  `PLAN.md`, `.mprlab/ISSUES.md`, `tests/helpers/fixtures.js`, and `tests/specs/header-auth-state.spec.js`.

- [x] [B065] (P1) Make lifecycle targets safe to inspect with Make dry-run mode.
  ### Summary
  The aggregate gateway workflow verifier uses Make's no-execute mode to inspect app-owned release, publish, and deploy recipes without activating them, but LoopAware rejects that standard planning mode before emitting its lifecycle plan.
  ### Deliverables
  - Accept Make's `-n`/`--dry-run` planning mode for lifecycle targets.
  - Continue rejecting ignore-errors, touch, and question modes.
  - Prove release, publish, and deploy planning succeeds without executing lifecycle scripts.

  ### Resolution
  Lifecycle targets now accept Make's standard no-execute modes while continuing to reject ignore-errors, touch, and question modes. The black-box lifecycle contract proves `make -n release`, `make -n publish`, and `make -n deploy` emit the canonical scripts without executing them. `make lifecycle-orchestration-contract-check`, `make release-workflow-check`, each exact dry-run target, and the complete `make ci` passed; the full gate included all release contracts, Go race coverage, the real containerized stack, and 456 browser scenarios in 3.6 minutes.

  Changed Files:
  `PLAN.md`, `.mprlab/ISSUES.md`, `CHANGELOG.md`, `Makefile`, `scripts/test-lifecycle-orchestration.sh`, and `scripts/validate-release-workflow.mjs`.

- [x] [B066] (P1) Measure deployable Linux memory instead of unused pages.
  ### Summary
  Production preflight rejected `tutosh` because Ansible reported 447 MiB of completely unused memory, even though Linux reported 3,808 MiB available to start a deployment without swapping. The preflight must measure deployable capacity rather than requiring the page cache to be empty.

  ### Deliverables
  - Read the canonical Linux `/proc/meminfo` `MemAvailable` value and require at least 512 MiB.
  - Fail closed when `MemAvailable` is absent or malformed; do not fall back to `MemFree`.
  - Keep the architecture and memory assertions distinct and report the detected capacity on failure.
  - Cover the app-owned preflight contract without invoking a production deployment.

  ### Resolution
  Replaced Ansible's `ansible_memfree_mb` assertion with a fail-closed parser for the Linux `MemAvailable` contract, kept architecture validation separate, and made low-capacity failures report the detected available MiB. A read-only host probe confirmed the rejected gateway had 447 MiB `MemFree` but 3,808 MiB `MemAvailable`, so no cache purge or gateway resize was warranted. The regression first failed with `deploy_preflight_must_measure_linux_available_memory`; `make release-workflow-check`, the pinned Ansible 2.19.8 production-playbook syntax check, and final `make ci` then passed, including Go race detection and all 457 browser scenarios in 4.3 minutes. No production deployment was attempted.

  Changed Files:
  `PLAN.md`, `.mprlab/ISSUES.md`, `.mprlab/deploy/ansible/tasks/preflight.yml`, `README.md`, and `scripts/validate-release-workflow.mjs`.

- [x] [B067] (P1) Verify the app-owned TAuth tenant contract without obsolete dotenv metadata.
  ### Summary
  Production preflight fails before deployment because the dependency verifier still requires `TAUTH_TENANT_ID_LOOPAWARE` and `TAUTH_SESSION_COOKIE_NAME_LOOPAWARE` in the shared TAuth dotenv. The app-owned `tauth_tenant` resource now owns those stable values, and the active gateway environment intentionally contains only the LoopAware signing secret.

  ### Deliverables
  - Read the LoopAware tenant ID and session cookie name from LoopAware's canonical environment while retaining the shared signing-key identity check.
  - Reject missing canonical LoopAware metadata and dependency secret mismatches without restoring obsolete shared TAuth dotenv keys.
  - Surface the verifier's sanitized failure reason from Ansible while keeping all dependency environment documents censored.
  - Cover the public dependency-verifier CLI and app-owned preflight contract without invoking a production deployment.

  ### Resolution
  Removed the verifier's obsolete reads of shared TAuth tenant ID and cookie metadata. The verifier now requires those stable values from LoopAware's canonical environment, continues to compare the LoopAware signing key with TAuth's active tenant signing key, and uses the canonical LoopAware values for the authenticated TAuth canary. The protected Ansible command records a non-fatal result under `no_log`, then a separate assertion reports only the verifier's sanitized stderr when the contract fails. Added black-box CLI coverage for the canonical environment shape, secret mismatch redaction, and missing app-owned metadata.

  The regression first failed with `tauth: missing required TAUTH_TENANT_ID_LOOPAWARE`. The focused dependency contract, aggregate release workflow contract, Python and shell syntax checks, Git diff check, and pinned Ansible 2.19.8 production-playbook syntax check passed. A live read-only invocation against the active LoopAware, TAuth, and Pinguin environments passed both authenticated canaries. Baseline `make ci` passed all 457 browser scenarios in 4.1 minutes, and final `make ci` passed all release and deployment contracts, Go race checks, and all 457 browser scenarios in 4.3 minutes. No production deployment or publication was attempted.

  Changed Files:
  `PLAN.md`, `.mprlab/ISSUES.md`, `.mprlab/deploy/ansible/tasks/preflight.yml`, `Makefile`, `scripts/test-loopaware-dependency-contract.sh`, `scripts/validate-release-workflow.mjs`, and `scripts/verify-loopaware-dependency-contract.py`.

- [x] [B068] (P0) Restore the production login control after mpr-ui config rejection.
  Goal:
  Restore the visible Google login control on the production landing page by bringing LoopAware's browser auth configuration onto the current shared `mpr-ui` contract.

  Requirements:
  Remove the retired `authButton` key from every environment, declare the canonical `/auth/session` path in each environment's auth config, and keep login presentation owned by the existing static `mpr-header` markup. Add no compatibility path or app-owned Google authentication scaffolding.

  Deliverables:
  - Update the served browser auth configuration and its external-asset test fixture.
  - Add black-box coverage that the served config applies the current session boundary, produces one visible header login control, and emits no orchestration error.
  - Verify the focused browser path and final repository CI gate.

  Validation:
  Reproduce the current production `config-ui.yaml does not allow authButton` console error, run the focused login browser scenarios, and run final `make ci` without deploying production.

  Resolution:
  Removed the retired `authButton` presentation object from both served browser environments and declared `/auth/session` as their canonical TAuth session boundary. Preserved the existing static `mpr-header sign-in-label="Login"` presentation owner, updated the CDN asset fixture to the current schema, and added a black-box browser regression that inspects the served YAML, verifies the applied session attribute and single visible header Google control, and rejects `mpr-ui-config` orchestration errors.

  Validation Results:
  Live read-only inspection reproduced the production `config-ui.yaml does not allow authButton` failure and confirmed the static header presentation was already present. `make lint-js` passed. `make test-integration` passed all 458 browser/API scenarios in 4.3 minutes. Final `make ci` passed config audit, builds, release and deployment contracts, JavaScript type checks, Go tests with race detection, and all 458 browser/API scenarios in 4.4 minutes. No release, publication, or production deployment was run.

  Review follow-up 2026-07-31:
  Made config audit fail closed when production environment inputs are absent and gave each production Compose service an explicit tracked `x-config-audit-env-file`; documentation-only `.env.*.example` and `.sample` paths are rejected. Removed the hard-coded test YAML parser so the browser regression now loads LoopAware's configured CDN parser and applies the actual served `/config-ui.yaml`. Aligned README environment guidance with those runtime and audit boundaries. `make config-audit`, `make lint-js`, and `make test-unit` passed; a clean archived copy with no private environment files passed config audit without warnings; final `make ci` passed all build, release, deployment-contract, type, Go race, and 458 browser/API scenarios in 4.5 minutes. No release, publication, or production deployment was run.

  Changed Files:
  `PLAN.md`, `.mprlab/ISSUES.md`, `.mprlab/deploy/docker-compose.yml`, `README.md`, `cmd/configaudit/main.go`, `cmd/configaudit/main_additional_test.go`, `cmd/configaudit/main_test.go`, `docker-compose.yml`, `docker-compose.computercat.yml`, `web/config-ui.yaml`, `tests/helpers/externalAssets.js`, and `tests/specs/header-auth-state.spec.js`.

- [x] [B069] (P0) Require signed tokens for public subscription state changes.
  Goal:
  Prevent unauthenticated callers from discovering or changing subscription state with only a site ID and email address.

  Requirements:
  - Keep public subscription creation and signed confirmation/unsubscribe email links.
  - Delete the tokenless status, confirmation, and unsubscribe contracts without aliases, compatibility responses, or origin-based authorization.
  - Preserve authenticated operator subscription management.

  Deliverables:
  - Remove the obsolete routes, handlers, payload types, helpers, and public documentation.
  - Add black-box API coverage that obsolete routes do not exist and signed link mutations still enforce valid tokens.

  Validation:
  - Run the focused public API integration suite and final `make ci`.

  Resolution:
  Deleted the public tokenless status, confirmation, and unsubscribe routes together with their handlers, payload types, helpers, tests, and API documentation. Subscription creation remains origin-bound, authenticated operators retain subscriber management, and recipient state changes now exist only behind signed confirmation and unsubscribe link tokens. Added black-box API coverage proving all three obsolete routes return 404 while valid, missing, invalid, already-used, confirmation, and unsubscribe token paths retain their current behavior.

  Validation Results:
  `make test-unit` passed. `make lint-js` passed. `make test-integration-api` passed all 122 API scenarios. The completion audit found and removed stale tokenless endpoint references from the tracked architecture and design documents. The exact final `make ci` passed configuration, browser, container, and workflow audits; vulnerability scans; builds; lint and type checks; Go tests and race tests; and all 452 browser/API scenarios in 4.4 minutes. The test-owned Compose projects cleaned up on exit; no production lifecycle action was run.

  Changed Files:
  `PLAN.md`, `.mprlab/ISSUES.md`, `ARCHITECTURE.md`, `README.md`, `docs/LA-100-email-subscriptions.md`, `docs/LA-116-split-frontend-backend.md`, `cmd/server/main.go`, `cmd/server/routes.go`, `internal/api/public.go`, `internal/api/public_handlers_additional_test.go`, `internal/api/public_subscription_errors_test.go`, `internal/api/public_test.go`, `tests/helpers/api.js`, and `tests/specs/api-public.spec.js`.

- [x] [B070] (P0) Prevent stored script execution in traffic path rendering.
  Goal:
  Render attacker-controlled visit paths as inert text on the authenticated traffic test page.

  Requirements:
  - Build traffic table rows with safe DOM APIs and text nodes; do not use HTML interpolation for stored visit fields.
  - Preserve the existing traffic data and display contract.

  Deliverables:
  - Replace the vulnerable row renderer.
  - Add a browser regression that records a poisoned path through the public collector, opens the authenticated page, and proves the payload renders as text without executing or creating attacker markup.

  Validation:
  - Run the focused traffic browser coverage and final `make ci`.

  Resolution:
  Replaced the authenticated traffic utility's stored-path HTML interpolation with explicit table-row and table-cell construction that assigns the path through `textContent`. Added a black-box browser regression that submits an image-event payload through the public visit collector, opens the authenticated traffic utility, and proves the exact payload remains visible as text without creating an image element or executing its handler.

  Validation Results:
  The focused Playwright scenario passed. Final `make ci` passed config audit, builds, release/deployment contracts, type checks, Go race checks, and all 448 browser/API scenarios in 4.4 minutes. The first focused harness invocation used a dashboard-only authentication wait on the standalone utility page; the corrected fixture uses the utility page's existing session-cookie contract. The test-owned Compose project was removed, and no production lifecycle action was run.

  Changed Files:
  `PLAN.md`, `.mprlab/ISSUES.md`, `web/app/traffic-test/index.html`, and `tests/specs/traffic-test-page.spec.js`.

- [x] [B071] (P0) Remove known vulnerable dependencies and unsupported container bases.
  Goal:
  Make shipped Go, JavaScript, mobile, and container artifacts use current supported dependency contracts with repeatable vulnerability checks.

  Requirements:
  - Upgrade reachable vulnerable Go modules and vulnerable production npm dependency paths.
  - Move build and runtime images to supported, immutable bases while preserving multi-architecture builds.
  - Add repository-native audit gates that fail on actionable vulnerabilities without introducing compatibility shims.

  Deliverables:
  - Updated manifests, lockfiles, Dockerfile bases, Make targets, and CI coverage.
  - Documented audit commands and zero-actionable-finding results for shipped artifacts.

  Validation:
  - Run Go vulnerability analysis, production npm audits, image inspection, the focused build gates, and final `make ci`.

  Resolution:
  Upgraded the four reachable vulnerable Go module families to current fixed releases, advanced Expo SDK 56 packages and the pinned EAS CLI to their supported patch contracts, and replaced vulnerable production and EAS-tool transitive npm paths with exact secure versions. Replaced the end-of-life Alpine 3.20 runtime and floating Go builder with supported Go 1.26.5/Alpine 3.24 multi-architecture manifest-list digests. Added a pinned `govulncheck` plus full npm audit and immutable container-base contract to `make security-audit` and `make ci`. Updated `tidy-check` to Go 1.26's non-mutating `go mod tidy -diff` so intentional module upgrades can pass while actual tidy drift still fails.

  Validation Results:
  Expo's dependency compatibility check reported all packages current. A clean `npm ci`, all three full npm audits, mobile configuration/API-boundary/type checks, and EAS CLI 21.5.0 startup passed with zero npm vulnerabilities. `govulncheck` reported zero affected symbols and zero vulnerable imported packages; its verbose module-only notice is the unfixed, unimported `x/crypto/openpgp` package. The pinned manifests resolved for amd64 and arm64, `docker build --pull` succeeded, and Docker Scout reported 0 critical, 0 high, 0 medium, and 0 low findings across the 64-package runtime image. Final `make ci` passed the new security gate, config audit, builds, release/deployment contracts, type checks, Go race checks, and all 448 browser/API scenarios in 4.3 minutes. The audit-only image and test-owned Compose project were removed; no production lifecycle action was run.

  Changed Files:
  `PLAN.md`, `.mprlab/ISSUES.md`, `Dockerfile`, `Makefile`, `README.md`, `go.mod`, `go.sum`, `mobile/eas.json`, `mobile/package.json`, `mobile/package-lock.json`, `mobile/scripts/validate-mobile-config.mjs`, and `scripts/audit-container-bases.sh`.

- [x] [B072] (P0) Enforce least-privilege GitHub repository controls.
  Goal:
  Protect production branches and automation from mutable actions, writable default tokens, unreviewed changes, and disabled security scanning.

  Requirements:
  - Pin workflow actions to immutable commits and declare least-privilege workflow permissions.
  - Protect `master` and the gateway-owned Pages branch without breaking the declared publication owner.
  - Restrict Actions and enable supported dependency, secret, and code scanning controls.

  Deliverables:
  - Hardened workflow source plus verified repository, branch, and scanning settings.
  - Black-box/static workflow contract coverage for immutable action references and required paths.

  Validation:
  - Validate workflow syntax and policy checks, query the applied GitHub settings, and run final `make ci`.

  Resolution:
  Pinned every CI action to its resolved 40-character GitHub commit, limited the workflow token to `contents: read`, added the Dockerfile and all GitHub control files to both event filters, and added weekly grouped Dependabot coverage for Go, Docker, Actions, and all three npm roots. Added tracked static workflow and authenticated live-repository audit commands. Restricted live Actions execution to full-SHA GitHub-owned actions, made default workflow tokens read-only and unable to approve pull requests, enabled Dependabot security updates, secret scanning, push protection, and extended CodeQL default setup. Protected `master` with the current `test` status, one stale-dismissing last-push-independent approval, linear history, resolved conversations, and no force pushes or deletion. Protected `gh-pages` from deletion while retaining the exact gateway-required force-with-lease activation contract.

  Validation Results:
  Workflow and Dependabot YAML parsing plus the static workflow policy gate passed. `make github-security-audit` verified all applied Actions, scanning, branch, and Pages settings. CodeQL setup run 30874488083 completed successfully for Actions, Go, JavaScript/TypeScript, and Python. Dependabot and secret scanning reported zero open alerts; CodeQL exposed 13 existing source findings that are tracked separately as B078 instead of being silently dismissed. Final `make ci` passed the new workflow policy check, vulnerability gates, config audit, builds, release/deployment contracts, type checks, Go race checks, and all 448 browser/API scenarios in 4.2 minutes. GitHub did not enable the optional non-provider secret patterns or validity checks for this public personal-account repository; the supported provider scanning and push-protection controls are enabled. No release, publication, or production deployment was run.

  Changed Files:
  `PLAN.md`, `.mprlab/ISSUES.md`, `.github/workflows/ci.yml`, `.github/dependabot.yml`, `Makefile`, `README.md`, `scripts/audit-github-workflow.py`, and `scripts/audit-github-repository.sh`; live GitHub repository settings were also updated.

- [x] [B073] (P1) Pin browser dependencies and enforce a static-hosting CSP.
  Goal:
  Make browser dependency delivery reproducible and constrain executable content on the GitHub Pages frontend.

  Requirements:
  - Replace mutable CDN selectors with one canonical immutable mpr-ui release and integrity-checked third-party assets.
  - Add a static-hosting-compatible CSP to every HTML entry point while preserving mpr-ui and TAuth ownership of authentication.
  - Keep third-party browser dependencies on the approved CDN path; do not vendor them or create app-owned auth scaffolding.

  Deliverables:
  - Pinned asset declarations, CSP policy, local proxy alignment, documentation, and automated drift guards.

  Validation:
  - Run browser auth, security-header, and static dependency policy scenarios plus final `make ci`.

  Resolution:
  Pinned mpr-ui v3.11.5 to its full release commit and added verified SHA-384 integrity to every directly fetched jsDelivr dependency, including the supported js-yaml 4.3.0 release. Added one canonical static CSP source, placed the matching policy before active content in all 49 HTML entry points, and aligned local, test, and ComputerCat proxy headers with the same policy plus edge-only `frame-ancestors 'none'`. Preserved mpr-ui/TAuth authentication ownership and the approved CDN-only dependency path. Added a static drift audit and a focused browser-security Make target; served-document assertions now verify pinned declarations without depending on third-party network timing.

  Validation Results:
  Re-fetched all four pinned assets and independently verified their SHA-384 digests. `make browser-security-audit` confirmed 49 protected HTML entry points, 44 immutable mpr-ui declarations, and three aligned proxy policies. `make lint-js` passed. The focused proxy/browser gate passed all 81 authentication, immutable-asset, CSP, and security-header scenarios in 59.7 seconds. Final `make ci` passed vulnerability scans, dependency audits, build, lint, type checks, Go tests, race tests, and all 448 browser/API scenarios in 4.0 minutes. No publication, release, or production deployment was run.

  Changed Files:
  `PLAN.md`, `.mprlab/ISSUES.md`, `Makefile`, `README.md`, `configs/content-security-policy.txt`, `docker-compose.yml`, `docker-compose.computercat.yml`, `tests/docker-compose.yml`, `tests/package.json`, `tests/helpers/externalAssets.js`, `tests/specs/header-auth-state.spec.js`, `tests/specs/security-headers.spec.js`, `scripts/audit-browser-assets.py`, and every `web/**/*.html` entry point.

- [x] [B074] (P1) Define trusted proxy and edge metadata boundaries.
  Goal:
  Ensure client identity, rate limiting, and location analytics cannot be influenced by untrusted forwarding or edge-geo headers.

  Requirements:
  - Require and validate the canonical trusted proxy CIDR configuration at startup.
  - Configure Gin's proxy trust explicitly and accept forwarded client metadata only across that boundary.
  - Reject or strip caller-supplied edge location metadata unless an owned proxy supplies the canonical value.

  Deliverables:
  - Typed runtime config, proxy middleware, deployment/local configuration, documentation, and black-box spoofing regressions.

  Validation:
  - Prove direct and proxied client-IP behavior, rate-limit resistance to spoofed headers, geo-header rejection, and final `make ci`.

  Resolution:
  Added a required typed proxy contract that accepts only canonical non-catch-all CIDRs, rejects duplicates, and requires every edge-geo proxy to be a subset of the general proxy boundary. The server now configures Gin's trusted proxies explicitly, uses only the owned `X-Forwarded-For` chain for client identity, and removes forwarding headers from direct or otherwise untrusted peers before HTTPS detection, logging, storage, and rate limiting. Edge location headers are independently stripped unless the immediate peer is in the explicit edge-geo boundary; every tracked environment keeps that boundary empty, so caller-supplied Cloudflare, Vercel, and CloudFront metadata is rejected. Local, test, and ComputerCat Compose stacks assign gHTTP a fixed address and trust only that address; the production resource declaration trusts the gateway-owned internal network. Public feedback, mobile feedback, subscriptions, and browser error limits are scoped by operation, site, and verified client IP so one site's traffic does not consume another site's window.

  Validation Results:
  Config-loading regressions proved missing, non-canonical, duplicate, catch-all, and out-of-bound edge proxy declarations fail closed. Middleware HTTP regressions proved direct spoof headers are removed and configured proxy metadata is accepted. The focused proxy stack passed five API cases and the dashboard geo-fallback case; its feedback, subscription, and browser-error tests rotated spoofed client headers on every request and still reached `429`. The exact final `make ci` passed config and browser policy audits, vulnerability scans, dependency audits, builds, lint and type checks, Go tests and race tests, and all 448 browser/API scenarios in 4.3 minutes. No release, publication, or production deployment was run.

  Changed Files:
  `PLAN.md`, `.mprlab/ISSUES.md`, `.mprlab/deploy/resources.yml`, `Makefile`, `README.md`, `cmd/configaudit/main_test.go`, `cmd/server/main.go`, `cmd/server/main_test.go`, `configs/.env.loopaware.example`, `configs/.env.loopaware.computercat.example`, `configs/config.loopaware.yml`, `docker-compose.yml`, `docker-compose.computercat.yml`, `internal/api/middleware.go`, `internal/api/middleware_test.go`, `internal/api/public.go`, `internal/api/public_helpers_test.go`, `internal/api/public_test.go`, `internal/api/sentry.go`, `internal/api/visit_collector_additional_test.go`, `internal/serverconfig/config.go`, `tests/configs/config.loopaware.yml`, `tests/configs/loopaware.env`, `tests/docker-compose.yml`, `tests/package.json`, `tests/specs/api-admin.spec.js`, `tests/specs/api-la-sentry.spec.js`, `tests/specs/api-public.spec.js`, and `tests/specs/dashboard-traffic.spec.js`.

- [x] [B075] (P1) Neutralize spreadsheet formulas in subscriber CSV exports.
  Goal:
  Prevent exported subscriber-controlled fields from executing as spreadsheet formulas.

  Requirements:
  - Apply the existing CSV cell neutralization contract to every subscriber-controlled string field.
  - Preserve CSV structure, Unicode data, and ordinary email/name values.

  Deliverables:
  - Hardened authenticated subscriber export and black-box formula-payload coverage.

  Validation:
  - Create malicious subscriber values through the public API, export them through the authenticated API, and run final `make ci`.

  Resolution:
  Applied the existing spreadsheet-formula neutralization contract to every string column in authenticated subscriber CSV exports. The black-box API flow now creates subscribers whose names begin with `=`, `+`, `-`, and `@`, plus an ordinary quoted Unicode name, then verifies that export prefixes formula-bearing cells while preserving email values, Unicode, and CSV quoting.

  Validation Results:
  `make test-unit`, `make lint-js`, and `make test-integration-api` passed; the focused API stack ran 122 scenarios including public subscriber creation and authenticated CSV export. The exact final `make ci` passed configuration and browser policy audits, vulnerability and dependency scans, builds, lint and type checks, Go tests and race tests, and all 448 browser/API integration scenarios in 4.4 minutes.

  Changed Files:
  `PLAN.md`, `.mprlab/ISSUES.md`, `README.md`, `internal/api/admin.go`, and `tests/specs/api-admin.spec.js`.

- [x] [B076] (P1) Bound HTTP request resources and public visit ingestion.
  Goal:
  Prevent oversized or slow requests and unbounded public visit bursts from consuming server resources.

  Requirements:
  - Enforce route-appropriate body limits at the HTTP edge with a stable 413 response.
  - Configure read and idle server timeouts and a bounded header size without breaking server-sent events.
  - Add a dedicated public visit ingestion limit that preserves normal tracking traffic.

  Deliverables:
  - Central request-limit middleware, server resource settings, visit limiter, documentation, and black-box limit coverage.

  Validation:
  - Exercise oversized, slow-boundary, SSE, and visit-burst behavior through public HTTP routes and run final `make ci`.

  Resolution:
  Added a central route-aware body boundary that reads request content before dispatch, permits 64 KiB for standard mutations and 1 MiB for protected LA Sentry ingestion, rejects bodies on pixel visits and bodyless methods, and returns the stable `413 request_too_large` response for oversized content. The HTTP server now bounds header reads, complete request reads, idle connections, and header bytes while explicitly retaining a zero write timeout for server-sent events. Public visits use a separate bounded 120-request/30-second window per site and verified client address with bounded counter storage.

  Validation Results:
  Server-construction coverage verified the read, idle, header, and zero-write-timeout contract. The black-box API stack used a random loopback-only backend port to prove an incomplete body is cut off at the 10-second read boundary while authenticated SSE remains available, oversized JSON returns `413`, and the 120th visit succeeds while the 121st returns `429` despite rotating spoofed forwarding headers. `make test-unit`, `make lint`, and `make test-integration-api` passed. The exact final `make ci` passed configuration and browser policy audits, vulnerability and dependency scans, builds, lint and type checks, Go tests and race tests, and all 451 browser/API scenarios in 4.5 minutes.

  Changed Files:
  `PLAN.md`, `.mprlab/ISSUES.md`, `README.md`, `cmd/server/main.go`, `cmd/server/main_test.go`, `internal/api/middleware.go`, `internal/api/public.go`, `internal/api/visit_collector.go`, `tests/docker-compose.yml`, `tests/helpers/config.js`, `tests/scripts/run-integration.sh`, and `tests/specs/api-public.spec.js`.

- [x] [B077] (P1) Enforce private modes for local secret inputs.
  Goal:
  Prevent local dotenv and private-key inputs from being group- or world-readable.

  Requirements:
  - Fail configuration audit when an existing private input is not mode `0600`.
  - Repair current ignored local secret/key modes without reading or exposing their contents.
  - Keep absent local inputs compatible with the tracked audit-fixture contract.

  Deliverables:
  - Config-audit enforcement, CLI-level regression coverage, corrected local modes, and documentation alignment.

  Validation:
  - Exercise secure, insecure, and absent inputs through the public config-audit command and run final `make ci`.

  Resolution:
  Extended `config-audit` to collect actual Compose env files, repo-root and canonical `configs/` dotenv inputs, `.mprlab/deploy/.env`, and explicitly mounted private-key files, then require every existing private input to be a non-symlink regular mode-`0600` file. Documentation examples and tracked audit fixtures remain outside the private runtime-input set, and missing runtime env/key files remain valid for source-only audits. The ComputerCat Compose contract now mounts only the exact certificate and private-key files instead of the containing certificate directory. Without reading their contents, narrowed `configs/.env.ghttp` and the four ignored `configs/.env.*.integration` files from `0644` to `0600`; all other discovered local private inputs were already `0600`, and the documented external ComputerCat key was absent.

  Validation Results:
  CLI-level regressions exercised absent runtime inputs with a tracked audit fixture, secure dotenv/key/deployment inputs, separate insecure dotenv, private-key, and deployment modes, and rejection of a mode-`0600` symlink to a regular private-key target. `make config-audit` and `make test-unit` passed. The exact final `make ci` passed the enforced mode audit, browser and workflow policy audits, vulnerability and dependency scans, builds, lint and type checks, Go tests and race tests, and all 452 browser/API scenarios in 4.4 minutes.

  Changed Files:
  `PLAN.md`, `.mprlab/ISSUES.md`, `README.md`, `cmd/configaudit/main.go`, `cmd/configaudit/main_test.go`, and `docker-compose.computercat.yml`; local permission-only changes were applied to five ignored `configs/.env.*` files.

- [ ] [B078] (P0) Resolve the CodeQL security findings exposed by default setup.
  Goal:
  Eliminate actionable CodeQL security paths and leave every non-production or tool-only detection with an evidence-backed disposition.

  Requirements:
  - Replace any attacker-influenced DOM-to-HTML reinterpretation and polynomial regular-expression paths with bounded, text-safe contracts.
  - Remove exploitable filesystem race or log-injection behavior without weakening existing functionality.
  - Treat test-fixture detections as findings until source and runtime evidence prove they are non-production false positives; dismiss only with a specific recorded justification.
  - Keep the immutable workflow-permissions fix that addresses the remote master alert already corrected by B072.

  Deliverables:
  - Production fixes, focused black-box regressions, explicit false-positive dispositions, and a zero-unreviewed-alert CodeQL result after the corrected source reaches GitHub.

  Validation:
  - Run focused browser/client/tool coverage, local CodeQL analysis when available, inspect the resulting GitHub alerts without exposing secrets, and run final `make ci`.

  Implementation Status (2026-08-03):
  Replaced the DOM-configured unauthorized redirect with the canonical `/login` path and removed the obsolete redirect field, replaced both React Native slash-trimming expressions with a length-bounded linear scan, replaced test fixture `Math.random` identifiers with `crypto.randomBytes`, and changed the iOS project patcher to open one no-follow file descriptor for validation, reading, writing, truncation, and sync. Request logging now removes carriage returns and line feeds, repairs invalid UTF-8, and caps every request-derived field before structured logging. The test harness now accepts only HTTP(S) loopback frontend and backend origins.

  Validation Results:
  Focused React Native package, mobile tool, Go HTTP, and browser-security gates passed, including a poisoned dashboard redirect, bounded request-log values, iOS patcher idempotence, and symlink rejection. The exact GitHub analyzer version, CodeQL 2.26.2 with the security-extended suites, reported zero Go findings and zero Actions findings. JavaScript reported only two paths in which the non-secret `TAUTH_SESSION_COOKIE_NAME` test fixture is placed into a Cookie header sent to the runner-owned loopback stack; GitHub alerts 9 and 10 were dismissed as false positives with that specific runtime justification. The remaining 11 live alerts still describe the old `10ad3f511c49c6dedebb03ed3514e754e03e2c69` master source and must close through a default-setup rerun after the corrected source reaches GitHub. No commit or push was performed without operator authorization.

  Follow-up (2026-08-04):
  Removed the direct file-derived Cookie request header from the slow-body/SSE regression instead of retaining its false-positive disposition. The regression now installs the signed test session as Playwright browser state and verifies the SSE response through a same-origin browser fetch, preserving the authenticated resource-boundary contract without an explicit outbound header data flow.

  Final `make ci` passed the configuration, browser, container, and workflow audits; Go and npm vulnerability scans; builds; lint and type checks; Go tests and race tests; and all 452 browser/API scenarios in 4.4 minutes. The authenticated live-repository audit also reconfirmed least-privilege Actions, extended CodeQL default setup, protected production branches, zero open Dependabot alerts, and zero open secret-scanning alerts. The integration project shut down cleanly, and no release, publication, or production deployment was run.

  Changed Files:
  `PLAN.md`, `.mprlab/ISSUES.md`, `README.md`, `clients/react-native/src/index.tsx`, `internal/api/middleware.go`, `internal/api/middleware_test.go`, `mobile/scripts/fix-ios-project-warnings.mjs`, `mobile/scripts/validate-mobile-config.mjs`, `tests/README.md`, `tests/helpers/config.js`, `tests/helpers/fixtures.js`, `tests/specs/api-public.spec.js`, `tests/specs/header-auth-state.spec.js`, and `web/app/index.html`; GitHub alerts 9 and 10 also received evidence-specific false-positive dispositions before the direct test request-header flow was removed.

- [ ] [B079] (P1) Restore authenticated landing redirects and permit trusted browser asset connections.
  Goal:
  Make authenticated landing states resolve to the dashboard and eliminate CSP violations caused by trusted immutable browser assets.

  Requirements:
  - Redirect authenticated `/` and `/login` visits to `/app`, including silent session restoration, so landing content cannot remain paired with an authenticated header.
  - Preserve the visible Google sign-in control for anonymous sessions and retain mpr-ui ownership of Google authentication.
  - Keep authenticated resource, pricing, policy, and subscription pages on their current public-page contract unless the user starts a login flow.
  - Allow browser connections to the already trusted jsDelivr asset origin without widening the policy to unrelated origins.
  - Keep every HTML meta policy and proxy response-header policy aligned with the canonical CSP.

  Deliverables:
  - Authenticated landing redirects, canonical CSP alignment, black-box session-restoration coverage, and browser-boundary verification.

  Validation:
  - Run the browser asset/security gates, authenticated landing regression, final `make ci`, and a Chrome reload against the owned local stack.

  Correction in Progress (2026-08-04):
  The first B079 implementation incorrectly made authenticated landing content a supported state by adding an account menu and asserting that `/` remained visible after session restoration. The required contract is the inverse: authenticated `/` and `/login` states must navigate to `/app`. B079 was reopened to remove that menu, restore the redirect invariant, and replace the incorrect regression. The correction also removed the persisted explicit-logout override that could downgrade a newly restored authenticated session after navigation; the current mpr-ui contract emits its logout event only after TAuth accepts the logout request.

  Hosted CI Follow-up (2026-08-04):
  The first hosted correction run passed CodeQL and 455 of 456 full integration scenarios, then exposed one same-document transition gap: an authenticated DOM snapshot could clear the in-memory logout-pending overlay immediately after mpr-ui emitted its server-confirmed logout event. The follow-up keeps that transient state for the lifetime of the current document while leaving the removed cross-navigation persistence deleted, so a newly loaded authenticated document still redirects to `/app`. The focused browser-security target now includes the logout-hardening scenarios that exposed this boundary.

  Validation Results:
  The clean pre-change `make ci` baseline passed all 452 browser/API scenarios. The initial corrected focused `make test-integration-browser-security` gate passed all 86 scenarios, including anonymous Google sign-in, normal and silent authenticated root redirects, and the jsDelivr response-policy assertion; after the hosted logout-transition finding, the expanded focused gate passed all 100 auth, logout, CSP, and security-header scenarios. A cache-bypassing Chrome reload reached a hydrated `/app` with both auth layers authenticated, successful `/auth/session` and `/api/me` responses, and no CSP violation after the ignored local verifier input was aligned with the active TAuth tenant. The local full gate passed every pre-integration audit, vulnerability scan, build, lint/type check, Go test, and race test; three unchanged 456-scenario integration attempts were terminated only by the binding 350-second outer watchdog after 400, 400, and 371 passing scenarios on a Docker host shared with unrelated active workloads. The first hosted full run passed 455 scenarios and exposed the corrected logout transition above; a fresh hosted run remains the completion boundary.

  Changed Files:
  `PLAN.md`, `.mprlab/ISSUES.md`, `configs/content-security-policy.txt`, `docker-compose.yml`, `docker-compose.computercat.yml`, `tests/docker-compose.yml`, `tests/helpers/fixtures.js`, `tests/package.json`, `tests/specs/header-auth-state.spec.js`, `tests/specs/logout-hardening.spec.js`, `tests/specs/security-headers.spec.js`, `web/header-auth.js`, `web/index.html`, `web/login/index.html`, and the other 47 `web/**/*.html` entry points whose CSP meta declaration mirrors the canonical policy.


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

  Resolution:
  Replaced the schema-1 dispatch stub with the complete schema-v3 resource graph for LoopAware's backend, retained data, exported HTTP capability, Caddy route, public health check, Pages site, React Native package, native app, TAuth tenant, and deployment-only private values. Removed the obsolete app-owned production Ansible, Compose, artifact, publication, store, and deployment controllers; the three production lifecycle targets now delegate to the exact sibling gateway. Added the pinned EAS contract, advanced the npm client to `0.7.52`, excluded the private deployment input from Git and Docker contexts, and preserved local Compose plus mobile development, native identity, API-boundary, and config validation.

  Validation Results:
  The clean pre-change `make ci` baseline passed all 458 Playwright/API scenarios. Focused `make mobile-check`, `make config-audit`, and `go test ./cmd/configaudit` passed. Sealed gateway `plan-app-release`, `plan-app-publish`, and `plan-app-deploy` validation passed against the candidate committed bytes and a secret-free private-input fixture. Final `make ci` passed all Go, JavaScript, mobile, package, race, config-audit, and 458 Playwright/API scenarios after restoring the still-current local mobile invariants. Prepared the ignored mode-`0600` deployment input from the existing private LoopAware and TAuth values, verified the six exact nonempty bindings, and did not log secret bytes. No release, publication, or production deployment was run.

  Blocked: the pinned EAS CLI has no authenticated Expo account or linked `loopaware-mobile` project in this checkout. An operator must authenticate, initialize the EAS project, complete one interactive production build for iOS and Android so Expo and store credentials are configured, and commit the generated project linkage before the gateway's non-interactive release can succeed.

  Changed Files:
  `.dockerignore`, `.github/workflows/ci.yml`, `.gitignore`, `.mprlab/ISSUES.md`, `.mprlab/deploy/resources.yml`, `.mprlab/deploy/ansible/**`, `.mprlab/deploy/docker-compose.yml`, `CHANGELOG.md`, `Makefile`, `README.md`, `clients/react-native/package.json`, `clients/react-native/package-lock.json`, `cmd/configaudit/main.go`, `configs/.env.loopaware.example`, `configs/.env.loopaware.computercat.example`, `mobile/eas.json`, `mobile/package.json`, `mobile/package-lock.json`, `mobile/scripts/validate-mobile-config.mjs`, removed mobile production build/publish scripts, and removed app-owned lifecycle scripts under `scripts/` and `scripts/release/`.

- [x] [I034] (P1) Declare LoopAware's TAuth tenant requirements in the app-owned deployment manifest.
  ### Summary
  Keep stable tenant identity, hosted origin, native redirect metadata, cookie names, and provider references in .mprlab/deploy/resources.yml; credential values, shared TTLs, and server policy remain gateway-owned.
  ### Deliverables
  - Add one canonical tauth_tenant resource.
  - Prove gateway fleet assembly and generic TAuth doctor validation succeed.
  ### Resolution
  Added the canonical app-owned `tauth_tenant` declaration. The gateway assembled the 13-tenant fleet, the generic TAuth doctor accepted it with zero errors, and—after removing the stale Compose stack left by an earlier timeout-killed test run—`make ci` passed all 455 browser scenarios plus the full Go, mobile, integration, and release-contract suite.
- [x] [I001] (P1) Add a public SEO resources cluster.
  ### Summary
  LoopAware needs a deliberate set of crawlable, internally linked public resource pages that advertise distinct product surfaces without creating hidden doorway pages or thin near-duplicate pages.
  ### Deliverables
  - Add a `/resources` index that links to focused public pages.
  - Add focused pages for feedback, subscribers, analytics, LA Sentry, self-hosting, SaaS, agencies, and lightweight analytics use cases.
  - Keep every crawlable page canonical, indexable, and linked from at least one other public resource page.
  - Update `robots.txt`, `sitemap.xml`, README public-page docs, and black-box SEO coverage.
  ### Resolution
  Added a crawlable `/resources` hub and eight focused resource pages for feedback widgets, subscriber capture, privacy-first analytics, lightweight analytics, LA Sentry monitoring, self-hosted feedback, SaaS feedback, and agency client sites. Each page has crawl-friendly metadata, canonical URLs, structured data, and internal links within the resource cluster while leaving the main login surface unlinked. Updated `robots.txt`, `sitemap.xml`, README documentation, and black-box SEO coverage. Validation passed with `git diff --check`, `make test-integration-all`, Browser checks for `/resources` and `/resources/la-sentry-monitoring`, and final `make ci`.
  ### Changed Files
  `PLAN.md`, `.mprlab/ISSUES.md`, `README.md`, `tests/specs/seo-public-pages.spec.js`, `web/robots.txt`, `web/sitemap.xml`, `web/resources/index.html`, `web/resources/styles.css`, `web/resources/feedback-widget/index.html`, `web/resources/subscriber-capture/index.html`, `web/resources/privacy-first-analytics/index.html`, `web/resources/lightweight-analytics/index.html`, `web/resources/la-sentry-monitoring/index.html`, `web/resources/self-hosted-feedback/index.html`, `web/resources/saas-feedback/index.html`, `web/resources/agency-client-sites/index.html`.
- [x] [I002] (P1) Tighten visitor-location edge geo inference and aggregation.
  ### Summary
  Review found that country-only edge geo is dropped when the country is not in LoopAware's small country-anchor map, and the locations endpoint now scans every matching visit row before aggregating in Go.
  ### Deliverables
  - Preserve country-only edge geo as the strongest available signal even when the country lacks a local centroid.
  - Group raw location signal tuples in SQL and apply inferred-location aggregation by grouped counts.
  - Add focused regression coverage for unmapped country-only edge geo.
  ### Resolution
  Preserved country-only edge geo for unmapped ISO countries by keeping it as an edge_geo location with a shared unmapped-country anchor instead of falling back to timezone or locale. Changed the locations query to group raw signal tuples with COUNT(*) before inference, so aggregation no longer scans one inferred row per visit. Added regression coverage for unmapped country-only CloudFront geo. Validation passed with `go test ./internal/api -run 'TestDatabaseSiteStatisticsProviderLocationDistribution|TestLocationDistributionHelperFunctions'`, `make test-unit`, `make lint-js`, and `make ci` including 405 Playwright/API integration specs.
  ### Changed Files
  `PLAN.md`, `.mprlab/ISSUES.md`, `internal/api/site_stats.go`, `internal/api/site_stats_additional_test.go`.
- [x] [I003] (P1) Improve visitor-location confidence with edge geo signals.
  ### Summary
  The Locations map now uses timezone, locale, network, and unknown signals, but timezone and locale alone are still weak location proxies. When LoopAware runs behind an edge or reverse proxy that supplies geo headers, the collector should preserve that signal and the analysis layer should report confidence.
  ### Deliverables
  - Collect supported edge geo headers into visit records.
  - Prefer edge geo over timezone/locale when inferring visitor location and expose country, region, city, and confidence metadata.
  - Include the stronger location signal in API responses, dashboard map metadata, CSV export, traffic report emails, docs, and tests.
  ### Resolution
  Added trusted edge geo collection for Cloudflare, Vercel, and CloudFront visit requests, storing source, country, region, city, and coordinates on visit records. Location inference now prefers edge geo over timezone, locale, and local-network fallbacks, exposes country/region/city/confidence metadata in the locations API, dashboard map DOM, CSV export, and traffic report emails, and documents the stronger signal path. Validation passed with `make test-unit`, `make lint-js`, `make test-integration-api`, `make test-integration`, and final `make ci` including 405 Playwright/API integration specs.
  ### Changed Files
  `PLAN.md`, `.mprlab/ISSUES.md`, `ARCHITECTURE.md`, `README.md`, `internal/model/visit.go`, `internal/model/visit_test.go`, `internal/api/admin.go`, `internal/api/admin_test.go`, `internal/api/site_stats.go`, `internal/api/site_stats_additional_test.go`, `internal/api/templates/traffic_report_email.txt`, `internal/api/traffic_report_schedule_test.go`, `internal/api/visit_collector.go`, `internal/api/visit_collector_additional_test.go`, `tests/helpers/api.js`, `tests/specs/api-admin.spec.js`, `tests/specs/dashboard-traffic.spec.js`, `web/app/index.html`.
- [x] [I004] (P1) Replace visitor timezone reporting with inferred locations.
  ### Summary
  The selected-site Traffic map currently presents browser timezones as the reporting dimension, but the useful operator question is visitor location. Timezones should become one signal among several, alongside browser locale and local-network/unknown fallbacks.
  ### Deliverables
  - Supersede the selected-site Timezones segment with Locations.
  - Collect browser locale during traffic collection and infer one visitor-location point from timezone, locale, network, or unknown signals.
  - Replace the `/visits/timezones` API contract with `/visits/locations`; no compatibility alias is required.
  - Update dashboard, traffic report email, CSV export, docs, and black-box coverage for the location contract.
  ### Resolution
  Superseded the selected-site Timezones surface with Locations and replaced the traffic API contract with `/api/sites/:id/visits/locations`, with no old `/visits/timezones` route alias. The traffic pixel and public collector now store browser locale, location distribution infers one point per visit from timezone, locale, local network, or unknown signals, and the dashboard map exposes label/source/signal metadata. CSV export and traffic report emails now include inferred location details, docs describe the new endpoint and collection signals, and the active world-map generator/check was renamed to location terminology. Validation passed with baseline `make ci`, `make test-unit`, `make lint-js`, `make test-integration-api`, `make test-integration`, and final `make ci` with 404 Playwright/API integration specs.
  ### Changed Files
  `PLAN.md`, `.mprlab/ISSUES.md`, `ARCHITECTURE.md`, `README.md`, `Makefile`, `cmd/server/main.go`, `cmd/server/routes.go`, `internal/model/visit.go`, `internal/model/visit_test.go`, `internal/api/admin.go`, `internal/api/admin_helpers_test.go`, `internal/api/admin_test.go`, `internal/api/site_stats.go`, `internal/api/site_stats_additional_test.go`, `internal/api/templates/traffic_report_email.txt`, `internal/api/traffic_report_schedule.go`, `internal/api/traffic_report_schedule_test.go`, `internal/api/visit_collector.go`, `internal/api/visit_collector_additional_test.go`, `tests/helpers/api.js`, `tests/package.json`, `tests/scripts/generate-location-world-map.mjs`, `tests/scripts/generate-timezone-world-map.mjs`, `tests/specs/api-admin.spec.js`, `tests/specs/dashboard-elements.spec.js`, `tests/specs/dashboard-labels.spec.js`, `tests/specs/dashboard-traffic.spec.js`, `web/app/index.html`, `web/pixel.js`.
- [x] [I005] (P1) Add selected-site Traffic intervals and CSV export.
  ### Summary
  The selected-site Traffic card always shows all-time totals and does not offer a direct export for the traffic data operators are reviewing.
  ### Deliverables
  - Add a Traffic interval selector with 1 day, 30 days, and All options.
  - Apply the selected interval to the selected-site traffic totals, trend, pages, sources, engagement, devices, and timezones.
  - Add an authenticated selected-site Traffic CSV download using the selected interval.
  - Add black-box API and dashboard coverage for interval filtering and CSV export.
  ### Resolution
  Added selected-site Traffic interval controls for 1 day, 30 days, and All, applied the interval to selected-site traffic endpoints and dashboard refreshes, and added authenticated CSV export with formula-safe cells. Added black-box API and dashboard coverage for interval filtering, CSV export, and labels. `make test-unit` and `make ci` pass.
  ### Changed Files
  `PLAN.md`, `.mprlab/ISSUES.md`, `ARCHITECTURE.md`, `README.md`, `cmd/server/main.go`, `cmd/server/routes.go`, `internal/api/admin.go`, `internal/api/site_stats.go`, `internal/api/admin_helpers_test.go`, `internal/api/admin_test.go`, `internal/api/site_stats_additional_test.go`, `internal/api/traffic_report_schedule_test.go`, `tests/specs/api-admin.spec.js`, `tests/specs/dashboard-elements.spec.js`, `tests/specs/dashboard-labels.spec.js`, `tests/specs/dashboard-traffic.spec.js`, `web/app/index.html`.
- [x] [I006] (P1) Replace duplicate timezone views with a visit-density map.
  ### Summary
  The selected-site Traffic tab rendered timezone distribution twice as a bar chart and a table. Operators need one map view where higher visit counts appear as larger bubbles.
  ### Deliverables
  - Replace the timezone bar chart/table pair with a single map view.
  - Render timezone visit counts as map bubbles whose radius scales with visits.
  - Preserve empty and error states in the new map container.
  - Add black-box dashboard coverage for map rendering and bubble sizing.
  ### Resolution
  Replaced the duplicate timezone chart/table surface with an SVG timezone map backed by the existing `/visits/timezones` API response. The renderer uses known IANA timezone anchors plus region-based placement for unsupported zones, labels each bubble, exposes accessible SVG titles, and scales bubble radius by visit count. Updated dashboard traffic coverage to assert visible map bubbles, proportional sizing, placeholder text, and fetch failure text. Validation passed with `make lint-js`, `npm --prefix tests run test -- specs/dashboard-traffic.spec.js`, and `npm --prefix tests run test -- specs/dashboard-elements.spec.js`.
  Final validation passed with `make ci` on 2026-06-08, including 393 Playwright/API integration specs.
  ### Changed Files
  `PLAN.md`, `web/app/index.html`, `tests/specs/dashboard-traffic.spec.js`, `tests/specs/dashboard-elements.spec.js`, `.mprlab/ISSUES.md`.
- [x] [I007] (P1) Replace duplicate device views with an icon row graph.
  ### Summary
  The selected-site Traffic tab rendered device distribution twice as a bar chart and a table. Operators need one device row graph where each row shows a device icon and the visit count.
  ### Deliverables
  - Remove the device distribution table so the Devices section has one view.
  - Render each device row with a device icon, label, proportional bar, and visit count.
  - Preserve empty and error states in the device row graph container.
  - Add black-box dashboard coverage for the singular device row graph.
  ### Resolution
  Replaced the duplicate device chart/table surface with a single row graph in `#device-types-chart`. Each row now renders a desktop, tablet, or mobile SVG icon, a device label, a proportional bar, and the visible visit count. Updated dashboard traffic coverage to assert device rows, icons, counts, placeholder text, and fetch failure text. Validation passed with `make lint-js` and `npm --prefix tests run test -- specs/dashboard-traffic.spec.js specs/dashboard-elements.spec.js`.
  Final validation passed with `make ci` on 2026-06-08, including 393 Playwright/API integration specs.
  ### Changed Files
  `PLAN.md`, `web/app/index.html`, `tests/specs/dashboard-traffic.spec.js`, `.mprlab/ISSUES.md`.
- [x] [I008] (P1) Replace duplicate top-pages views with a ranked path row graph.
  ### Summary
  The selected-site Traffic tab rendered path data twice as a bar chart and a table. Operators need one path view that ranks pages and keeps the visit count visible.
  ### Deliverables
  - Remove the duplicate Top pages table so the section has one view.
  - Render each path row with rank, a page icon, the path, a proportional bar, and the visit count.
  - Preserve empty and error states in the path row graph container.
  - Add black-box dashboard coverage for the singular path row graph.
  ### Resolution
  Replaced the duplicate Top pages chart/table surface with a single ranked row graph in `#top-pages-chart`. Each row now renders rank, an inline page SVG icon, a path label, a proportional bar, and the visible visit count. Updated dashboard traffic coverage to assert path rows, icons, ranks, counts, placeholder text, and stats failure text, and updated the shell element coverage for the removed table body. Validation passed with `make lint-js`, `npm --prefix tests run test -- specs/dashboard-traffic.spec.js specs/dashboard-elements.spec.js`, and `make ci` with 393 Playwright/API integration specs.
  ### Changed Files
  `PLAN.md`, `web/app/index.html`, `tests/specs/dashboard-traffic.spec.js`, `tests/specs/dashboard-elements.spec.js`, `.mprlab/ISSUES.md`.
- [x] [I009] (P1) Rename selected-site traffic breakdown section headings.
  ### Summary
  The selected-site Traffic tab should use parallel section labels under the total visits graph: Pages, Devices, and Timezones.
  ### Deliverables
  - Rename the Top pages heading to Pages.
  - Rename the Timezone map heading to Timezones.
  - Preserve the Devices heading and add black-box coverage for all three section labels.
  ### Resolution
  Renamed the selected-site traffic breakdown headings to Pages, Devices, and Timezones, added stable heading IDs, and covered those labels in the dashboard shell and label specs. Validation passed with `make lint-js` and `make ci` with 396 Playwright/API integration specs.
  ### Changed Files
  `PLAN.md`, `web/app/index.html`, `tests/specs/dashboard-labels.spec.js`, `tests/specs/dashboard-elements.spec.js`, `.mprlab/ISSUES.md`.
- [x] [I010] (P1) Clean up the selected-site timezone map visual treatment.
  ### Summary
  The first timezone map used crude land shapes and oversized labels, making the selected-site Timezones section look unpolished.
  ### Deliverables
  - Remove the fake continent shapes from the timezone map.
  - Use a cleaner map surface with smaller proportional bubbles.
  - Move timezone labels into readable pills with connector lines.
  - Add black-box coverage for the cleaned-up map contract.
  ### Resolution
  Reworked the timezone map into a restrained coordinate map with subtle grid lines, capped bubbles, connector lines, and label pills. Removed the faux land paths entirely and updated dashboard traffic coverage to assert the cleaner map structure and bubble sizing. Validation passed with `make lint-js`, `npm --prefix tests run test -- specs/dashboard-traffic.spec.js`, and `make ci` with 396 Playwright/API integration specs.
  ### Changed Files
  `PLAN.md`, `web/app/index.html`, `tests/specs/dashboard-traffic.spec.js`, `.mprlab/ISSUES.md`.
- [x] [I011] (P1) Render the Timezones bubble graph on a real world map.
  ### Summary
  The cleaned timezone map was visually restrained but no longer looked like a real map, so the Timezones section lost the geographic context required for a map-based bubble graph.
  ### Deliverables
  - Add a recognizable world land outline behind the timezone bubbles.
  - Keep the proportional visit bubbles and readable labels on top of the map.
  - Align timezone coordinates to the same projection used by the land outline.
  - Add dashboard coverage requiring the world land layer.
  ### Resolution
  Added a static simplified Natural Earth world land outline behind the Timezones bubbles, aligned the timezone projection to the same centered equirectangular map, and kept the capped visit bubbles plus readable label pills on top. Updated dashboard traffic coverage to require the world land layer. Validation passed with `make lint-js`, `npm --prefix tests run test -- specs/dashboard-traffic.spec.js`, and `make ci` with 396 Playwright/API integration specs.
  ### Changed Files
  `PLAN.md`, `web/app/index.html`, `tests/specs/dashboard-traffic.spec.js`, `.mprlab/ISSUES.md`.
- [x] [I012] (P1) Prove Timezones bubbles use real geographic placement.
  ### Summary
  The Timezones bubble graph must be an actual world map, with circles placed from each timezone's latitude and longitude rather than arbitrary dashboard layout positions.
  ### Deliverables
  - Expose the world-map source and projection on the rendered SVG.
  - Store each bubble's source latitude and longitude in the rendered circle.
  - Add black-box dashboard coverage asserting known timezone bubbles render at their expected projected coordinates.
  ### Resolution
  Exposed the Timezones SVG as a Natural Earth 110m equirectangular world map, stored each rendered bubble's source latitude and longitude, and expanded dashboard coverage to assert New York and London bubbles land at their expected projected coordinates. Validation passed with `make lint-js`, `npm --prefix tests run test -- specs/dashboard-traffic.spec.js`, and `make ci` with 396 Playwright/API integration specs.
  ### Changed Files
  `PLAN.md`, `web/app/index.html`, `tests/specs/dashboard-traffic.spec.js`, `.mprlab/ISSUES.md`.
- [x] [I013] (P1) Back the Timezones world map with supported geo tooling.
  ### Summary
  The Timezones map should not rely on an opaque hand-edited SVG path. The land outline needs a reproducible source using maintained geospatial packages while keeping the dashboard runtime simple.
  ### Deliverables
  - Generate the Natural Earth world land path from `world-atlas` through `topojson-client` and D3's equirectangular projection.
  - Keep the generated path embedded in the dashboard HTML so the browser does not need a runtime map library or bundler.
  - Add a CI-enforced check that fails when the embedded path drifts from the generator output.
  - Preserve black-box dashboard coverage for the rendered map source, projection, bubbles, and geographic coordinates.
  ### Resolution
  Added `tests/scripts/generate-timezone-world-map.mjs`, which converts `world-atlas` Natural Earth `land-110m` TopoJSON to the dashboard SVG path through `topojson-client` and D3's equirectangular projection. The embedded land path is now marked as generated, and `make lint-js` runs `npm --prefix tests run check:timezone-map` so CI fails if the checked-in path drifts from the generator. Dashboard coverage still asserts the rendered map source, projection, bubble sizes, and projected New York/London coordinates. Validation passed with `make lint-js`, `env LOOPAWARE_BASE_URL=http://localhost:8090 npm --prefix tests run test -- specs/dashboard-traffic.spec.js`, and `make ci` with 396 Playwright/API integration specs.
  ### Changed Files
  `PLAN.md`, `Makefile`, `tests/package.json`, `tests/package-lock.json`, `tests/scripts/generate-timezone-world-map.mjs`, `web/app/index.html`, `.mprlab/ISSUES.md`.
- [x] [I014] (P1) Differentiate tablet and mobile device icons in the Devices row graph.
  ### Summary
  The Devices row graph uses tablet and mobile icons with similar narrow outlines, so operators cannot quickly distinguish the two rows.
  ### Deliverables
  - Render tablet as a clearly wider slate icon.
  - Render mobile as a narrow phone icon with distinct phone details.
  - Keep desktop as a monitor icon.
  - Add dashboard coverage proving all three device icon shapes are distinct.
  ### Resolution
  Rendered mobile as a narrow phone, tablet as a landscape slate, and desktop as a monitor. Expanded the dashboard traffic spec to seed mobile, tablet, and desktop visits and assert each rendered icon frame has distinct dimensions. Focused validation passed with `make lint-js` and `env LOOPAWARE_BASE_URL=http://localhost:8090 npm --prefix tests run test -- specs/dashboard-traffic.spec.js`. Full completion-gate validation later passed on the same branch with `make ci` and 396 Playwright/API integration specs, so the earlier transient integration blocker is no longer active.
  ### Changed Files
  `PLAN.md`, `web/app/index.html`, `tests/specs/dashboard-traffic.spec.js`, `.mprlab/ISSUES.md`.
- [x] [I015] (P1) Mark X-axis time labels on traffic trend charts.
  ### Summary
  Traffic trend charts expose the count scale, but the horizontal axis is still unlabeled, so operators cannot tell which days the plotted points represent.
  ### Deliverables
  - Add visible time labels to the shared trend chart renderer.
  - Keep selected-site and all-sites traffic trend charts visually consistent.
  - Add black-box dashboard coverage proving rendered trend charts expose the X-axis time labels.
  ### Resolution
  Added first/middle/last date tick marks and UTC-formatted date labels to the shared traffic trend SVG renderer used by selected-site and all-sites traffic charts. Updated dashboard traffic browser coverage to assert the visible X-axis labels for the selected-site 7-day trend and the all-sites 30-day report trend. `make ci` passed with 384 integration tests.
  ### Changed Files
  `web/app/index.html`, `tests/specs/dashboard-traffic.spec.js`, `.mprlab/ISSUES.md`.
- [x] [I016] (P1) Show count scales on traffic trend charts.
  ### Summary
  Traffic trend charts currently show only line shape, so operators cannot read the visit-count scale from the graph itself.
  ### Deliverables
  - Add visible count-scale labels and grid references to the shared trend chart renderer.
  - Keep selected-site and all-sites traffic trend charts visually consistent.
  - Add black-box dashboard coverage proving the rendered charts expose the count scale.
  ### Resolution
  Added visible y-axis count labels, grid lines, and the `Visits / visitors` unit label to the shared trend chart SVG renderer used by selected-site and all-sites traffic views. Updated black-box dashboard traffic coverage to assert the rendered SVG exposes the scale labels through the SVG text nodes. Focused `dashboard-traffic.spec.js` coverage and full `make ci` passed after the SVG text assertion fix.
  ### Changed Files
  `web/app/index.html`, `tests/specs/dashboard-traffic.spec.js`, `.mprlab/ISSUES.md`.
- [x] [I017] (P1) Add graphical and portfolio traffic reporting.
  ### Summary
  Traffic reporting currently exposes numeric per-site metrics. Operators need visual trend/breakdown graphics and an all-sites report that summarizes the sites they own.
  ### Deliverables
  - Add dashboard graphics for selected-site traffic trends, attribution, engagement, device, and timezone metrics using existing traffic reporting APIs.
  - Add an authenticated all-sites traffic reporting API scoped to the current user's owned/created sites.
  - Add an all-sites dashboard reporting mode that summarizes portfolio totals, trend, top pages, and per-site rows.
  - Add portfolio traffic report scheduling and test-report endpoints without overloading per-site schedules.
  - Add black-box API and dashboard coverage for the new reporting surfaces.
  ### Resolution
  Added native dashboard graphics for selected-site traffic trends, top pages, attribution, engagement, device, and timezone reporting. Added an authenticated all-sites traffic report API scoped to the current user's owned/created sites, portfolio dashboard mode, portfolio report scheduling, and portfolio test-report delivery. Added black-box API and browser coverage for the graphical and portfolio reporting surfaces, and verified `make ci` passes.
  ### Changed Files
  `cmd/server/main.go`, `cmd/server/routes.go`, `internal/api/portfolio_traffic_report.go`, `internal/api/templates/portfolio_traffic_report_email.txt`, `internal/api/traffic_report_schedule.go`, `internal/model/traffic_report_schedule.go`, `internal/storage/database.go`, `tests/helpers/api.js`, `tests/specs/api-admin.spec.js`, `tests/specs/dashboard-traffic.spec.js`, `web/app/index.html`.
- [x] [I018] (P1) Move all-sites traffic reporting entry into settings.
  ### Summary
  The Traffic tab is scoped to the selected site, so placing an all-sites report toggle inside that tab makes the reporting scope ambiguous.
  ### Deliverables
  - Remove the selected-site/all-sites scope selector from the Traffic report card.
  - Add a Settings entry point that opens all-sites traffic reporting as a distinct global surface.
  - Keep direct Traffic tab navigation scoped to the selected site.
  - Rename portfolio-facing copy from properties to sites.
  - Add black-box dashboard coverage for the new entry point.
  ### Resolution
  Removed the selected-site/all-sites scope selector from the Traffic report card. Added a Reports section in Account Settings with an All sites traffic entry point that opens the portfolio reporting surface, while direct Traffic tab navigation resets to selected-site reporting. Updated portfolio UI and email copy to use sites instead of properties, added black-box dashboard coverage for the Settings entry point and selected-site return path, and verified `make ci` passes.
  ### Changed Files
  `web/app/index.html`, `internal/api/templates/portfolio_traffic_report_email.txt`, `internal/model/traffic_report_schedule.go`, `tests/specs/api-admin.spec.js`, `tests/specs/dashboard-elements.spec.js`, `tests/specs/dashboard-labels.spec.js`, `tests/specs/dashboard-traffic.spec.js`, `.mprlab/ISSUES.md`.
- [x] [I019] (P1) Split all-sites traffic into a separate dashboard view.
  ### Summary
  The all-sites traffic surface is still rendered inside the selected-site dashboard column, so the user can see portfolio metrics while the page still appears to be scoped to one selected site.
  ### Deliverables
  - Add a distinct all-sites traffic screen outside the selected-site account/site layout.
  - Keep the selected-site Traffic tab focused only on one site's widget, report schedule, and analytics.
  - Move all-sites report scheduling, totals, trend, top pages, and site table into the separate screen.
  - Preserve the Settings entry point and add a clear return path to the site dashboard.
  - Add black-box dashboard coverage proving the selected-site workspace is hidden in the all-sites screen.
  ### Resolution
  Added a separate all-sites traffic dashboard view that hides the selected-site account/site workspace. Moved all-sites report scheduling, totals, trend chart, top pages, and per-site table into that screen with an independent back path to the selected-site Traffic tab. Removed the all-sites table and portfolio mode from the selected-site Traffic cards. Added dashboard coverage for the Settings entry point, hidden site workspace, all-sites schedule autosave, and return path. `make ci` passed.
  ### Changed Files
  `web/app/index.html`, `tests/specs/dashboard-traffic.spec.js`, `tests/specs/dashboard-elements.spec.js`, `tests/specs/dashboard-labels.spec.js`, `.mprlab/ISSUES.md`.
- [x] [I020] (P1) Introduce global report-library visual language.
  ### Summary
  The global reporting page needs a durable visual model for multiple reports. Operators should be able to select or create report definitions and choose which sites belong to each report, without changing the selected-site Traffic tab.
  ### Deliverables
  - Rework the global reporting page into a report library with a saved-report list and selected-report detail.
  - Add create-report controls on the global page.
  - Add included-sites selection controls on the global page.
  - Keep the selected-site Traffic tab unchanged and isolated from global reporting state.
  - Add dashboard coverage for creating/selecting multiple global reports and changing included sites.
  ### Resolution
  Reworked the global all-sites traffic screen into a report-library surface with a saved-report rail, selected-report detail, editable custom report names, and included-site selection controls. Kept the selected-site Traffic tab isolated from global reporting state. The default all-sites report remains tied to the existing portfolio schedule, while custom report definitions can scope the preview table to selected sites. Added dashboard coverage for creating a custom report, renaming it, changing included sites, preserving the default all-sites schedule behavior, and returning to the selected-site Traffic tab. `make ci` passed.
  ### Changed Files
  `web/app/index.html`, `tests/specs/dashboard-traffic.spec.js`, `tests/specs/dashboard-elements.spec.js`, `tests/specs/dashboard-labels.spec.js`, `.mprlab/ISSUES.md`.
- [x] [I021] (P1) Persist scoped global traffic report definitions.
  ### Summary
  The global reporting page can create and edit report definitions visually, but custom reports are still local UI state. Operators need saved report definitions whose included sites drive portfolio previews and scheduled/test report delivery.
  ### Deliverables
  - Add persisted custom global traffic report definitions scoped to the current user.
  - Add persisted included-site membership for each custom report.
  - Add authenticated APIs to list, create, and update custom global report definitions.
  - Scope portfolio report data, schedules, and test report delivery by report definition.
  - Keep the implicit all-sites report available as the default global report.
  - Add API and dashboard coverage for persistence, site scoping, and report-specific schedules.
  ### Resolution
  Added persisted portfolio traffic report definitions and included-site membership. Added authenticated list/create/update APIs, scoped portfolio report previews by `report_id`, and made portfolio schedules and test-report delivery report-specific while keeping the implicit all-sites report as the default. Rewired the global reporting page to load and save report definitions through the API instead of local storage. Added black-box API and dashboard coverage for persistence, scoped site membership, and separate default/custom schedules. `make ci` passed.
  ### Changed Files
  `cmd/server/main.go`, `cmd/server/routes.go`, `internal/api/portfolio_traffic_report.go`, `internal/api/templates/portfolio_traffic_report_email.txt`, `internal/api/traffic_report_schedule.go`, `internal/model/portfolio_traffic_report.go`, `internal/model/traffic_report_schedule.go`, `internal/storage/database.go`, `tests/helpers/api.js`, `tests/specs/api-admin.spec.js`, `tests/specs/dashboard-traffic.spec.js`, `web/app/index.html`, `.mprlab/ISSUES.md`.
- [x] [I022] (P1) Move the account card into Account Settings.
  ### Summary
  The dashboard side column should stay focused on site selection. Account identity belongs in the existing Account Settings modal.
  ### Deliverables
  - Move the avatar, email, and role card into Account Settings.
  - Remove the account card from the dashboard side column so only Sites remains there.
  - Preserve the existing account hydration path from `/api/me`.
  - Add black-box dashboard coverage for the new placement.
  ### Resolution
  Moved the account card into Account Settings while preserving the existing `/api/me` hydration IDs. Removed the account card from the dashboard side column so the side column contains only Sites. Updated dashboard readiness helpers for hidden modal account fields, added black-box coverage for the modal account card and side-column layout, and verified `make ci` passes.
  ### Changed Files
  `web/app/index.html`, `tests/helpers/fixtures.js`, `tests/specs/dashboard-labels.spec.js`, `tests/specs/dashboard-layout.spec.js`, `tests/specs/dashboard-traffic.spec.js`, `tests/specs/dashboard-user-menu.spec.js`, `tests/specs/logout-hardening.spec.js`, `README.md`, `.mprlab/ISSUES.md`.
- [ ] [I023] (P1) Consider a design of a current accordion design of different surfaces.
  We may want to have a better split out.
- [x] [I024] (P1) Keep production deploy revision selection automatic.
  ### Summary
  The deploy flow should not ask operators to name or select a revision for Pages/backend deployment. The release workflow owns tagging; deploy consumes the release tag at repository `HEAD`.
  ### Resolution
  Removed the deploy wrapper's manual `--tag` option and `DEPLOY_TAG` override. `make deploy` now derives the v* release tag from `HEAD` when Pages or image verification needs it, and otherwise tells the operator to run the release flow before deploy. Gateway Ansible owns Pages dispatch from the app manifest. Validation passed with `bash -n scripts/deploy.sh`, the deploy no-op dry run, `git diff --check`, and `timeout -k 1200s -s SIGKILL 1200s make ci`.
- [x] [I025] (P1) Advertise LA Sentry on the public landing page.
  ### Summary
  The public landing page currently presents feedback, subscriber capture, and traffic analytics, but omits LA Sentry even though it is now a first-class developer monitoring surface.
  ### Deliverables
  - Update landing-page metadata and visible copy to include LA Sentry developer error monitoring.
  - Add LA Sentry as a first-class feature card alongside the other embeddable surfaces.
  - Update public-page tests that assert landing-page copy.
  ### Resolution
  Updated `/login` landing metadata, hero copy, feature grid, and setup copy to advertise LA Sentry as a first-class developer monitoring surface; updated public-page and auth-state tests; verified `make ci` passes.
- [x] [I026] (P1) Consolidate LA Sentry client discovery under `clients/`.
  ### Summary
  Make the first-party LA Sentry clients discoverable from a dedicated `clients/` entrypoint instead of requiring readers to know that Go, Python, and browser surfaces live in different runtime-oriented folders.
  ### Deliverables
  - Add a client index under `clients/` for Go, Python, and browser usage.
  - Move the Go client implementation under `clients/` so client-facing SDKs live outside the server package namespace.
  - Document why the browser harness remains served from `web/la-sentry.js`.
  - Update repo docs and integration fixtures to prefer the dedicated client locations.
  ### Resolution
  Added `clients/README.md` as the LA Sentry client index, moved the Go client implementation to `clients/go/lasentry`, removed the legacy `pkg/lasentry` package so SDKs are exposed only from `clients/`, added browser and Go client docs under `clients/`, updated README references and the Go integration fixture, and verified `make ci` passes.
- [ ] [I027] (P1) Replace placeholder-only inputs with labeled fields in the static frontend.
  Added `clients/README.md` as the LA Sentry client index, moved the Go client implementation to `clients/go/lasentry` with a `pkg/lasentry` compatibility package, added browser and Go client docs under `clients/`, updated README references and the Go integration fixture, and verified `make ci` passes.
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
- [x] [I029] (P1) Add distinct SEO resource pages for supported LoopAware workflows.
  ### Summary
  The current public resource cluster covers core product surfaces but omits supported workflows that have distinct setup, objections, examples, and conversion paths: mobile feedback, uptime monitoring, traffic report emails, and server-side error capture.
  ### Deliverables
  - Add crawlable resource pages for the four supported workflows without unsupported proof claims or doorway variants.
  - Link the new pages from `/resources` and related resource pages where relevant.
  - Add the new canonical URLs to `web/sitemap.xml`.
  - Update SEO integration coverage for the expanded resource cluster.
  ### Resolution
  Added four repo-grounded resource pages for mobile app feedback, uptime monitoring, traffic report emails, and server-side error capture. Updated `/resources` metadata, structured data, and cards, added internal links from adjacent resource pages, extended `web/sitemap.xml`, and expanded SEO public-page coverage to verify every canonical resource URL. Validation passed with `make lint` and `make ci` with 444 Playwright/API integration specs.
  ### Changed Files
  `PLAN.md`, `.mprlab/ISSUES.md`, `tests/specs/seo-public-pages.spec.js`, `web/resources/index.html`, `web/resources/mobile-app-feedback/index.html`, `web/resources/uptime-monitoring/index.html`, `web/resources/traffic-report-emails/index.html`, `web/resources/server-side-error-capture/index.html`, `web/resources/agency-client-sites/index.html`, `web/resources/feedback-widget/index.html`, `web/resources/la-sentry-monitoring/index.html`, `web/resources/lightweight-analytics/index.html`, `web/resources/privacy-first-analytics/index.html`, `web/resources/saas-feedback/index.html`, `web/resources/self-hosted-feedback/index.html`, `web/sitemap.xml`.
- [x] [I030] (P1) Expand the SEO resource cluster with deeper supported use-case pages.
  ### Summary
  The resource cluster can cover more supported LoopAware workflows without creating doorway pages by drilling into concrete setup decisions, operator workflows, reporting views, team access, uptime safety, and LA Sentry triage surfaces that already exist in the product.
  ### Deliverables
  - Add at least 20 crawlable resource pages for distinct supported workflows, not keyword-only variants.
  - Link the new pages from `/resources` and adjacent pages through useful internal paths.
  - Add the new canonical URLs to `web/sitemap.xml`.
  - Update SEO integration coverage for the expanded resource cluster.
  ### Resolution
  Added 20 repo-grounded resource pages covering multi-origin feedback, widget appearance, sentiment-only feedback, source context, subscriber confirmation and export, inline subscribe forms, traffic CSV export, top pages, attribution, device and location reporting, bot-filtered analytics, no-JavaScript traffic fallback, portfolio reports, team access, owner/admin site management, favicon refresh notifications, browser error capture, and LA Sentry issue triage. Updated the `/resources` index metadata, structured data, and cards, added all new canonical URLs to `web/sitemap.xml`, and expanded SEO public-page coverage. Validation passed with `git diff --check`, `make lint`, and `make ci` with 444 Playwright/API integration specs. Review follow-up corrected the no-JavaScript traffic pixel example to use the real absolute `/public/visits` endpoint instead of a nonexistent root-relative `/public/visits/pixel` path, with SEO coverage locking that contract. A second review follow-up corrected the inline subscribe forms example to use the hosted `https://loopaware.mprlab.com/subscribe.js` script instead of a root-relative customer-origin script, with SEO coverage for that contract.
  ### Changed Files
  `PLAN.md`, `.mprlab/ISSUES.md`, `tests/specs/seo-public-pages.spec.js`, `web/resources/index.html`, `web/sitemap.xml`, `web/resources/multi-origin-feedback/index.html`, `web/resources/widget-appearance-controls/index.html`, `web/resources/sentiment-only-feedback/index.html`, `web/resources/feedback-source-context/index.html`, `web/resources/subscriber-confirmation-flow/index.html`, `web/resources/subscriber-csv-export/index.html`, `web/resources/inline-subscribe-forms/index.html`, `web/resources/traffic-csv-export/index.html`, `web/resources/top-pages-reporting/index.html`, `web/resources/traffic-attribution-breakdown/index.html`, `web/resources/visitor-device-breakdown/index.html`, `web/resources/visitor-location-signals/index.html`, `web/resources/bot-filtered-analytics/index.html`, `web/resources/no-javascript-traffic-pixel/index.html`, `web/resources/portfolio-traffic-reports/index.html`, `web/resources/team-member-site-access/index.html`, `web/resources/owner-admin-site-management/index.html`, `web/resources/favicon-refresh-notifications/index.html`, `web/resources/browser-error-capture/index.html`, `web/resources/sentry-issue-triage/index.html`.
- [x] [I031] (P1) Make resource pages part of the public site shell.
  ### Summary
  The `/resources` page cluster uses a separate hand-rolled header, footer, typography, and color system, so it reads as an appended microsite rather than a first-class LoopAware public page.
  ### Deliverables
  - Use the same shared public `mpr-header` and `mpr-footer` shell as `/login`, `/pricing`, `/privacy`, and `/terms`.
  - Keep resource pages inside the same dark/light public theme and flex header/content/footer layout.
  - Preserve crawlable resource content, canonical metadata, and internal resource links.
  - Add black-box coverage proving `/resources` participates in the public auth/chrome surface without direct Google Identity loading.
  ### Resolution
  Rebuilt all 33 `/resources` pages on the shared public page shell with `mpr-header`, `mpr-footer`, MPR UI assets, Bootstrap runtime, the LoopAware public favicon, shared public auth handling, and a resource theme bridge that keeps dark/light theme state aligned with the public landing pages. Restyled the resource cluster with the LoopAware public dark/light palette, static footer rendering for long resource pages, responsive header/footer overrides, and mobile overflow fixes while preserving all crawlable resource content, canonical metadata, structured data, and internal links. Added public-page coverage so `/resources` now participates in auth redirect, CDN asset, Google Identity delegation, brand-link, footer utility, and SEO shell checks. Follow-up removed all added public header utility links, keeping Resources only as a footer utility before Pricing and leaving the public header to brand plus auth controls. Validation passed with `git diff --check`, `make lint`, focused compose-backed `seo-public-pages` and `header-auth-state` Playwright specs with 85 passing tests, compose-backed visual/header metrics confirming `390px` mobile `scrollWidth`, static rendered footer, 33 resource cards, no header utility links, and footer order `Terms of Service`, `Resources`, `Pricing`, plus final `make ci` with 454 Playwright/API integration specs.
  ### Changed Files
  `PLAN.md`, `.mprlab/ISSUES.md`, `tests/specs/header-auth-state.spec.js`, `tests/specs/seo-public-pages.spec.js`, `web/login/index.html`, `web/pricing/index.html`, `web/privacy/index.html`, `web/terms/index.html`, `web/resources/styles.css`, `web/resources/public-theme.js`, `web/resources/agency-client-sites/index.html`, `web/resources/bot-filtered-analytics/index.html`, `web/resources/browser-error-capture/index.html`, `web/resources/favicon-refresh-notifications/index.html`, `web/resources/feedback-source-context/index.html`, `web/resources/feedback-widget/index.html`, `web/resources/index.html`, `web/resources/inline-subscribe-forms/index.html`, `web/resources/la-sentry-monitoring/index.html`, `web/resources/lightweight-analytics/index.html`, `web/resources/mobile-app-feedback/index.html`, `web/resources/multi-origin-feedback/index.html`, `web/resources/no-javascript-traffic-pixel/index.html`, `web/resources/owner-admin-site-management/index.html`, `web/resources/portfolio-traffic-reports/index.html`, `web/resources/privacy-first-analytics/index.html`, `web/resources/saas-feedback/index.html`, `web/resources/self-hosted-feedback/index.html`, `web/resources/sentiment-only-feedback/index.html`, `web/resources/sentry-issue-triage/index.html`, `web/resources/server-side-error-capture/index.html`, `web/resources/subscriber-capture/index.html`, `web/resources/subscriber-confirmation-flow/index.html`, `web/resources/subscriber-csv-export/index.html`, `web/resources/team-member-site-access/index.html`, `web/resources/top-pages-reporting/index.html`, `web/resources/traffic-attribution-breakdown/index.html`, `web/resources/traffic-csv-export/index.html`, `web/resources/traffic-report-emails/index.html`, `web/resources/uptime-monitoring/index.html`, `web/resources/visitor-device-breakdown/index.html`, `web/resources/visitor-location-signals/index.html`, `web/resources/widget-appearance-controls/index.html`.
- [x] [I032] (P1) Improve Google indexing readiness for resource pages.
  ### Summary
  The public resource cluster needs stronger indexing signals after the pages were added: canonical sitemap URLs should match served trailing-slash URLs, the root page should be indexable, each resource page should have deeper visible FAQ/checklist content with matching structured data, and the main site plus repo docs should expose the resource hub.
  ### Deliverables
  - Align canonical, sitemap, Open Graph, JSON-LD, and internal links to the current public URL contract.
  - Make `/` an indexable public landing page while keeping `/login` available and canonicalized to root.
  - Add visible FAQ/checklist depth and Article, BreadcrumbList, and FAQPage schema to resource pages.
  - Link the resource hub from the public landing page and product documentation.
  - Update black-box SEO coverage for the expanded indexing contract.
  ### Resolution
  Standardized public SEO URLs on trailing-slash canonical forms for `/`, `/pricing/`, `/resources/`, and all resource detail pages, removed `/login` from `sitemap.xml`, and kept `/login` root-canonical for authentication flows. Replaced the root redirect/noindex page with the public landing page, added first-screen and CTA links to `/resources/`, and updated all affected public footer/resource links to root-relative canonical URLs. Added visible checklist and FAQ depth to every resource detail page, added hub guidance to `/resources/`, and extended resource JSON-LD with `datePublished`, `dateModified`, `author`, `publisher`, `BreadcrumbList`, and visible-content `FAQPage` schema. Updated README, PRD, and architecture docs to reference the canonical resource hub. Validation passed with JSON-LD parse checks, `git diff --check`, focused static SEO Playwright coverage, and final `make ci` with 454 Playwright/API integration specs.
  ### Changed Files
  `PLAN.md`, `.mprlab/ISSUES.md`, `ARCHITECTURE.md`, `PRD.md`, `README.md`, `tests/specs/seo-public-pages.spec.js`, public HTML pages under `web/`, all resource detail pages under `web/resources/`, `web/resources/index.html`, `web/resources/styles.css`, `web/robots.txt`, `web/sitemap.xml`.

- [x] [I033] (P1) Embed traffic pixel script in LoopAware core pages.
  ### Summary
  The LoopAware website does not load the traffic pixel (`pixel.js`), resulting in 0 tracked visits on its own dashboard.
  ### Deliverables
  - Add the traffic pixel script `<script defer src="https://loopaware.mprlab.com/pixel.js?site_id=a3222433-92ec-473a-9255-0797226c2273"></script>` to LoopAware's core HTML pages: `web/index.html`, `web/login/index.html`, `web/pricing/index.html`, `web/privacy/index.html`, `web/terms/index.html`, and `web/app/index.html`.
  - Fix pre-existing Go lint issues in the repository.
  ### Resolution
  Manually embedded the canonical traffic pixel script tag on all core HTML pages of the static frontend. Fixed pre-existing TypeScript index signature typing errors in the mobile scripts (`publish-android-play.mjs` and `submit-ios.mjs`). Verified all checks pass cleanly with `make lint` and the unit test suite.
  ### Changed Files
  `web/index.html`, `web/login/index.html`, `web/pricing/index.html`, `web/privacy/index.html`, `web/terms/index.html`, `web/app/index.html`, `mobile/scripts/publish-android-play.mjs`, `mobile/scripts/submit-ios.mjs`.

## Maintenance

### Recurring

- [ ] [M400R] (P2) Backlog hygiene and archive
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

- [ ] [M401R] (P2) Polish open issues
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

- [ ] [M402R] (P2) Architecture and policy review
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

- [ ] [M403R] (P1) Dependency and security audit
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

- [ ] [M404R] (P1) CI, release, and artifact health
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

- [ ] [M405R] (P1) Code contract and static hygiene
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

- [ ] [M406R] (P1) Production drift and health
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

- [ ] [M407R] (P2) Documentation and runbook hygiene
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
- [x] [F009] (P1) Add Expo-compatible mobile feedback widget support.
  ### Summary
  Mobile apps need a native feedback button equivalent to the web feedback widget, with screen and app context attached to each submission. This is separate from LA Sentry error capture and should not require native code for the first version.
  ### Deliverables
  - Add a public mobile feedback submission contract that validates a registered mobile app for the selected site.
  - Persist and return mobile source, screen, app metadata, and bounded context with feedback messages.
  - Display mobile feedback context in the dashboard Feedback tab without changing web widget behavior.
  - Add an Expo-compatible React Native feedback client with explicit screen/context props.
  - Document setup and add black-box API/dashboard coverage.
  ### Resolution
  Added registered mobile app records per site, authenticated mobile-app list/create endpoints, and `/public/mobile-feedback` for native app feedback submissions with registered client validation. Feedback records now preserve source kind, mobile client ID, screen, app metadata, and bounded JSON context, and the dashboard renders/searches mobile metadata inside the Feedback table without changing web widget submissions. Added an Expo-compatible React Native feedback button/provider client plus README/client docs. Validation passed with baseline `make ci`, focused `go test ./internal/api ./internal/model ./internal/storage`, `npm --prefix tests run typecheck`, `git diff --check`, and Docker-backed `LOOPAWARE_TEST_SUITE=test:all ./tests/scripts/run-integration.sh` with 409 specs.
  Post-review hardening rejects browser-origin mobile feedback submissions before public app-client validation and wires the React Native client package into the Makefile-backed TypeScript check.
  ### Changed Files
  `PLAN.md`, `.mprlab/ISSUES.md`, `Makefile`, `README.md`, `clients/README.md`, `clients/react-native/package.json`, `clients/react-native/README.md`, `clients/react-native/src/index.tsx`, `clients/react-native/tsconfig.json`, `clients/react-native/types/react.d.ts`, `clients/react-native/types/react-native.d.ts`, `cmd/server/main.go`, `cmd/server/routes.go`, `internal/api/admin.go`, `internal/api/public.go`, `internal/model/models.go`, `internal/model/mobile_feedback.go`, `internal/storage/database.go`, `internal/storage/migrations.go`, `tests/helpers/api.js`, `tests/specs/api-public.spec.js`, `tests/specs/dashboard-feedback.spec.js`, `web/app/index.html`.
- [x] [F010] (P1) Add per-site team members.
  ### Summary
  Site admins need to add individual Google-authenticated email addresses to one site so those teammates can see that site's existing dashboard data after login.
  ### Product Decisions
  - Keep team membership attached directly to a site; do not introduce reusable teams or multi-site team management.
  - Treat existing site owners, creators, and global admins as site admins for this narrow feature.
  - Store normalized email addresses; Google authentication remains the only login path.
  - Team members receive read-only site access and do not manage site settings, team membership, tokens, or schedules.
  ### Deliverables
  - Per-site team member model and migration.
  - Authenticated API routes for site admins to list, add, and remove team members.
  - Membership-aware site listing and read-only site data access.
  - Dashboard controls for adding/removing team emails and read-only behavior when a team member views an assigned site.
  - Black-box API/dashboard coverage.
  ### Resolution
  Added direct per-site team member assignments keyed by normalized email, migrated the new table, and made site listing plus read-only site data endpoints include matching team-member access. Site admins can list, add, and remove team emails through `/api/sites/:id/team`, while team members cannot mutate site settings, membership, subscribers, Sentry state, tokens, or traffic report schedules. The dashboard includes an Admin section for site managers and hides those team management controls when the selected site has `access_role: "team_member"`. README documents the new role contract and API endpoints. Validation passed with `go test ./internal/model ./internal/storage ./internal/api ./cmd/server`, `npm --prefix tests run typecheck`, `make test-integration-api`, and final `make ci` with 418 Playwright/API integration specs.
  Post-review fix restored the missing-subscriber delete response to `unknown_subscription` and added API coverage for that contract. Final validation passed with `make test-integration-api` and `make ci` with 419 Playwright/API integration specs.
  UI refinement moved team-member management into an Admin dashboard section that is visible only for sites the current user can manage; assigned read-only team members do not see the Admin section for those sites. Validation passed with `make test-integration` and final `make ci`, both with 418 Playwright/API integration specs.
  Traffic report scheduling now supports manager-only, whole-team, and selected-team-member recipients for selected-site reports. Selected recipients are validated against current per-site team membership, whole-team sends resolve the owner/creator plus current team members at delivery time, and the dashboard Traffic card exposes the recipient selector for site managers. Validation passed with `go test ./internal/model ./internal/api`, `make lint-js`, `make test-integration-api`, `make test-integration`, and final `make ci` with 423 Playwright/API integration specs.
  Review follow-up made scheduled multi-recipient report delivery non-retryable after partial success so already-delivered recipients are not emailed again, while test-report sends still surface partial failures. Favicon and feedback SSE streams now use the same team-member view-access filter as the read APIs. Validation passed with focused `go test ./internal/api -run 'TestTrafficReportSchedulerDoesNotRetryPartialTeamDelivery|TestTrafficReportSchedulerRecordsSendFailure|TestStreamFaviconUpdatesAllowsTeamMemberSite|TestStreamFeedbackUpdatesAllowsTeamMemberSite|TestStreamFaviconUpdatesSkipsUnauthorizedSite|TestStreamFeedbackUpdatesSkipsUnauthorizedSite'` and final `make ci` with 423 Playwright/API integration specs.
  Review follow-up rejects RFC 5322 display-name forms for site team-member emails so invites store only the Google login addr-spec that access checks compare. Focused validation passed with `make test-unit` and `make test-integration-api`.
  ### Changed Files
  `PLAN.md`, `.mprlab/ISSUES.md`, `README.md`, `cmd/server/main.go`, `cmd/server/routes.go`, `internal/api/admin.go`, `internal/api/admin_helpers_test.go`, `internal/api/portfolio_traffic_report.go`, `internal/api/sentry.go`, `internal/api/site_access.go`, `internal/api/traffic_report_schedule.go`, `internal/api/traffic_report_schedule_test.go`, `internal/model/site_team.go`, `internal/model/traffic_report_schedule.go`, `internal/model/traffic_report_schedule_test.go`, `internal/storage/database.go`, `tests/specs/api-admin.spec.js`, `tests/specs/dashboard-elements.spec.js`, `tests/specs/dashboard-labels.spec.js`, `tests/specs/dashboard-site-actions.spec.js`, `tests/specs/dashboard-traffic.spec.js`, `web/app/index.html`.
- [x] [F011] (P1) Add a native operator mobile app.
  ### Summary
  Operators need a downloadable iOS and Android client that logs in with the same Google/TAuth identity and shows the same sites, stats, feedback, subscribers, LA Sentry issues, traffic reports, and role-aware data available in the web dashboard.
  ### Deliverables
  - Replace the default Expo scaffold in `mobile/` with a LoopAware operator app.
  - Use TAuth native Google sign-in with system-browser PKCE and standard TAuth cookies; do not add mobile bearer-token auth to the backend.
  - Load authenticated data from the existing `/api` endpoints with the same site-access semantics as the web dashboard.
  - Add iOS and Android bundle/package configuration, custom scheme, and EAS build profiles.
  - Add Makefile targets for mobile validation, local iOS/Android runs, and EAS builds.
  - Add CI/config validation that fails when the mobile app is out of the repo-native build contract.
  - Document native TAuth client placeholders in the tracked TAuth env templates.
  ### Resolution
  Replaced the default Expo scaffold with a LoopAware operator app that signs in through TAuth native Google AuthSession/PKCE, reuses standard TAuth cookies, and reads the existing authenticated dashboard API surface for account data, visible sites, feedback, subscribers, traffic stats/trends/attribution/engagement/devices/locations, LA Sentry issues, mobile feedback app registrations, team metadata for site admins, selected-site traffic schedules, and all-sites traffic reporting. Added environment-aware Expo config for iOS and Android package IDs, custom schemes, EAS build profiles, native TAuth client placeholders, `mobile-check`, `run-ios`, `run-android`, `build-ios`, and `build-android` Make targets. CI now triggers/caches mobile changes and `make ci` runs mobile config validation and TypeScript checks. Validation passed with `make mobile-check`, `make config-audit`, `npx expo config --json`, and final `make ci` with 423 Playwright/API integration specs.
  Review follow-up registers the iOS Google redirect scheme from the configured native redirect URI, adds EAS/profile validation for that required build-time scheme, restores the missing native Google placeholders to tracked TAuth env templates so config audit can validate every compose target, keeps the mobile dashboard signed in when server logout fails, and guards selected-site/interval dashboard loads against stale responses. Validation passed with `make config-audit`, `make mobile-check`, `git diff --check`, an Expo config probe for the TAuth-style iOS redirect URI, and final `make ci`.
  Native OAuth setup follow-up applies the real iOS build-time redirect URI from the LoopAware iOS Google client plist to the EAS profiles, while the ignored local TAuth env carries the matching iOS client ID and redirect URI. Validation passed with `make config-audit`, `make mobile-check`, `git diff --check`, an Expo config probe showing the registered iOS scheme, and final `make ci`.
  Final review fix restores the tracked TAuth native Google placeholders and exports the configured iOS Google redirect URI through `make run-ios` so local dev-client builds register the same reversed-client scheme as EAS builds. Validation passed with `make mobile-check`, `make config-audit`, `git diff --check`, and Expo config probes with and without the local Makefile redirect environment.
  Local iOS run follow-up forces the Expo dev-client launch URL through `EXPO_PACKAGER_PROXY_URL=http://localhost:<metro_port>` so the simulator opens the same localhost-only Metro server that `expo start --localhost` serves instead of landing on the empty development-server launcher. Local validation confirmed Expo now emits `exp+loopaware-mobile://...url=http://localhost:8081` while `localhost:8081/status` is running and `127.0.0.1:8081/status` remains unreachable on this machine. Validation passed with `make mobile-check`, `make config-audit`, `git diff --check`, a local `make run-ios` deep-link/status probe, and final `make ci` with 425 Playwright/API integration specs.
  Native polish follow-up replaces the Expo scaffold icon with the canonical LoopAware eye mark across mobile assets, updates Expo to `~56.0.12`, pins a safe `uuid` override for a clean mobile audit, removes default Metro cache clearing, unsets inherited `NO_COLOR` for mobile and integration-test npm commands, applies repeatable generated-iOS warning fixes before local dev builds, and fingerprints native-affecting inputs so stale installed dev builds are rebuilt automatically. The signed-out restore path now treats missing TAuth refresh sessions as signed out and native TAuth misconfiguration surfaces as a readable sign-in error instead of a generic 404. Local validation confirmed `make run-ios` rebuilds and starts with `0 warning(s)`, a `localhost:8081` dev-client URL, the real logo, and no startup error. Final validation passed with `make mobile-check`, `make config-audit`, `npm --prefix mobile audit --audit-level=moderate`, `npx expo install --check`, `git diff --check`, and `make ci` with 425 Playwright/API integration specs.
  Android run follow-up makes `make run-android` self-heal malformed ignored Android native project directories by preparing native Android before `expo run:android`, moves `local.properties` creation into that preparation script, removes Expo SDK 56 Android config warnings by adding `expo-system-ui` and dropping `edgeToEdgeEnabled`, and restores native Google placeholders in tracked TAuth env examples so config audit passes. Validation passed with `make config-audit`, `npm --prefix mobile run validate-config`, `npm --prefix mobile run android:prepare-native`, `make mobile-check`, `npm --prefix mobile audit --audit-level=moderate`, `npx expo install --check`, `git diff --check`, and final `make ci` with 425 Playwright/API integration specs.
  Native fingerprint review follow-up includes build-time bundle/package and iOS Google redirect/client environment values in the local dev-build fingerprint, and passes Makefile bundle/package overrides into native build/start commands so installed dev clients rebuild when callback schemes or app identifiers change. Validation passed with env-sensitive fingerprint probes for iOS redirect, iOS bundle identifier, and Android package changes, plus `make mobile-check`, `make config-audit`, `git diff --check`, and final `make ci`.
  Config-audit CI follow-up adds the missing tracked default gHTTP env template, unignores tracked config env examples, and keeps native LoopAware Google placeholders present in both TAuth env templates so clean CI checkouts validate the same compose env set as local stacks. Validation passed with `go mod tidy && git diff --exit-code go.mod go.sum`, `make config-audit`, a clean-copy `make config-audit`, `make mobile-check`, `git diff --check`, and final `make ci`.
  ### Changed Files
  `PLAN.md`, `.mprlab/ISSUES.md`, `.gitignore`, `.github/workflows/ci.yml`, `Makefile`, `configs/README.md`, `configs/config.tauth.yml`, `configs/.env.ghttp.example`, `configs/.env.tauth.example`, `configs/.env.tauth.computercat.example`, `tests/configs/tauth.env`, `tests/scripts/run-integration.sh`, `mobile/AGENTS.md`, `mobile/App.tsx`, `mobile/LICENSE`, `mobile/app.config.js`, `mobile/eas.json`, `mobile/index.ts`, `mobile/package.json`, `mobile/package-lock.json`, `mobile/tsconfig.json`, `mobile/assets/android-icon-background.png`, `mobile/assets/android-icon-foreground.png`, `mobile/assets/android-icon-monochrome.png`, `mobile/assets/favicon.png`, `mobile/assets/icon.png`, `mobile/assets/splash-icon.png`, `mobile/scripts/fix-ios-project-warnings.mjs`, `mobile/scripts/native-build-fingerprint.mjs`, `mobile/scripts/prepare-android-project.mjs`, `mobile/scripts/resolve-metro-port.mjs`, `mobile/scripts/validate-mobile-config.mjs`, `mobile/src/api.ts`, `mobile/src/auth.ts`, `mobile/src/config.ts`, `mobile/src/format.ts`, `mobile/src/types.ts`.
- [x] [F012] (P1) Add per-site uptime health monitoring and outage notifications.
  ### Summary
  Site managers need LoopAware to notify them when a configured customer site is completely down and again when it recovers. The first version should use server-side synthetic HTTP checks owned by LoopAware, not customer-page heartbeat JavaScript, because a page heartbeat cannot report a fully unreachable site.
  ### Product Decisions
  - Add health monitoring as a first-class site feature rather than piggybacking on feedback, traffic, or LA Sentry.
  - Use server-side public HTTP checks for MVP; do not require an embedded heartbeat script.
  - Alert only on state transitions after a configured failure threshold so transient failures do not send repeated emails.
  - Treat HTTP responses below 500 as reachable and HTTP 5xx, DNS, connect, TLS, redirect, or timeout failures as down.
  - Resolve recipients through the existing site manager/team recipient modes used by scheduled traffic reports.
  ### Deliverables
  - Per-site health monitor model, current-state persistence, transition history, and deletion cleanup.
  - Public-target validating HTTP prober with bounded timeout, redirect target checks, deterministic error codes, and no private-network probing.
  - Background manager that runs due checks, updates monitor state, and sends down/recovered Pinguin notifications.
  - Authenticated APIs for reading, configuring, and manually running a site's health monitor with team-member read access and manager-only mutation.
  - Dashboard Health tab with status, settings, recipient controls, and manual check action.
  - Operator mobile app read-only health status for the selected site.
  - README documentation and black-box/focused coverage for the feature.
  ### Resolution
  Added first-class per-site health monitoring with a persisted current monitor, transition history, public-network-only HTTP probing, scheduled due checks, manual check execution, and down/recovered Pinguin alerts. Health targets are validated as public HTTP(S) URLs, redirects are revalidated during probing, HTTP responses below 500 count as reachable, and failures are classified with stable error codes for HTTP 5xx, DNS, TLS, redirect, timeout, network, and invalid-target cases.
  Added authenticated health APIs for reading, saving, and manually checking a site monitor, preserving team-member read access and manager-only mutation. Alert recipients now share the canonical site-recipient contract used by traffic reports: manager, whole team, or selected team members. Site deletion removes health monitors and transition events.
  Added the dashboard Health tab with status summary, enablement, target URL, interval, timeout, failure threshold, recipient controls, selected-member checkboxes, and manual check action. The operator mobile app now loads and displays read-only health status for the selected site.
  Documented the server-side synthetic-check design in README, including that the health monitor does not require a customer-site heartbeat script. Validation passed with focused backend tests, API integration, UI integration, mobile checks, and final `make ci` with 444 Playwright/API integration specs.
  Review follow-up makes failed down/recovered alert delivery retryable from the persisted monitor status versus `last_alerted_status`, so a transient Pinguin send failure no longer permanently suppresses the transition alert. Added handler-level regression coverage for down-alert and recovery-alert retry without duplicate transition events. Validation passed with baseline `make ci`, `make test-unit`, and final `make ci` with 444 Playwright/API integration specs.
  ### Changed Files
  `.mprlab/ISSUES.md`, `PLAN.md`, `README.md`, `cmd/server/main.go`, `cmd/server/routes.go`, `internal/api/admin.go`, `internal/api/origin_repro_test.go`, `internal/api/site_health_monitor.go`, `internal/api/site_health_monitor_test.go`, `internal/api/site_health_probe.go`, `internal/api/site_health_probe_test.go`, `internal/api/site_recipients.go`, `internal/api/traffic_report_schedule.go`, `internal/model/site_health_monitor.go`, `internal/model/site_health_monitor_test.go`, `internal/model/site_recipients.go`, `internal/model/traffic_report_schedule.go`, `internal/storage/database.go`, `mobile/App.tsx`, `mobile/scripts/test-api-boundaries.mjs`, `mobile/src/api.ts`, `mobile/src/types.ts`, `pkg/favicon/resolver.go`, `pkg/outbound/http.go`, `tests/specs/api-admin.spec.js`, `tests/specs/dashboard-elements.spec.js`, `tests/specs/dashboard-labels.spec.js`, `web/app/index.html`.
- [x] [F013] (P1) Integrate the shared runtime config package.
  ### Summary
  LoopAware still owns most backend runtime configuration through Viper, flags, and direct environment variable reads, while YAML only supplements the admin list. The backend runtime contract should move to one typed YAML config loaded through `github.com/tyemirov/utils/runtimeconfig`, with shell variables used only as interpolation inputs while parsing the selected YAML file.
  ### Deliverables
  - Replace Viper-backed runtime config reads with a typed YAML contract covering server address, public base URL, database, session secret, TAuth, Pinguin, notifications, and admins.
  - Keep only command structure and config-path selection in the server command; remove runtime config flags, the `ADMINS` override, and the `GRPC_AUTH_TOKEN` alias.
  - Update config templates, docs, config audit, and tests so YAML is the canonical backend runtime source and `web/config.yml` remains browser-facing only.
  - Verify valid interpolation, missing interpolation values, unknown YAML fields, missing required values, removed env overrides, and the real command/config path.
  ### Resolution
  Added `internal/serverconfig` as the LoopAware-specific typed YAML contract backed by `github.com/tyemirov/utils/runtimeconfig`, reduced `cmd/server` to `--config` selection plus runtime wiring, removed Viper-backed runtime reads and runtime config flags, and removed the `ADMINS` override and `GRPC_AUTH_TOKEN` alias from the backend path. Updated LoopAware config templates so backend runtime values live in YAML, added a test-specific mounted backend config, taught `config-audit` to resolve the mounted YAML with compose env placeholders, and updated docs for the new one-parse config boundary. Validation passed with focused server/config-audit tests, `make config-audit`, `go test ./... -count=1`, `make lint`, and final `make ci` with 444 Playwright/API integration specs.
  ### Changed Files
  `PLAN.md`, `.mprlab/ISSUES.md`, `ARCHITECTURE.md`, `CHANGELOG.md`, `README.md`, `cmd/configaudit/main.go`, `cmd/configaudit/main_additional_test.go`, `cmd/configaudit/main_test.go`, `cmd/server/main.go`, `cmd/server/main_test.go`, `configs/.env.loopaware.computercat.example`, `configs/.env.loopaware.example`, `configs/README.md`, `configs/config.loopaware.yml`, `go.mod`, `go.sum`, `internal/serverconfig/config.go`, `tests/configs/config.loopaware.yml`, `tests/configs/loopaware.env`, `tests/docker-compose.yml`, `tests/helpers/config.js`.
- [x] [F014] (P1) Publish operator mobile builds to Apple and Google.
  ### Summary
  The operator mobile app has build targets for iOS and Android, but LoopAware does not expose the store-submission targets needed to upload completed packages to App Store Connect/TestFlight and Google Play.
  ### Deliverables
  - Add Makefile targets for iOS, Android, and combined mobile package submission.
  - Build and publish from local iOS IPA and Android AAB artifacts instead of EAS Submit.
  - Keep Apple and Google credentials outside the repository while documenting the required operator inputs.
  - Extend mobile config validation so CI fails if the submission contract drifts.
  - Document the mobile publishing runbook alongside the existing release/publish/deploy flow.
  ### Resolution
  Replaced the EAS submission contract with standard local store publishing. iOS now builds a local Xcode App Store Connect IPA from the Expo prebuild output, writes a hash-checked IPA manifest, and uploads with `xcrun altool` using App Store Connect API-key inputs or an Apple ID app-specific password. Android writes a signed AAB build manifest with the R8 mapping sidecar and publishes through the Google Play Android Publisher API using Google Application Default Credentials. A follow-up replaced manual iOS build-number and Android version-code inputs with one UTC CalVer release timestamp: the native user-visible version is `YYYY.M.D`, while the internal iOS build number and Android `versionCode` are generated from seconds since `2020-01-01T00:00:00Z`. A Google Play follow-up mirrored the Kamu release identity contract by adding the tracked Android release identity and using its `loopaware` Google Cloud project as the default quota project. The release follow-up wires `make release` to upload both iOS and Android native builds after the GitHub Release verification step, with explicit skip flags for intentional partial releases. Review follow-up made Android AAB builds require the real upload keystore and verify its certificate against the tracked release identity, completed the required `altool --upload-package` metadata, and passed direct `make submit-ios` App Store Connect defaults into the submit script. Removed `mobile/eas.json`, updated the Makefile targets, extended mobile config validation to reject EAS submission drift and manual version-code drift, and documented the new Apple/Google operator inputs in README. Baseline validation passed with `make ci` before edits; final validation passed with focused mobile checks, CalVer submit-default probes, Android publish dry runs, the review follow-up `make mobile-check`, and final `make ci`.

  Current contract note: Superseded by B039 and B042. Release builds both signed store artifacts; publish preflights both providers before either upload and does not accept partial-release skip flags.
  ### Changed Files
  `.mprlab/ISSUES.md`, `PLAN.md`, `CHANGELOG.md`, `README.md`, `Makefile`, `scripts/release.sh`, `mobile/android-release-identity.json`, `mobile/app.config.js`, `mobile/scripts/build-android-bundle.mjs`, `mobile/scripts/build-ios-archive.mjs`, `mobile/scripts/mobile-calver-version.mjs`, `mobile/scripts/publish-android-play.mjs`, `mobile/scripts/submit-ios.mjs`, `mobile/scripts/validate-mobile-config.mjs`.


## Planning
*do not implement yet*
