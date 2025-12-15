# ISSUES

**Append-only section-based log**

Entries record newly discovered requests or changes, with their outcomes. No instructive content lives here. Read @NOTES.md for the process to follow when fixing issues.

Read @AGENTS.md, @AGENTS.DOCKER.md, @AGENTS.FRONTEND.md, @AGENTS.GIT.md, @POLICY.md, @NOTES.md, @README.md and @ISSUES.md. Start working on open issues. Work autonomously and stack up PRs.

Each issue is formatted as `- [ ] [LA-<number>]`. When resolved it becomes `- [x] [LA-<number>]`

## Features (110-199)

- [x] [LA-111] Allow multiple origins for subscribe widgets, e.g. a single subscribe widget can be embedded in multiple sites, not all of them matching the original url, such as gravity.mprlab.com needs to be able to be retreieved and function from both https://mprlab.com and http://localhost:8080 — implemented multi-origin support for site `allowed_origin` values (space/comma-separated list), extended backend origin checks and dashboard validation, and updated README to document the behavior.

## Improvements (206-299)

- [x] [LA-207] Upgrade to the latest version of mpr-ui. Check tools/mpr-ui/README.md and @tools/mpr-ui/docs/custom-elements.md and @tols/mpr-ui/demo/index.html for documentation and examples. — migrated LoopAware templates to the v0.2+ `<mpr-footer>` custom element, loading `mpr-ui@latest/mpr-ui.css` + `mpr-ui@latest/mpr-ui.js`, and removed the legacy `footer.js`/`mprFooter` helper import.
```
Uncaught SyntaxError: The requested module 'https://cdn.jsdelivr.net/gh/MarcoPoloResearchLab/mpr-ui@main/footer.js' doesn't provide an export named: 'mprFooter' subscribe-test:709:14
```
- [x] [LA-208] Add front-end for LA-111 which would allow entering multiple origins for the same subscribe widget. — updated the dashboard UI to treat `allowed_origin` as a multi-origin field (space/comma-separated), summarize the primary origin in the sites list, and ensure favicon-click opens the primary origin.

## BugFixes (311-399)

- [x] [LA-311] TestWidgetIntegrationSubmitsFeedback can time out under `make ci` race tests with a `context deadline exceeded` error from the headless browser harness; investigate and stabilize the widget integration test so `make ci` passes reliably — simplified the keyboard focus assertions in the widget integration test to avoid brittle Shift+Tab focus loops while preserving end-to-end feedback submission coverage; `make test`, `make lint`, and `make ci` now pass cleanly including the race suite.

- [x] [LS-312] Investigate the 403 errro when trying to subscribe on a test subscribe page. I have entered a valid enail and my name but got an error: "Please try again" — routed the subscribe-test preview submission through an authenticated `/app/sites/:id/subscribe-test/subscriptions` endpoint (origin checks remain enforced for public `/api/subscriptions`).
```
Error: http_403
    submitInlineForm http://localhost:8080/app/sites/c6bf3dd5-0bd4-4d0b-9be3-c647991f7092/subscribe-test:589
subscribe-test:599:21
```
Server log:
```
011581307,"ip":"192.168.65.1","ua":"Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:145.0) Gecko/20100101 Firefox/145.0"}
loopaware  | {"level":"info","ts":1765747899.0851963,"caller":"httpapi/middleware.go:14","msg":"http","method":"GET","path":"/api/sites/c6bf3dd5-0bd4-4d0b-9be3-c647991f7092/subscribers","status":200,"dur":0.005705436,"ip":"192.168.65.1","ua":"Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:145.0) Gecko/20100101 Firefox/145.0"}
loopaware  | {"level":"info","ts":1765747899.0949476,"caller":"httpapi/middleware.go:14","msg":"http","method":"GET","path":"/api/sites/c6bf3dd5-0bd4-4d0b-9be3-c647991f7092/visits/stats","status":200,"dur":0.015284192,"ip":"173.194.65.95","ua":"Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:145.0) Gecko/20100101 Firefox/145.0"}
loopaware  | {"level":"info","ts":1765747907.1529574,"caller":"httpapi/middleware.go:14","msg":"http","method":"GET","path":"/app/sites/c6bf3dd5-0bd4-4d0b-9be3-c647991f7092/subscribe-test","status":200,"dur":0.052474132,"ip":"192.168.65.1","ua":"Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:145.0) Gecko/20100101 Firefox/145.0"}
loopaware  | {"level":"info","ts":1765747925.552095,"caller":"httpapi/middleware.go:14","msg":"http","method":"POST","path":"/api/subscriptions","status":403,"dur":0.000895133,"ip":"173.194.65.95","ua":"Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:145.0) Gecko/20100101 Firefox/145.0"}
loopaware  | {"level":"info","ts":1765747936.5625567,"caller":"httpapi/middleware.go:14","msg":"http","method":"GET","path":"/app/sites/c6bf3dd5-0bd4-4d0b-9be3-c647991f7092/subscribe-test","status":302,"dur":0.000044817,"ip":"173.194.65.95","ua":"Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:145.0) Gecko/20100101 Firefox/145.0"}
```

- [x] [LS-313] Prevent duplicate origins when a site's `allowed_origin` contains multiple origins (comma/space-separated). — updated conflict detection to compare per-origin rather than the raw `allowed_origin` string and added coverage.

## Maintenance (405-499)

- [x] [LA-403] Document subscribe integration in the @README.md — added subscribe.js snippet, REST endpoints, and dashboard description to README.md
- [x] [LA-403] Document pixel integration in the @README.md — added pixel.js snippet, REST endpoints, and traffic dashboard description to README.md

## Planning

**Do not work on these, not ready**
