# ISSUES

**Append-only section-based log**

Entries record newly discovered requests or changes, with their outcomes. No instructive content lives here. Read @NOTES.md for the process to follow when fixing issues.

Read @AGENTS.md, @AGENTS.DOCKER.md, @AGENTS.FRONTEND.md, @AGENTS.GIT.md, @POLICY.md, @NOTES.md, @README.md and @ISSUES.md. Start working on open issues. Work autonomously and stack up PRs.

Each issue is formatted as `- [ ] [LA-<number>]`. When resolved it becomes `- [x] [LA-<number>]`

## Features (110-199)

- [x] [LA-111] Allow multiple origins for subscribe widgets, e.g. a single subscribe widget can be embedded in multiple sites, not all of them matching the original url, such as gravity.mprlab.com needs to be able to be retreieved and function from both https://mprlab.com and http://localcost:8080 — implemented multi-origin support for site `allowed_origin` values (space/comma-separated list), extended backend origin checks and dashboard validation, and updated README to document the behavior.

## Improvements (206-299)

## BugFixes (311-399)

- [ ] [LA-311] TestWidgetIntegrationSubmitsFeedback can time out under `make ci` race tests with a `context deadline exceeded` error from the headless browser harness; investigate and stabilize the widget integration test so `make ci` passes reliably.

## Maintenance (405-499)

- [x] [LA-403] Document subscribe integration in the @README.md — added subscribe.js snippet, REST endpoints, and dashboard description to README.md
- [x] [LA-403] Document pixel integration in the @README.md — added pixel.js snippet, REST endpoints, and traffic dashboard description to README.md

## Planning

**Do not work on these, not ready**
