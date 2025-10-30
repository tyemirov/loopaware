# ISSUES (Append-only Log)

Entries record newly discovered requests or changes, with their outcomes. No instructive content lives here. Read @NOTES.md for the process to follow when fixing issues.

## Features (100-199) 

## Improvements (200-299)_

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
    4. Each item in the  Build by Marco Polo Research Lab shall have links opening a new page.
- [ ] [LA-203] Remove the theme switch from a user menu under the avatar. 
    1. Use the same alpine ui footer component as other pages (but style it with current color palette used in dashboard)
    2. Remove user's specific light/dark theme switch

## BugFixes (300-399)

- [x] [LA-300] When logged in with the dark theme the dashboard theme is light, when logged in from the light theme, the dashboard theme is dar, find the bug and fix it
- [ ] [LA-301] The logout functionality behaviour: display a message after 60 seconds of inactivity. The message should match the theme of the page. Log out after 120 seconds of inactivity (same as +60 seconds since being displayed)
- [x] [LA-302] LoopAware server exits at startup complaining about missing `pinguin-auth-token` even when running with default docker compose. Resolved by requiring environment-provided bearer token and mirroring `GRPC_AUTH_TOKEN` fallback (go test ./...).

## Maintenance (400-499)

## Planning (do not work on these, not ready)
- [x] [LA-300] Dashboard theme now honors the latest public selection; regression integration test ensures public preference overrides stale dashboard storage (go test ./...).

## BugFixes (300-399) — Resolution Log
- [x] [LA-300] Dashboard theme now honors the latest public selection; regression integration test ensures public preference overrides stale dashboard storage (go test ./...).
## Improvements (200-299) — Resolution Log
- [x] [LA-201] Theme switch now lives in the footer beside the Built by Marco Polo branding; public landing/privacy tests enforce placement (go test ./...).
- [x] [LA-200] Added Pinguin-backed notifications for feedback submissions and surfaced delivery statuses across API and dashboard (go test ./...).
- [x] [LA-202] Footer now rendered by shared Alpine component from mpr-ui; templates load CDN module and tests confirm config payload & markup (go test ./...).
