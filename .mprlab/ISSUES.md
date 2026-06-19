# ISSUES

Entries record newly discovered requests or changes.

Read @AGENTS.md (Workflow section), @POLICY.md, and relevant stack guides before implementing changes.

Format: `- [ ] [B042] (P1) {I007} Title`

- `[ ]` open, `[!]` blocked, `[x]` closed.
- Blocked issues (`[!]`) must include a `Blocked:` line in the body.

## BugFixes

- [!] [B032] (P0) Unblock Android native Google sign-in in Google Cloud.
  ### Summary
  The Android operator app opens the Google OAuth browser flow but Google rejects the request before account consent.
  ### Evidence
  The installed dev build resolves `com.mprlab.loopaware:/oauth2redirect/google` to `com.mprlab.loopaware/.MainActivity`, TAuth production returns the LoopAware Android native client, and Chrome opens `https://accounts.google.com/o/oauth2/v2/auth` with `client_id=281540686395-7lt9u98ir3oincpqhdrjur5169qel8n9.apps.googleusercontent.com` plus `redirect_uri=com.mprlab.loopaware:/oauth2redirect/google`.
  The Google error details dialog reports `Error 400: invalid_request` and `Custom URI scheme is not enabled for your Android client.`
  ### Deliverables
  - Enable the matching Google Cloud Android OAuth client to accept the `com.mprlab.loopaware:/oauth2redirect/google` custom URI scheme, or replace the Android login implementation with Google's recommended Android identity flow.
  - Confirm the Android OAuth client is bound to package `com.mprlab.loopaware` and the active debug/release SHA-1 fingerprint before re-testing.
  - Re-run the Android dev-client login flow after Google Cloud propagation.
  Blocked: Google Cloud OAuth client settings are external to the LoopAware repo and TAuth deployment.

- [x] [B033] (P0) Keep empty API collections as arrays for mobile dashboard rendering.
  ### Summary
  The Android operator dashboard can hit a React render error after login because `/api/sites/:id/subscribers` returns `subscribers: null` for sites without subscribers, and the mobile dashboard treats the API collection contract as an array.
  ### Deliverables
  - Return `subscribers: []` from the site subscribers API when a site has no subscribers.
  - Add black-box API coverage that empty dashboard collection endpoints return arrays instead of `null`.
  - Verify focused API coverage and the full CI gate.
  ### Resolution
  Initialized the site subscribers API response slice so empty subscriber lists serialize as `[]` instead of `null`. Added a black-box API contract test that creates an empty site and asserts the mobile dashboard collection endpoints return arrays for `messages`, `subscribers`, `team_members`, `mobile_apps`, and Sentry `issues`. Validation passed with baseline `make ci`, focused `LOOPAWARE_TEST_SUITE=test:api ./tests/scripts/run-integration.sh`, and final `make ci` with 426 integration specs.
  ### Changed Files
  `PLAN.md`, `.mprlab/ISSUES.md`, `internal/api/admin.go`, `tests/specs/api-admin.spec.js`.

- [x] [B034] (P0) Prevent mobile dashboard render crashes from deployed null collections.
  ### Summary
  The Android dev client still crashes against the currently deployed API before the B033 backend fix is deployed because the mobile API client passes `null` collection payloads directly into React dashboard state.
  ### Deliverables
  - Normalize legacy deployed `null` empty collections at the mobile API boundary so React only receives arrays.
  - Reject non-array collection payloads with a controlled API error instead of a render exception.
  - Add a mobile API boundary regression check to `make mobile-check`.
  - Verify focused mobile validation and the full CI gate.
  ### Resolution
  Normalized mobile dashboard API collection fields at the mobile API boundary so deployed `null` empty collections become `[]` before React renders, while non-array values raise `mobile_api_invalid_collection`. Added a mobile API-boundary regression script to `make mobile-check`, and extended the config validator to require that guard. Validation passed with focused `make mobile-check` and final `make ci` with 426 integration specs.
  ### Changed Files
  `PLAN.md`, `.mprlab/ISSUES.md`, `Makefile`, `mobile/package.json`, `mobile/scripts/test-api-boundaries.mjs`, `mobile/scripts/validate-mobile-config.mjs`, `mobile/src/api.ts`.

- [x] [B031] (P1) Stabilize seeded dashboard auth after login redirects.
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

- [x] [B027] (P1) Ignore external-asset route fetches cancelled by test teardown.
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

- [x] [B028] (P1) Keep Timezones labels aligned with geographic markers.
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

- [x] [B029] (P2) Show native feedback context in the operator mobile app.
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

- [x] [B030] (P2) Proxy LA Sentry routes in the local gHTTP template.
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

- [x] [B026] (P0) Replace incorrect four-hour login test with console-clean stale-idle coverage.
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

- [x] [B025] (P0) Make Google popup auth compatible with edge opener policy.
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

- [x] [B024] (P0) Cover landing-page login after four idle hours.
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

- [x] [B023] (P1) Use `mpr-ui` Google Identity testing helper instead of stub globals.
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

- [x] [B022] (P1) Share Google Identity stub readiness across auth specs.
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

- [x] [B021] (P1) Stabilize auth browser harness readiness in CI.
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

- [x] [B020] (P1) Restore feedback bubble color customization.
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

- [x] [B019] (P0) Recover login after long-idle Google nonce expiry.
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

- [x] [B018] (P0) Restore GitHub Actions browser setup before CI timeout.
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
  Blocked: applying and verifying the corrected production runtime config still requires the `mprlab-gateway` deploy step that previously stopped at the interactive `Gateway sudo password:` prompt.
  ### Changed Files
  `cmd/configaudit/main.go`, `cmd/configaudit/main_test.go`, `configs/.env.loopaware.example`, `configs/.env.loopaware.computercat.example`, `tests/specs/header-auth-state.spec.js`, `web/app/index.html`, `.mprlab/ISSUES.md`.

- [x] [B012] (P0) Restore integration Pinguin startup after config schema drift.
  ### Summary
  Baseline `make ci` no longer reaches the API and browser integration suites because the Pinguin service in `tests/docker-compose.yml` rejects LoopAware's bundled Pinguin config. The runtime now treats TAuth identity metadata as a shared-shell concern and fails on the stale `tenants[].identity` shape.
  ### Deliverables
  - Update the LoopAware-owned Pinguin config fixture to match the current Pinguin runtime schema.
  - Keep TAuth provider metadata in the TAuth/shared-shell config path, not inside Pinguin tenant config.
  - Preserve email notification test wiring for the LoopAware tenant.
  - Verify the API integration slice starts successfully before proceeding with reporting work.
  ### Resolution
  Removed stale Pinguin tenant identity metadata from the LoopAware Pinguin config, added the current disabled SMTP submission/forwarding section defaults to the tracked test and example envs, and removed the obsolete config-audit Google-provider invariant. Verified the API integration slice passes with 108 tests and the full `make ci` gate passes with 358 integration tests.

- [x] [B013] (P0) Scope scheduled report device and timezone totals to the report window.
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

- [x] [B014] (P1) Fix global traffic report review findings.
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

- [x] [B015] (P1) Remove aggregate top-pages sections from all-sites traffic reports.
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

- [x] [B017] (P1) Keep dashboard SSE streams alive through gateway read timeouts.
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

## Improvements

- [x] [I027] (P1) Add a public SEO resources cluster.
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

- [x] [I026] (P1) Tighten visitor-location edge geo inference and aggregation.
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

- [x] [I025] (P1) Improve visitor-location confidence with edge geo signals.
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

- [x] [I024] (P1) Replace visitor timezone reporting with inferred locations.
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

- [x] [I023] (P1) Add selected-site Traffic intervals and CSV export.
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

- [x] [I014] (P1) Replace duplicate timezone views with a visit-density map.
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

- [x] [I015] (P1) Replace duplicate device views with an icon row graph.
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

- [x] [I016] (P1) Replace duplicate top-pages views with a ranked path row graph.
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

- [x] [I017] (P1) Rename selected-site traffic breakdown section headings.
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

- [x] [I018] (P1) Clean up the selected-site timezone map visual treatment.
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

- [x] [I020] (P1) Render the Timezones bubble graph on a real world map.
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

- [x] [I021] (P1) Prove Timezones bubbles use real geographic placement.
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

- [x] [I022] (P1) Back the Timezones world map with supported geo tooling.
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

- [x] [I019] (P1) Differentiate tablet and mobile device icons in the Devices row graph.
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

- [x] [I013] (P1) Mark X-axis time labels on traffic trend charts.
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

- [x] [I012] (P1) Show count scales on traffic trend charts.
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

- [x] [I006] (P1) Add graphical and portfolio traffic reporting.
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

- [x] [I007] (P1) Move all-sites traffic reporting entry into settings.
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

- [x] [I008] (P1) Split all-sites traffic into a separate dashboard view.
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

- [x] [I009] (P1) Introduce global report-library visual language.
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

- [x] [I010] (P1) Persist scoped global traffic report definitions.
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

- [x] [I011] (P1) Move the account card into Account Settings.
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


## Planning
*do not implement yet*
