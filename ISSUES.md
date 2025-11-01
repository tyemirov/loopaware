# ISSUES (Append-only Log)

Entries record newly discovered requests or changes, with their outcomes. No instructive content lives here. Read @NOTES.md for the process to follow when fixing issues.

## Features (100-199)

- [x] [LA-100] Add settings menu item under the avatar dropdown. Settings is a full screen modal over the page, clicking outside of the settings modal dismisses it — dashboard/widget templates now include settings modal with bootstrap-driven dismissal and integration coverage (go test ./internal/httpapi -run TestDashboardSettingsModalOpensAndDismissesViaBackdrop -count=1).
- [x] [LA-101] Add auto logout configuration to settings menu: Auto logout enabled/disabled. If enabled, have fields to enter time for showing notification and for the logout (default to 60 and 120 seconds). Settings modal now surfaces the toggle and configurable durations persisted locally; idle manager reconfigures on change with integration coverage (go test ./internal/httpapi -run TestDashboardSettingsAutoLogoutConfiguration -count=1).

## Improvements (200-299)

- [x] [LA-201] Move theme switch to the footer, on the left of Built by Marco Polo — toggle now renders beside the Built by prefix within the footer (go test ./...)
- [x] [LA-200] Integrate with Pinguin service. Find the code under @tools/pinguin. Read documentation and understand the code.
      Aceptance criteria are integration tests that verify:
  - When a feedback is received, send a message to the owner (not the registar).
  - Have a column in the feedback messages table titled Delivery with values either "mailed" or "texted" or "no"
  - Docker orchestration of both penguin and loopaware services
  - Updated technical documentation
    The tests must confirm the cotract fulfillment at the boundaries (message sent and it matches expected payload, message received).
    In case bugs are discovered in Pinguin, or enhancements are needed in Pinguin, document them as an issue, and stop working before we fix Pinguin.
- [x] [LA-202] Implement a footer as an alpine component. Ensure that the component accepts styling parameters from the outside. Place the components in MarcoPoloResearchLab/mpr-ui (the repo is under @tools/mpr-ui). Load the footer from the GitHub CDN. Perform changes in @tools/mpr-ui.
      The component shall have
  1. Privacy Terms
  2. Light/Dark theme toggle
  3. Build by Marco Polo Research Lab drop up (pointing up)
  4. Each item in the Build by Marco Polo Research Lab shall have links opening a new page.
- [x] [LA-203] Remove the theme switch from a user menu under the avatar.
  1. Use the same alpine ui footer component as other pages (but style it with current color palette used in dashboard)
  2. Remove user's specific light/dark theme switch
- [x] [LA-204] Make clicking on a favicon of a site open a site itself in a new window — favicon interaction now opens the allowed origin in a new tab with keyboard support; integration test captures window.open calls (go test ./...).
- [x] [LA-205] Make the bottom offset dialog move by the widget position vy 10 pixels when controls are used and allow to manually enter pixel precision
- [x] [LA-206] Clicking Save placement shall be closing return to the dashboard.

## BugFixes (300-399)

- [x] [LA-300] When logged in with the dark theme the dashboard theme is light, when logged in from the light theme, the dashboard theme is dar, find the bug and fix it
- [x] [LA-301] The logout functionality behaviour: display a message after 60 seconds of inactivity. The message should match the theme of the page. Log out after 120 seconds of inactivity (same as +60 seconds since being displayed) — synthetic DOM events no longer reset the idle timer, so the prompt stays visible and the 60/120-second flow holds; regression covers the prompt under synthetic activity (go test ./...).
- [x] [LA-302] LoopAware server exits at startup complaining about missing `pinguin-auth-token` even when running with default docker compose. Resolved by requiring environment-provided bearer token and mirroring `GRPC_AUTH_TOKEN` fallback (go test ./...).
- [x] [LA-301] Ensure that feedback is being sent the very moment it has been received without utilizing a scheduling feature. Do not pass any time for the scheduling time in order to mail the feedback immididatley — Pinguin now ignores scheduled timestamps and dispatches notifications immediately; model and service tests enforce immediate mailing (go test ./..., go test ./tools/pinguin/...).
- [x] [LA-302] Logout dialog is getting dismissed on a mouse move. That is incorrect, once shown, the mouse movement shall not dismiss it — session timeout manager now ignores generic activity when the banner is visible; coverage via go test ./internal/httpapi -run 'TestDashboardSessionTimeout(.*)' -count=1.
- [ ] [LA-303] Site preview shows the Widget test in the light theme despite the rest of the site and theme toggle being in the dark theme. Use the same theme on the widget test as selected.
- [ ] [LA-304] Investigate the send failure from the test page: loopaware  | {"level":"info","ts":1761935043.4114969,"caller":"httpapi/middleware.go:14","msg":"http","method":"POST","path":"/app/sites/e0021c61-fdfd-4d75-8c0b-3c68f3171643/widget-test/feedback","status":401,"dur":0.00002919,"ip":"172.24.0.1","ua":"Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:144.0) Gecko/20100101 Firefox/144.0"}. Notice that it is sufficient not to send the timestamp to pinguin for pinguin to schedule an immediate delivery. @tools/pinguin master is the source of truth on how the pinguin service works
- [ ] [LA-3o5] Clicking Save widget didnt save the new placement of the widget


## Maintenance (400-499)

## Planning (do not work on these, not ready)

- [x] [LA-300] Dashboard theme now honors the latest public selection; regression integration test ensures public preference overrides stale dashboard storage (go test ./...).

## Resolution Log

- [x] [LA-300] Dashboard theme now honors the latest public selection; regression integration test ensures public preference overrides stale dashboard storage (go test ./...).
- [x] [LA-201] Theme switch now lives in the footer beside the Built by Marco Polo branding; public landing/privacy tests enforce placement (go test ./...).
- [x] [LA-200] Added Pinguin-backed notifications for feedback submissions and surfaced delivery statuses across API and dashboard (go test ./...).
- [x] [LA-202] Footer now rendered by shared Alpine component from mpr-ui; templates load CDN module and tests confirm config payload & markup (go test ./...).
- [x] [LA-203] Dashboard theme switch removed from avatar menu; footer toggle drives persistence and integration tests click the new footer control (go test ./...).
- [x] [LA-204] Dashboard site list favicons now open the allowed origin in a new tab with keyboard activation; regression test verifies window.open is invoked (go test ./...).
- [x] [LA-206] Widget test Save placement now redirects back to the dashboard and the updated offset persists; coverage via go test ./internal/httpapi -run 'TestDashboardWidgetBottomOffsetStepButtonsAdjustAndPersist|TestWidgetTestBottomOffsetControlsAdjustPreviewAndPersist' -count=1.
- [x] [LA-205] Bottom offset controls now step by 10px via buttons and keyboard while accepting manual input; dashboard/widget tests cover UI wiring and persistence (go test ./internal/httpapi -run 'TestDashboardWidgetBottomOffsetStepButtonsAdjustAndPersist|TestWidgetTestBottomOffsetControlsAdjustPreviewAndPersist' -count=1).
- [x] [LA-301] Idle prompt now ignores synthetic document events so inactivity still triggers the 60/120-second warning and logout path; browser test dispatches an untrusted mousemove to confirm the banner remains visible (go test ./...).
- [x] [LA-301] Pinguin notification scheduling disabled so feedback mailers send immediately; service/model regressions assert scheduled timestamps are cleared and retries run without delays (go test ./..., go test ./tools/pinguin/...).
- [x] [LA-101] Auto logout controls now live in the settings modal with persisted durations and test coverage for disabling and customizing the idle timers (go test ./internal/httpapi -run TestDashboardSettingsAutoLogoutConfiguration -count=1).
- [x] [LA-100] Dashboard header exposes Settings modal with overlay dismissal; integration test ensures modal hides via bootstrap instance (go test ./internal/httpapi -run TestDashboardSettingsModalOpensAndDismissesViaBackdrop -count=1).
