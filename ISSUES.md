# ISSUES (Append-only Log)

Entries record newly discovered requests or changes, with their outcomes. No instructive content lives here. Read @NOTES.md for the process to follow when fixing issues.

## Features (100-199) 

## Improvements (200-299)_

- [ ] [LA-201] Move theme switch to the footer, on the left of Built by Marco Polo
- [ ] [LA-200] Integrate with Pinguin service. Find the code under @tools/pinguin. Read documentation and understand the code. 
    Aceptance criteria are integration tests that verify:
    - When a feedback is received, send a message to the owner (not the registar). 
    - Have a column in the feedback messages table titled Delivery with values either "mailed" or "texted" or "no"
    The tests must confirm the cotract fulfillment at the boundaries (message sent and it matches expected payload, message received).
    In case bugs are discovered in Pinguin, or enhancements are needed in Pinguin, document them as an issue, and stop working before we fix Pinguin.

## BugFixes (300-399)

- [ ] [LA-300] When logged in with the dark theme the dashboard theme is light, when logged in from the light theme, the dashboard theme is dar, find the bug and fix it


## Maintenance (400-499)

## Planning (do not work on these, not ready)
- [x] [LA-300] Dashboard theme now honors the latest public selection; regression integration test ensures public preference overrides stale dashboard storage (go test ./...).

## BugFixes (300-399) — Resolution Log
- [x] [LA-300] Dashboard theme now honors the latest public selection; regression integration test ensures public preference overrides stale dashboard storage (go test ./...).
