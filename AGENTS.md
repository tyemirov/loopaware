# AGENTS.md

## Forward-Only Contract Discipline

This repository follows a forward-only, confident programming paradigm. This is a binding agent contract: no fallbacks, no backward compatibility, no legacy support, and no compatibility shims. Do not spend design or implementation effort on backward compatibility considerations except for explicit one-off data migrations into the current canonical contract.

Repeat for emphasis because this rule is binding: no fallbacks, no backward compatibility, no legacy compatibility. Delete or reject obsolete code paths, stale schemas, deprecated config, and old persisted shapes instead of preserving them through compatibility layers, dual reads/writes, aliases, or best-effort recovery.

One-off data migrations are allowed only when they move existing persisted data into the current schema in a bounded operation. After migration, remove the bridge and keep only the current contract.

## LOOPAWARE

LoopAware collects customer feedback through a lightweight widget, authenticates operators with Google, and offers a
role-aware dashboard for managing sites and messages. See README.md for details

## Document Roles

- AGENTS.md: Read-only workflow + behavior playbook maintained by leads. Agents never edit it during implementation cycles.
- ISSUES.md: Tracker for repository changes. Each entry records requested or completed repository work. Located at `.mprlab/ISSUES.md`.
- `.mprlab/<PLAN-ID>-PLAN.md`: Temporary execution plan. Use `.mprlab/PLANNING.md` for the plan ID.

### Document Precedence

- `POLICY.md` (`.mprlab/POLICY.md`) defines binding validation, error-handling, and "confident programming" rules.
- `AGENTS.md` (this file) defines repo-wide workflow, testing philosophy, and agent behavior; stack-specific AGENTS.* guides refine these rules for each technology.
- `AGENTS.*.md` files (in `.mprlab/`) never contradict `AGENTS.md` or `POLICY.md`; if guidance appears inconsistent, defer to `POLICY.md` first, then `AGENTS.md`, and treat the stack guide as a refinement.

### Issue Status Terms

- Resolved: Completed and verified; no further action (`[x]`).
- Unresolved: Needs decision and/or implementation (`[ ]`).
- Blocked: Requires an external dependency or policy decision (`[!]`); must include a `Blocked:` explanation in the issue body.

### Issue Tracker Scope

- `.mprlab/ISSUES.md` tracks changes to source code, tests, config, and technical documentation.
- Close an issue after its requested repository changes are completed and applicable validation passes.
- Do not use release, publication, deployment, or production availability as issue completion conditions.
- If one of these operations finds a necessary repository change, file a dependent issue.

### Validation & Confidence Policy

All rules for validation, error handling, invariants, and "confident programming" (no defensive checks, edge-only validation, smart constructors, CI gates) are defined in `.mprlab/POLICY.md`. Treat that document as binding; this file does not restate them.

### Build & Test Commands

- Use the repository `Makefile` for local automation. Invoke `make test`, `make lint`, `make ci`, or other documented targets instead of running ad-hoc tool commands.
- `make test` runs the canonical test suite for the active stack.
- `make lint` enforces linting rules before code review.
- `make ci` mirrors the CI workflow and should pass locally before opening a PR.

## Workflow

Operational playbook for working in this repository. Use it to coordinate planning, execution, and delivery. Code style, stack-specific rules, and tooling details remain in the `.mprlab/AGENTS.*` documents; this section focuses purely on day-to-day process.

### Authoritative References

- `AGENTS.md` + per-stack guides (`.mprlab/AGENTS.*.md`) for coding standards.
- `.mprlab/POLICY.md` for validation/confident-programming rules.
- `.mprlab/AGENTS.GIT.md` for Git/GitHub workflow.
- `.mprlab/AGENTS.DOCKER.md` for container expectations.
- `.mprlab/issues-md-format.md` for the canonical ISSUES.md entry format specification.
- `README.md` for product context.

### Workflow Overview

1. Read `AGENTS.md` and the relevant stack guides in `.mprlab/`.
2. Review the backlog in `.mprlab/ISSUES.md`. Work through BugFixes, Improvements, Maintenance, and Features in this order. Planning contains future work. Do not implement Planning items.
3. For the active issue, read `.mprlab/PLANNING.md`. Make the execution plan that this contract specifies.
4. For application changes, use the initial validation result from `.mprlab/POLICY.md`.
5. Implement the requested change. Limit edits to necessary files plus required issue-document updates. Use the smallest applicable target during the change.
6. Complete the applicable validation after the last change.
7. Report what changed and any blockers.

### Completion Gate (Non-negotiable)

1. Requested file/documentation changes are implemented.
2. Any required issue status/notes updates are made in `.mprlab/ISSUES.md`.
3. Blockers are reported clearly when present.
4. The applicable validation after the last change passes.

### Testing & Tooling

- Use `Makefile` targets (`make test`, `make lint`, `make ci`) for local verification.
- Run stack-specific formatters only when the issue requires local validation output or explicit formatting changes.

### Git & Release Flow

- `master` is production. Work branches use taxonomy prefixes (`feature/`, `improvement/`, `bugfix/`, `maintenance/`, `blocked/`) outlined in `.mprlab/AGENTS.GIT.md`.
- Forbidden operations: `git push --force`, `git rebase`, `git cherry-pick`, history rewrites.

### Output Requirements

- Always follow AGENTS* rules; do not restate them in PRs.
- Begin every implementation with the execution plan that `.mprlab/PLANNING.md` specifies.
- Do not touch `AGENTS.md` during normal work; treat it as read-only guidance.
- `.mprlab/ISSUES.md` tracks issue status; mark items `[x]` with a concise resolution note once tests pass.
- Keep each execution plan untracked. If Git tracks a plan, remove it with `git filter-repo --path-glob '.mprlab/*-PLAN.md' --invert-paths`.
- Summaries at the end of each issue should list changed files.

### Pre-Finish Checklist

1. The execution plan for the active issue shows the final execution state.
2. `.mprlab/ISSUES.md` entry is marked `[x]` with the resolution note.
3. Requested implementation and documentation updates are complete.
4. Any blockers are documented with concrete failure context.
5. Provide a short summary plus next steps before moving to the next issue.

If any checklist item is incomplete, do not claim completion. Complete the missing step(s) first.

### Action Items Reminder

- Read guiding docs (`README.md`, `AGENTS*`, `.mprlab/POLICY.md`) before planning.
- Keep working sequentially through the backlog — never parallelize issues.
- Add missing issues to `.mprlab/ISSUES.md` if you discover new work while investigating; plan and resolve them in order.

### Testing Philosophy

- Testing follows an **inverted test pyramid**: heavy bias to high-value black-box integration and end-to-end tests that exercise external public APIs.
- We **strive for (approximately) 100% test coverage**, with CI enforcing an agreed threshold. If coverage drops, add scenarios at the public entry points; do not chase coverage with isolated unit tests.
- For the Go backend, tests run the real HTTP server and call real endpoints, capturing responses and asserting observable results — not internal functions.
- For the frontend, tests run the app and backing web server, drive flows through the browser or JSDOM, and assert against the rendered page, DOM state, events, and other user-visible behavior.
- Use focused unit tests for complex algorithms, calculations, and isolated logic when useful.
- Require integration coverage of public behavior for product acceptance.

## Tech Stack Guides

Stack-specific instructions live in `.mprlab/`. Apply the relevant guide alongside the shared policies above.

- Front-End (Browser ES Modules, vanilla JS): `.mprlab/AGENTS.FRONTEND.md`
- Backend (Go): `.mprlab/AGENTS.GO.md`
- Docker and containerization: `.mprlab/AGENTS.DOCKER.md`
- Git and version control workflow: `.mprlab/AGENTS.GIT.md`

<!-- BEGIN ISSUES.MD MANAGED ONBOARDING -->
## ISSUES.md repository workflow

ISSUES.md manages this repository through the current application contract.

- Use `.mprlab/ISSUES.md` as the repository issue tracker.
- Follow `.mprlab/issues-md-format.md` for issue syntax and identifiers.
- Use `.mprlab/runtime.yml` as the repository execution contract.
- Keep these required documents current through the ISSUES.md onboarding pull request.
<!-- END ISSUES.MD MANAGED ONBOARDING -->

<!-- BEGIN MPRLAB-GOVERNANCE -->
## MPR Lab Governance

Most workflow context files live under `.mprlab/`. The root `AGENTS.md` remains the repository entrypoint for agents.

Read these files before editing:

- `.mprlab/POLICY.md`: binding validation and confident-programming rules.
- `.mprlab/PLANNING.md`: durable planning contract.
- `.mprlab/AGENTS.DOCS.md`: ASD-STE100 documentation rules.
- `.mprlab/TERMINOLOGY.md`: approved repository technical terms.
- `.mprlab/issues-md-format.md`: issue tracker format and recurring identifier rules.
- `.mprlab/ISSUES.md`: active issue tracker.
- `.mprlab/AGENTS.GIT.md`: Git and pull request workflow.
- `.mprlab/AGENTS.API.md`: RESTful HTTP and gRPC API guidance.
- `.mprlab/AGENTS.GO.md`: Go guidance.
- `.mprlab/AGENTS.PY.md`: Python guidance.
- `.mprlab/AGENTS.MOBILE.md`: mobile client guidance.
- `.mprlab/AGENTS.MOBILE.EXPO.md`: Expo and React Native guidance.
- `.mprlab/AGENTS.FRONTEND.md`: browser frontend guidance.
- `.mprlab/AGENTS.DOCKER.md`: Docker and container guidance.

File permission modes are outside agent scope.
Never examine, validate, compare, require, change, or record a file permission mode.
Never use a file permission mode in acceptance, security, credential, execution, publication, deployment, or failure analysis.
The values `0600` and `7777` have no governance meaning.
This rule does not change service authorization or operation authority.

Do not create `.mprlab/AGENTS.md`. Scoped guidance belongs in `.mprlab/AGENTS.*.md` files.
If guidance conflicts, follow `.mprlab/POLICY.md` first, then root `AGENTS.md`, then the relevant scoped guide.
<!-- END MPRLAB-GOVERNANCE -->
