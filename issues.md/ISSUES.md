# ISSUES (Append-only section-based log)

Entries record newly discovered requests or changes, with their outcomes. No instructive content lives here. Read @NOTES.md for the process to follow when fixing issues.

Read @AGENTS.md, @ARCHITECTURE.md, @README.md, @PRD.md. Read @POLICY.md, PLANNING.md, @NOTES.md, and @ISSUES.md under issues.md/.  Start working on open issues. Work autonomously and stack up PRs. Prioritize bugfixes.

Each issue is formatted as `- [ ] [LA-<number>]`. When resolved it becomes `- [x] [LA-<number>]`.

## Features (117–199)

- [x] [LA-113] Add `target` parameter to subscribe.js for rendering into specific DOM elements.
  Priority: P1
  Goal: Allow subscribe.js to render the subscribe form into a specific element instead of always appending to document.body. This enables embedding the subscribe widget inside cards, modals, or other constrained containers without using iframes.
  Deliverable: PR that adds `target` parameter support to subscribe.js; form renders into `document.getElementById(targetId)` when specified, falls back to `document.body` when not.
  Use case: Marco Polo Research Lab landing page embeds subscribe forms on flippable card backs. Using iframes causes CORS issues (srcdoc iframes have `Origin: null`). Direct embedding with target parameter avoids this.
  Resolution: Added target param/data-target support to subscribe.js and integration coverage ensuring inline forms render into the requested container.
  Implementation notes:
  - Add `target` to `parseConfig()` alongside existing params
  - Support both URL param (`?target=my-element-id`) and data attribute (`data-target="my-element-id"`)
  - Modify `renderInline()` to accept optional `targetElement` parameter
  - Resolve target in `main()`: `document.getElementById(config.targetId)` or fallback to `document.body`
  Docs/Refs:
  - `internal/httpapi/assets/subscribe.js`
- [x] [LA-114] Subscribe test page needs a target element ID input for subscribe.js previews.
  Priority: P1
  Goal: Let operators set the subscribe.js target element ID on the test page so inline previews render into the specified container.
  Deliverable: Target element ID input wired to the inline preview container and integration coverage verifying the preview updates.
  Docs/Refs:
  - `internal/httpapi/templates/subscribe_test.tmpl`
  - `internal/httpapi/site_subscribe_test_handlers.go`
  - `internal/httpapi/dashboard_integration_test.go`
  Resolution: Added a target input that updates the inline preview container ID, and integration coverage asserting the preview renders in the updated target; `make ci` passes.
- [ ] [LA-115] (P0) Integrate logged in drop down with the latest version of mpr-ui.
  mpr-ui provides the mpr-user element which can be integrated with tauth and shown instead of custom logic we employ for displaying a user. Check @tools/mpr-ui for the documentation and @tools/mpr-ui/demoi for the integration examples
- [x] [LA-116] (P1) Refactor LoopAware into a separate frontend and backend to adopt TAuth via mpr-ui.
  Priority: P0
  Goal: Split the UI into a dedicated frontend app that loads `tauth.js` + mpr-ui DSL, while the backend becomes a clean API that validates TAuth sessions.
  Deliverable: A documented architecture/migration plan that defines service boundaries, routing/origin model, auth flow, and rollout steps.
  Docs/Refs:
  - `tools/TAuth/docs/usage.md`
  - `tools/TAuth/docs/migration.md`
  - `issues.md/AGENTS.FRONTEND.md`
  - `issues.md/AGENTS.GO.md`
  - `issues.md/AGENTS.DOCKER.md`
  Resolution: Added `--serve-mode` (`monolith|web|api`) to `cmd/server`, updated `docker-compose.computercat.yml` + ghttp proxy templates to run split `loopaware-web` and `loopaware-api` behind a single origin, and documented rollout/routing in `docs/LA-116-split-frontend-backend.md`; `make ci` passes.


## Improvements (416–515)

- [x] [LA-213] Dashboard section tabs should span full width and split into 3 equal parts.
  Priority: P2
  Goal: Feedback/Subscriptions/Traffic tab buttons fill the available width and each takes exactly 1/3 of the row (responsive).
  Deliverable: PR adjusting tab markup/CSS and updating dashboard integration tests as needed.
  Resolution: Switched section tabs to `nav-justified` + `w-100` and added headless integration assertions for equal computed tab widths.
- [x] [LA-339] ![Full name login area](image-3.png) Remove the full name login area in favor of current avatar only design ![alt text](image-4.png).
  Ensure that the layout for the full name and log out button is deleted from all pages
  Resolution: Added auth script binding on widget/subscribe/traffic test pages and integration coverage confirming the default header profile layout is removed; `make ci` passes.
- [x] [LA-413] Autosave should not reload the dashboard or interrupt typing while editing site settings.
  Priority: P1
  Goal: Autosave site edits without refreshing the form or dropping focus while operators type.
  Deliverable: PR adding debounced autosave with integration coverage proving input focus/value persists during autosave.
  Docs/Refs:
  - `internal/httpapi/templates/dashboard.tmpl`
  - `internal/httpapi/dashboard_integration_test.go`
  Resolution: Added debounced autosave for site settings with a non-invasive render path and integration coverage that preserves subscribe-origin typing; `make ci` passes.
- [x] [LA-414] Remove the remnant avatar/name/sign-out markup that appears alongside the LoopAware header profile menu.
  Priority: P1
  Goal: Only the LoopAware profile dropdown renders in the dashboard header; default mpr-ui profile elements remain hidden/removed.
  Deliverable: PR that removes the duplicate header markup and adds integration coverage for a single visible profile control.
  Docs/Refs:
  - `internal/httpapi/templates/dashboard_header.tmpl`
  - `internal/httpapi/templates/dashboard.tmpl`
  - `internal/httpapi/public_assets.go`
  - `tools/mpr-ui/docs/custom-elements.md`
  Resolution: Removed default mpr-ui profile/settings/sign-in elements when the LoopAware profile menu is present and added integration coverage ensuring they are absent; `make ci` passes.
- [x] [LA-415] Dashboard sign-in requires multiple attempts instead of completing on first click.
  Priority: P1
  Goal: The first sign-in interaction completes authentication without extra prompts or repeat clicks.
  Deliverable: PR that eliminates double sign-in behavior and adds integration coverage for a single sign-in flow.
  Docs/Refs:
  - `internal/httpapi/public_assets.go`
  - `internal/httpapi/templates/dashboard_header.tmpl`
  - `tools/mpr-ui/docs/custom-elements.md`
  - `tools/TAuth/docs/usage.md`
  Resolution: Gated the Google sign-in control until the nonce-backed GIS initialization is available and added integration coverage to verify the gate releases after nonce readiness; `make ci` passes.


## BugFixes (339–399)

- [x] [LA-317] mpr-ui footer menu label “Built by Marco Polo Research Lab” is invisible.
  Priority: P1
  Goal: Footer label is visible in both light and dark themes and matches the mpr-ui demo styling.
  Deliverable: PR that removes/adjusts conflicting CSS and validates footer label visibility; screenshot-based evidence if needed.
  Resolution: Stop passing `toggle-button-id` so mpr-ui can populate the footer menu button label/classes, and synchronize `data-mpr-theme` with `data-bs-theme` so light-mode tokens apply; added headless integration coverage for computed label color.
  Docs/Refs:
  - `tools/mpr-ui/demo/index.html`
  - `internal/httpapi/templates/dashboard.tmpl`
  - `internal/httpapi/public_assets.go`
  Execution plan:
  - Compare LoopAware footer DOM/CSS against the mpr-ui demo.
  - Identify overriding selectors in LoopAware CSS (especially for footer text colors).
  - Adjust styles to avoid clobbering mpr-ui defaults; verify both themes.
- [x] [LA-318] Theme toggle defaults and mapping are wrong (left = light, right = dark).
  Priority: P0
  Goal: The left toggle state represents light theme, the right state represents dark theme, and the initial UI state matches the applied theme.
  Deliverable: PR that fixes theme toggle behavior across landing, dashboard, and public pages; automated coverage updated.
  Resolution: Reordered `mpr-footer` theme modes to `light` then `dark` and added headless integration coverage for landing/privacy/dashboard toggle state + applied theme.
  Docs/Refs:
  - `issues.md/AGENTS.FRONTEND.md`
  - `internal/httpapi/footer.go`
  - `internal/httpapi/templates/dashboard.tmpl`
  - `internal/httpapi/public_assets.go`
  Execution plan:
  - Identify the single source of truth for theme mode (document attribute + storage key).
  - Fix toggle UI state mapping (mpr-ui footer event + local storage + `data-bs-theme`).
  - Add/adjust integration assertions around theme attribute and toggle state.
- [x] [LA-319] Additional subscribe origins may not persist/display after saving and returning to the site editor.
  Priority: P1
  Goal: When operators add additional subscribe origins, save the site, and later re-open the same site in the dashboard, the saved origins are shown in the “Additional subscribe origins” editor.
  Deliverable: PR with a reproducible failing test + fix (or documented repro steps if environment-specific).
  Notes: Reported behavior is “added origins do not appear after returning to the site”; confirm whether the operator clicked Save, and whether the dashboard was refreshed or the site was re-selected without a reload.
  Resolution: Unable to reproduce after verifying persistence/rehydration coverage for subscribe origins.
  Docs/Refs:
  - `internal/httpapi/templates/dashboard.tmpl`
  - `internal/httpapi/admin.go` (site responses)
  - `internal/httpapi/subscribe_allowed_origins_dashboard_integration_test.go`
- [x] [LA-320] Login loop between `/login` and `/app` when TAuth session cookie name is customized.
  Priority: P0
  Goal: Dashboard loads after successful TAuth login; LoopAware validates the configured session cookie name.
  Deliverable: PR that passes the TAuth session cookie name into the validator and updates env/docs.
  Notes: LoopAware currently assumes `app_session`; TAuth tenant uses `app_session_loopaware`.
  Resolution: Added `TAUTH_SESSION_COOKIE_NAME` config/flag, wired it into session validation, and updated env samples/docs plus config-audit.
  Docs/Refs:
  - `cmd/server/main.go`
  - `internal/httpapi/auth.go`
  - `configs/.env.loopaware`
- [x] [LA-321] Logout should redirect to the landing page from authenticated views.
  Priority: P1
  Goal: After logging out via the header, the dashboard (and related authenticated pages) return to `/login`.
  Deliverable: PR that redirects on `mpr-ui:auth:unauthenticated` when configured on dashboard headers.
  Resolution: Added a logout-redirect flag to dashboard headers and routed unauthenticated events back to the landing path.
  Docs/Refs:
  - `internal/httpapi/public_assets.go`
  - `internal/httpapi/templates/dashboard_header.tmpl`
- [x] [LA-322] Landing theme preference does not persist to the dashboard after login.
  Priority: P1
  Goal: When no explicit theme is stored, the landing theme persists to localStorage and the dashboard uses it after login.
  Deliverable: PR that persists the landing default theme and adds integration coverage for landing-to-dashboard theme propagation.
  Notes: Landing defaults to dark without persisting; dashboard defaults to light when no preference exists.
  Resolution: Persisted landing default theme on boot and added integration coverage for landing-default propagation into the dashboard.
  Docs/Refs:
  - `internal/httpapi/public_assets.go`
  - `internal/httpapi/dashboard_integration_test.go`
  - `internal/httpapi/templates/dashboard.tmpl`
- [x] [LA-323] Dashboard header profile menu regressed to oversized buttons after TAuth migration.
  Priority: P1
  Goal: Restore the avatar dropdown with account settings and logout while keeping TAuth-backed authentication.
  Deliverable: PR that renders the profile dropdown and updates integration coverage for opening the settings modal.
  Notes: The header currently shows separate large buttons instead of a compact dropdown menu.
  Resolution: Reintroduced a compact avatar dropdown in the dashboard header, hid the built-in header actions, added dropdown open fallbacks, and updated integration coverage to open the menu before clicking settings; `make ci` passes.
  Docs/Refs:
  - `internal/httpapi/public_assets.go`
  - `internal/httpapi/templates/dashboard_header.tmpl`
  - `internal/httpapi/templates/dashboard.tmpl`
  - `internal/httpapi/dashboard_integration_test.go`
- [x] [LA-324] Landing page does not redirect to the dashboard after successful TAuth login.
  Priority: P0
  Goal: When a user authenticates via TAuth, the landing page should redirect them to `/app` once the header is authenticated.
  Deliverable: PR that detects already-authenticated header state on boot, redirects to the dashboard, and adds integration coverage.
  Notes: The header shows the logged-in user, but no navigation occurs.
  Resolution: Avoided HTML-escaped `<` in the auth script loop, added a `getCurrentUser` bootstrap fallback, and tightened the landing redirect integration assertions; `make ci` passes.
  Docs/Refs:
  - `internal/httpapi/public_assets.go`
  - `internal/httpapi/templates/landing.tmpl`
  - `internal/httpapi/landing.go`
  - `internal/httpapi/dashboard_integration_test.go`
- [x] [LA-325] Logout does not clear the TAuth session from the dashboard header.
  Priority: P0
  Goal: Clicking Logout clears the TAuth session cookie and returns the user to `/login`, with the header showing unauthenticated state on reload.
  Deliverable: PR that wires the custom dropdown logout to the same auth/logout path as mpr-ui (or TAuth), with integration coverage for logout + redirect.
  Notes: The custom menu uses `public_assets.go` `handleLogout` → `window.logout` fallback; if `tauth.js` is missing/mis-wired, we redirect without invalidating the session. Confirm the logout helper includes `X-TAuth-Tenant` and correct `tauth-url`. If the issue is in `tauth.js` or mpr-ui auth helpers, note/update those repos.
  Resolution: Always issue a direct `/auth/logout` POST after `window.logout` (tauth.js swallows errors), plus logout integration coverage that asserts the logout request and landing redirect; `make ci` passes.
  Docs/Refs:
  - `internal/httpapi/public_assets.go`
  - `internal/httpapi/templates/dashboard_header.tmpl`
  - `tools/mpr-ui/docs/custom-elements.md`
  - `tools/TAuth/docs/migration.md`
- [x] [LA-326] Account settings modal opens blank from the header dropdown.
  Priority: P1
  Goal: The Account settings action opens a modal with the expected auto-logout controls and descriptive copy across both themes.
  Deliverable: PR that ensures the modal renders content and remains readable in both light/dark modes; add integration coverage for settings modal visibility + content.
  Notes: The menu item targets `SettingsModalID`, but users report an empty modal. Verify Bootstrap modal wiring, `SettingsModalID` parity between `dashboard_header.tmpl` and `dashboard.tmpl`, and theme styling that could render content invisible. Consider migrating to `<mpr-settings>` if the modal is brittle.
  Resolution: Bound the dropdown settings action to open the Bootstrap modal directly from the custom profile menu and added integration coverage that asserts modal content text + contrast; `make ci` passes.
  Docs/Refs:
  - `internal/httpapi/templates/dashboard_header.tmpl`
  - `internal/httpapi/templates/dashboard.tmpl`
  - `internal/httpapi/web.go`
- [x] [LA-327] Dashboard header/footer palette is out of sync with body theme.
  Priority: P1
  Goal: Header, footer, and body share the same light/dark palette so the UI feels cohesive in both themes.
  Deliverable: PR that aligns mpr-ui theme tokens with Bootstrap theme state (or documents a new preset) and adds integration coverage for theme consistency.
  Notes: Landing page overrides `--mpr-color-*` tokens, but dashboard does not. mpr-ui uses its own CSS variables keyed off `data-mpr-theme`, while the dashboard applies Bootstrap classes; the palettes can diverge. If a shared preset or `data-bs-theme` integration belongs in mpr-ui, note it for that repo.
  Resolution: Bind Bootstrap body color variables to mpr-ui theme tokens and drop conflicting body bg classes, plus new integration coverage comparing header/footer palette deltas against the body in light/dark modes; `make ci` passes.
  Docs/Refs:
  - `internal/httpapi/templates/dashboard.tmpl`
  - `internal/httpapi/public_assets.go`
  - `tools/mpr-ui/README.md`
- [x] [LA-328] Dashboard header profile toggle should be avatar-only with dropdown actions.
  Priority: P1
  Goal: The header shows a single avatar (no wide name button) that opens the settings/logout dropdown; display name can appear inside the menu.
  Deliverable: PR that adjusts header markup/CSS and updates profile name injection logic to render inside the dropdown instead of the toggle.
  Notes: `dashboard_header.tmpl` currently renders the name inside the toggle button, producing a large pill button. Update the template and `public_assets.go` profile sync to support the compact avatar-only toggle.
  Resolution: Moved the display name into the dropdown, rendered an avatar-only toggle with updated header styling and profile sync attributes, and added integration coverage for avatar-only toggle/menu display; landing auth harness now stubs mpr-ui auth bootstrap to keep redirect checks stable; `make ci` passes.
  Docs/Refs:
  - `internal/httpapi/templates/dashboard_header.tmpl`
  - `internal/httpapi/templates/dashboard.tmpl`
  - `internal/httpapi/public_assets.go`
  - `internal/httpapi/dashboard_integration_test.go`
- [x] [LA-329] Logout button does not terminate the session after the header refactor.
  Priority: P1
  Goal: Clicking Logout clears the TAuth session and returns the user to the landing page without re-authentication loops.
  Deliverable: PR that hardens the logout flow against helper/fetch failures and adds integration coverage for the fallback behavior.
  Notes: Regression observed after the header dropdown refactor; logout should still work even when the TAuth helper or fetch path fails (e.g., transient CORS issues). If the fix needs a TAuth change, document it here.
  Resolution: Added a form-post fallback when both `window.logout` and the fetch path fail, and added integration coverage that forces both failures and verifies the fallback + redirect; `make ci` passes.
  Docs/Refs:
  - `internal/httpapi/public_assets.go`
  - `internal/httpapi/dashboard_integration_test.go`
- [x] [LA-330] Header dropdown actions are unresponsive in some sessions.
  Priority: P1
  Goal: Settings and logout reliably bind even when the header renders late or auth attributes update before slot content mounts.
  Deliverable: PR that retries custom profile binding and resolves logout endpoints from tauth.js when available, plus coverage if possible.
  Notes: Users report the settings modal and logout action do nothing; likely the custom menu handlers never attach or the logout request targets the wrong origin.
  Resolution: Added retry logic for custom profile binding, made logout prefer tauth.js `getAuthEndpoints()` when available, and relaxed settings click handling so Bootstrap’s data API still works when manual modal control is unavailable; `make ci` passes.
  Docs/Refs:
  - `internal/httpapi/public_assets.go`
  - `internal/httpapi/dashboard_integration_test.go`
- [x] [LA-331] Remove auth fallbacks and rely solely on mpr-ui auth events.
  Priority: P1
  Goal: Login/logout announcements and redirects are driven only by `mpr-ui:auth:*` events, without MutationObserver or manual bootstrap fallbacks.
  Deliverable: PR that removes attribute/observer fallbacks and confirms event-driven redirects still work.
  Notes: Required to align with mpr-ui integration guidance and avoid double-trigger behavior.
  Resolution: Removed attribute observers/bootstraps, and now dispatches `mpr-ui:auth:authenticated` when dashboard user data loads so the header updates via events only; `make ci` passes.
  Docs/Refs:
  - `internal/httpapi/public_assets.go`
  - `internal/httpapi/templates/dashboard.tmpl`
- [x] [LA-332] ![alt text](image-1.png) The logout notification is floating in space instead of being sticky to the bottom of the screen.
  Priority: P1
  Goal: Session timeout notification is fixed to the bottom edge of the viewport.
  Deliverable: PR that pins the timeout notification to the bottom and adds integration coverage for its position.
  Resolution: Added bottom anchoring to the timeout banner and integration coverage asserting its bottom alignment.
- [x] [LA-333] Safari header dropdown actions are unresponsive after auth.
  Priority: P1
  Goal: Avatar dropdown opens and settings/logout clicks work on Safari without missing bindings.
  Deliverable: PR that makes header menu bindings resilient to mpr-ui re-renders; update tests if feasible.
  Notes: Safari appears to drop or never attach the click handlers on the avatar menu items, so no logout request is sent and the settings modal never opens.
  Resolution: Switched auth script rendering to a text template to prevent HTML escaping in JS, and added delegated profile menu click handling to keep settings/logout responsive across browsers; `make ci` passes.
  Docs/Refs:
  - `internal/httpapi/public_assets.go`
  - `internal/httpapi/templates/dashboard_header.tmpl`
  - `tools/mpr-ui/docs/custom-elements.md`
- [x] [LA-334] Logout occurs much faster than the configured timeout.
  Priority: P1
  Goal: Authentication sessions honor the configured timeout before forcing logout.
  Deliverable: PR that identifies the premature logout trigger, aligns the effective timeout with configuration, and adds/updates integration coverage for session duration.
  Notes: Reported behavior indicates logout occurs significantly earlier than the configured session timeout; confirm whether this is driven by the dashboard inactivity timer vs. server/TAuth session expiry.
  Resolution: Scoped auto-logout settings to user-specific storage keys, clear legacy storage after migration, and added integration coverage to confirm per-user settings; `make ci` passes.
- [x] [LA-335] Google Sign-In auto-suggests login after a timed-out logout.
  Priority: P1
  Goal: After an inactivity-triggered logout, Google Sign-In should not immediately prompt or auto-suggest login without user action.
  Deliverable: PR that suppresses auto-prompt/auto-select behavior after timeout-driven logout and adds integration coverage for the post-timeout login UX.
  Notes: Observed behavior is a Google Sign-In prompt immediately after timeout logout; confirm whether GIS auto-select or mpr-ui auth bootstrap is responsible and ensure the prompt only appears on explicit user intent.
  Resolution: Disabled Google auto-select during timeout-driven logout and added integration coverage to verify the suppression; `make ci` passes.
- [x] [LA-336] Additional subscribe origins disappear after logout/login.
  Priority: P1
  Goal: Additional subscribe origins remain visible in the dashboard editor after a logout/login cycle and are enforced by the backend.
  Deliverable: PR that persists and rehydrates additional subscribe origins in the UI after re-auth and adds coverage for visibility + origin enforcement.
  Notes: Reported behavior: added origins are not shown after logging out and back in, even though they were saved.
  Resolution: Unable to reproduce; added headless coverage to rehydrate subscribe origins after re-login and verified persistence in storage.
- [x] [LA-337] Subscribe form renders a name field even when the widget disables it.
  Priority: P1
  Goal: When the subscribe widget requests no name field, the rendered form omits it consistently across embed/test flows.
  Deliverable: PR with a failing integration test plus a fix that honors the widget flag end-to-end.
  Notes: Reported behavior: subscribe form still renders the name input even when the widget says no name.
  Resolution: Omitted the name input when `name_field=false` in the embed script and subscribe test preview, with integration coverage asserting the field is absent.
  Docs/Refs:
  - `internal/httpapi/subscribe_template.go`
  - `internal/httpapi/subscribe_demo_template.go`
  - `internal/httpapi/templates/subscribe_test.tmpl`
- [x] [LA-338] Defer timeout start until user settings are loaded.
  Because applyAutoLogoutSettingsForUser(null) runs before loadUser() and the session-timeout manager is started before loadUser() resolves (see sessionTimeoutStartRequested/sessionTimeoutManager.start() later in this template), the idle timer begins with default settings until the user-specific key is known. After this change clears the legacy base key, a slow /me response (e.g., degraded API or high latency) can trigger the 60/120-second defaults even for users who have configured longer timeouts, reintroducing “premature logout” in that scenario. Consider delaying sessionTimeoutStartRequested until applyAutoLogoutSettingsForUser(state.user) runs or caching the last user key so the correct settings are loaded before starting the timer.
  Resolution: Deferred session-timeout start until after user settings load and added integration coverage with a delayed /api/me response to confirm the start gate.


## Maintenance (418–499)

- [x] [LA-406] Cleanup:.
  1. Review the completed issues and compare the code against the README.md and ARCHITECTURE.md files.
  2. Update the README.md and ARCHITECTURE.
  3. Clean up the completed issues.
  reconciled the README REST API table, subscription token routes, and dashboard feature list with the shipped behavior; expanded ARCHITECTURE.md with an overview of components and key flows.
- [x] [LA-407] Polish:.
  1. Review each open issue
  2. Add additional context: dependencies, documentation, execution plan, goal
  3. Add priroity and deliverable. Reaarange and renumber issues as needed.
- [x] [LA-408] Dashboard widget bottom offset integration test fails after dependency updates.
  Priority: P1
  Goal: Ensure the dashboard widget bottom offset test waits for site selection/value population so it passes after dependency updates.
  Deliverable: PR updating the integration test readiness with passing `make ci`.
  Notes: Failure observed in `TestDashboardWidgetBottomOffsetStepButtonsAdjustAndPersist` due to an empty bottom offset input value.
  Resolution: Added a site-selection readiness wait to the integration test; `make ci` passes.
- [x] [LA-409] Migrate LoopAware auth UI and server validation to the latest TAuth client integration.
  Priority: P0
  Goal: Replace GAuss session handling with TAuth session validation and use the mpr-ui declarative DSL (with `tauth.js`) for auth UI.
  Deliverable: PR updating templates, auth middleware, and config to use `tauth.js` + `<mpr-header>`; tests and `make ci` pass.
  Notes: Follow `tools/TAuth/docs/migration.md` and mpr-ui docs; ensure refresh handling via `apiFetch` and update login/logout flows.
  Resolution: Replaced GAuss sessions with TAuth validator, wired mpr-ui header + `tauth.js` across templates, updated env/config/doc/test coverage, and verified `make ci`.
- [x] [LA-411] Align LoopAware footer integration with mpr-ui v3.4.0 DSL.
  Priority: P1
  Goal: Remove legacy footer attributes (`links`, `theme-mode`) and sync theme state via `theme-config` + document attributes while keeping toggle behavior and link catalog intact.
  Deliverable: PR that updates footer rendering + theme scripts/tests to use `links-collection` and `theme-config.initialMode`; `make ci` passes.
  Docs/Refs:
  - `tools/mpr-ui/docs/custom-elements.md`
  - `tools/mpr-ui/CHANGELOG.md`
  - `pkg/footer/footer.go`
  - `internal/httpapi/public_assets.go`
  - `internal/httpapi/templates/dashboard.tmpl`
  Resolution: Switched footer rendering to `links-collection`, moved initial theme into `theme-config.initialMode`, removed `theme-mode` syncing, and updated landing/privacy/dashboard/theme toggle tests; `make ci` passes.
- [x] [LA-412] do not allow repeated login dialog after log out.
  Currently a dialog to log in appears after logout. DO not allow it, and expect users explicit actions instead.
  Google Sign in shows automatic pop up to log in. That is unnessary and we want to rely on users explicit click. Investiaget if google sign in offers a parameter in its initialization to disable auto-login, check if we can use it with TAuth/mpr-ui initialization (check @tools/TAuth and @tools/mpr-ui).
  Resolution: Disabled Google auto-select on explicit logout and unauthenticated events and added integration coverage for logout-triggered suppression; `make ci` passes.
- [x] [LA-416] Add missing product and integration docs.
  Priority: P2
  Goal: Provide missing product and integration docs referenced by process instructions and docs.
  Deliverable: PRs adding `PRD.md`, `PLANNING.md`, and mpr-ui custom element/integration guides.
  Docs/Refs:
  - `README.md`
  - `ARCHITECTURE.md`
  - `docs/LA-200-mpr-ui-gauth.md`
  Resolution: Added PRD/PLANNING docs for LoopAware, and documented mpr-ui custom elements/integration in MarcoPoloResearchLab/mpr-ui#127; `make ci` passes.
  ### Recurring (600-699)
  **close when done but do not remove**
- [x] [LA-417] Ensure coverage target creates output directory.
  Priority: P2
  Goal: `make coverage` should succeed in a clean checkout by creating the `.cache` directory before writing coverage output.
  Resolution: Added a `mkdir -p $(CURDIR)/.cache` step to the coverage target.
- [x] [LA-418] Header shows duplicate avatars (mpr-header + user menu).
  Priority: P1
  Goal: Render a single avatar (mpr-ui `mpr-user`) and ensure the user menu includes Account settings + Logout.
  Deliverable: Replace legacy profile dropdown with a single `mpr-user` in the header `aux` slot, remove unused legacy code, and add integration coverage guarding against duplicate avatars and missing menu items.
  Docs/Refs:
  - `tools/mpr-ui/docs/custom-elements.md`
  - `internal/httpapi/templates/dashboard_header.tmpl`
  - `internal/httpapi/public_assets.go`
  - `internal/httpapi/dashboard_integration_test.go`
  Resolution: Replaced legacy profile dropdown with a single `mpr-user` avatar menu (Account settings + Logout), removed unused legacy profile CSS/JS, and added integration coverage asserting a single visible avatar + expected menu items on both landing and dashboard; `make ci` passes.
- [x] [LA-419] Docker Compose should serve `computercat.tyemirov.net:4443` with TLS.
  Priority: P1
  Goal: Run the LoopAware + TAuth stack on `https://computercat.tyemirov.net:4443` (not `localhost`) using the shared certificate files.
  Deliverable: Add a docker compose variant that terminates TLS on port `4443` using certs from `/media/share/Drive/exchange/certs/computercat`, and update env/config defaults so the browser uses `https://computercat.tyemirov.net:4443` for LoopAware and TAuth endpoints.
  Docs/Refs:
  - `docker-compose.yml`
  - `tools/TAuth/docs/usage.md`
  Resolution: Added `docker-compose.computercat.yml` using `ghttp` TLS reverse proxy on `:4443` (no nginx), documented required env/proxy setup in `configs/README.md`, and verified `make ci`; `make ci` passes.
- [x] [LA-420] Consolidate environment templates under `configs/`.
  Priority: P1
  Goal: Avoid split `.env*` templates between the repo root and `configs/`; keep Docker Compose configuration and examples in one place.
  Deliverable: Move tracked env templates into `configs/` with `*.example` files, update `README.md` to reference `configs/` env files for `docker compose`, and remove legacy root env templates.
  Resolution: Removed legacy root templates (`.env.sample`, `config.tauth.yaml`), added tracked `configs/.env.*.example` templates, and updated docs to reference the `configs/` env layout; `make ci` passes.
- [x] [LA-421] Provide computercat-ready env templates under `configs/`.
  Priority: P1
  Goal: Make `docker-compose.computercat.yml` runnable on `https://computercat.tyemirov.net:4443` without editing the local-compose env files.
  Deliverable: Add `configs/.env.*.computercat.example` templates and update `docker-compose.computercat.yml` + `configs/README.md` to use dedicated `configs/.env.*.computercat` env files.
  Resolution: Added computercat env templates, switched `docker-compose.computercat.yml` to consume the dedicated env files, and updated docs with copy/edit instructions; `make ci` passes.


## Planning (500–59999)
*do not implement yet*

- [x] [LA-422] (P0) Make split deployment fully independent: web serves all HTML/CSS/JS, api is strictly API-only.
  Goal: Complete the LA-116 migration so ghttp can route `/api/*` to `loopaware-api` and everything else to `loopaware-web` (plus TAuth routes), with no HTML/JS assets served by the api service.
  Deliverable: Move operator tool HTML pages, subscription link pages, and public JS assets (`/widget.js`, `/subscribe.js`, `/pixel.js`) to the web service; move tool JSON endpoints under `/api/sites/*`; add any required public API endpoints (e.g. widget placement config) so the JS assets remain dynamic; update proxy templates and integration coverage; `make ci` passes.
  Docs/Refs:
  - `docs/LA-116-split-frontend-backend.md`
  - `cmd/server/routes.go`
  - `configs/.env.ghttp*.example`
  Resolution: Completed the API-only backend + static web split (see detailed resolution below).
  Verification: `make ci` passes.

- [x] [LA-422] (P0) Make split deployment fully independent: web serves all HTML/CSS/JS, api is strictly API-only.
  Resolution: Completed the strict split. `loopaware-web` now serves all HTML pages and public JS assets (`/widget.js`, `/subscribe.js`, `/pixel.js`, `/subscribe-demo`, subscription link pages, and operator tool pages). `loopaware-api` now serves only `/api/*` endpoints (public ingestion + authenticated APIs), including new public JSON endpoints for widget placement (`GET /api/widget-config`) and subscription link hydration (`GET /api/subscriptions/{confirm-link,unsubscribe-link}`), plus tool endpoints moved under `/api/sites/:id/...`.
  Notes:
  - Public JS assets are static and fetch runtime config from the API.
  - Tool pages and subscription pages are HTML shells that hydrate via API calls (no direct DB access in web mode).
  - gHTTP proxy templates updated to route `/api/` to `loopaware-api`, everything else to `loopaware-web`, plus TAuth routes.
  Verification: `make ci` passes.

- [ ] [LA-426] Replace placeholder-only inputs with labeled fields in the static frontend.
  Priority: P1
  Goal: Remove placeholder-only UX in the dashboard, widget, and subscribe flows and use explicit labels with specific copy.
  Deliverable: Update `web/app` pages plus `web/widget.js` and `web/subscribe.js` to render labeled inputs and remove placeholder text; update any draft/empty state copy to remain specific; `make ci` passes.

- [x] [LA-427] Raise Go coverage above 95% with focused edge-path tests.
  Priority: P1
  Goal: Add missing unit coverage for CLI entrypoints and edge branches without adding defensive production checks.
  Deliverable: New tests for server/configaudit entrypoints, pinguin proto no-ops, storage backfill/open errors, and visit rollup context cancellation; `make coverage` reports >95%.
  Resolution: Added targeted tests for CLI entrypoints, pinguin proto no-ops, storage open/backfill errors, and visit rollup edge paths.
  Verification: `make test`, `make lint`, `make ci`, `make coverage` (95.2% total).

- [x] [LA-428] Run integration tests against the dockerized stack.
  Priority: P1
  Goal: Restore UI/API integration coverage by running Playwright suites against a composed stack that serves `web/` via gHTTP and proxies `/api/*`.
  Deliverable: Add Playwright test harness under `tests/`, `docker-compose.integration.yml`, integration env templates, and update `make test`/`make ci` plus CI workflow triggers; `make ci` passes.
  Resolution: Added Playwright integration harness and Docker compose stack, updated Makefile/CI triggers, documented integration env templates, and added integration pages plus widget input IDs for UI tests.
  Verification: `make test`, `make lint`, `make ci`.

- [x] [LA-429] Support multi-origin GitHub Pages deployment for `loopaware.mprlab.com`.
  Priority: P1
  Change: When the frontend is served directly from GitHub Pages (no reverse proxy), default the static pages to call the API at `https://loopaware-api.mprlab.com` and load auth from `https://tauth-api.mprlab.com`.
  Resolution: Added runtime origin selection (hostname-based defaults plus `?api_origin=...&tauth_origin=...` overrides), updated widget/subscription/pixel snippets to carry `api_origin`, and removed hardcoded `localhost` references from assets.
  Verification: `make ci` passes.

- [x] [LA-430] Centralize frontend environment mapping for production vs development.
  Priority: P1
  Change: Define a hostname-based environment map so the static frontend can run on GitHub Pages in production and behind a single-origin reverse proxy in development (computercat).
  Resolution: Added `web/runtime-env.js` and updated pages to consume `window.__LOOPAWARE_{API,TAUTH,PINGUIN}_ORIGIN__` instead of duplicating hostname logic across HTML files.
  Verification: `make ci` passes.

- [x] [LA-431] Load frontend runtime origins from `web/config.yml` and drop unused service mappings.
  Priority: P1
  Change: Move the hostname-to-service origin map out of JavaScript and into a static `config.yml` fetched over HTTP at runtime.
  Resolution: Added `web/config.yml` and refactored `web/runtime-env.js` to fetch + validate it synchronously during boot (preserving script ordering), removed all Pinguin-related globals and query params, and now fail fast with a specific error when the hostname is not mapped.
  Verification: `make ci` passes.

- [x] [LA-432] Make `web/config.yml` real YAML (not JSON-in-YAML) for production editing.
  Priority: P1
  Change: Store the frontend environment map as standard YAML so operators can edit it without JSON syntax.
  Resolution: Converted `web/config.yml` to YAML and updated `web/runtime-env.js` to parse it via `js-yaml` (loaded from a pinned CDN script before bootstrap) while retaining strict validation + fail-fast behavior.
  Verification: `make ci` passes.
