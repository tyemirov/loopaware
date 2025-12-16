# ISSUES (Open items)

This file lists open work only. Completed entries live in `issues.md/ISSUES_ARCHIVE.md`.

Each issue is formatted as `- [ ] [<ID>-<number>]`. This list is curated and may be reordered as priorities change.

Issue metadata fields:
- **Priority**: `P0` (blocking/urgent), `P1` (high), `P2` (normal), `P3` (nice-to-have)
- **Goal**: end-state outcome
- **Deliverable**: what ships (PR + tests, UI changes, docs)
- **Docs/Refs**: relevant repo docs and code entry points
- **Execution plan**: concrete implementation steps

## BugFixes (312–399)

- [ ] [LS-318] Theme toggle defaults and mapping are wrong (left = light, right = dark).
  Priority: P0
  Goal: The left toggle state represents light theme, the right state represents dark theme, and the initial UI state matches the applied theme.
  Deliverable: PR that fixes theme toggle behavior across landing, dashboard, and public pages; automated coverage updated.
  Docs/Refs:
  - `issues.md/AGENTS.FRONTEND.md`
  - `internal/httpapi/footer.go`
  - `internal/httpapi/templates/dashboard.tmpl`
  - `internal/httpapi/public_assets.go`
  Execution plan:
  - Identify the single source of truth for theme mode (document attribute + storage key).
  - Fix toggle UI state mapping (mpr-ui footer event + local storage + `data-bs-theme`).
  - Add/adjust integration assertions around theme attribute and toggle state.

- [ ] [LS-317] mpr-ui footer menu label “Built by Marco Polo Research Lab” is invisible.
  Priority: P1
  Goal: Footer label is visible in both light and dark themes and matches the mpr-ui demo styling.
  Deliverable: PR that removes/adjusts conflicting CSS and validates footer label visibility; screenshot-based evidence if needed.
  Docs/Refs:
  - `tools/mpr-ui/demo/index.html`
  - `internal/httpapi/templates/dashboard.tmpl`
  - `internal/httpapi/public_assets.go`
  Execution plan:
  - Compare LoopAware footer DOM/CSS against the mpr-ui demo.
  - Identify overriding selectors in LoopAware CSS (especially for footer text colors).
  - Adjust styles to avoid clobbering mpr-ui defaults; verify both themes.

## Improvements (210–299)

- [ ] [LA-213] Dashboard section tabs should span full width and split into 3 equal parts.
  Priority: P2
  Goal: Feedback/Subscriptions/Traffic tab buttons fill the available width and each takes exactly 1/3 of the row (responsive).
  Deliverable: PR adjusting tab markup/CSS and updating dashboard integration tests as needed.
  Docs/Refs:
  - `internal/httpapi/templates/dashboard.tmpl`
  - existing dashboard integration tests under `internal/httpapi/*integration*_test.go`
  Execution plan:
  - Update tab container layout to a 3-column equal-width grid or flex distribution.
  - Verify keyboard focus/active styles remain correct.
  - Update integration tests to assert correct tab visibility and interaction.

- [ ] [LA-214] Add “additional source origins” UX to the subscriber widget (add/remove inputs).
  Priority: P2
  Goal: Dashboard exposes a dedicated UI to enter extra allowed origins for the subscribe widget (separate from the site’s `allowed_origin`), with +/− controls.
  Deliverable: PR adding UI, persisting the new configuration, and updating the subscribe widget to enforce the combined origin set.
  Docs/Refs:
  - `internal/httpapi/templates/dashboard.tmpl`
  - `internal/httpapi/public.go` (origin validation helpers)
  - `internal/model/site.go` (if new persisted fields are added)
  Execution plan:
  - Decide data model: extend `Site` with `subscribe_allowed_origins` (or equivalent) and migrate.
  - Add dashboard editor UI with add/remove controls and validation.
  - Extend subscribe widget + backend origin checks to consult both site and subscribe-specific origins.
  - Add integration coverage for “extra origin accepted / unknown origin rejected”.

- [ ] [LA-215] Improve subscribe widget instructions (separate snippet and rendered form).
  Priority: P3
  Goal: Dashboard instructions clearly explain (a) the script snippet to embed and (b) what the rendered form looks like / where it appears.
  Deliverable: PR that updates dashboard copy and/or adds an in-dashboard preview of the subscribe form.
  Docs/Refs:
  - `README.md` “Embedding the subscribe form”
  - `internal/httpapi/templates/dashboard.tmpl`
  Execution plan:
  - Rewrite instruction copy to be action-oriented and unambiguous.
  - Add a small static preview block or link to the existing subscribe demo page.
  - Validate copy is consistent with current query parameters and behavior.

## Planning (do not implement yet)
