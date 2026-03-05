# ISSUES

Working backlog for this repository. Keep it current and small. Use @issues-md-format.md for the canonical format.

- Status markers: `[ ]` open, `[!]` blocked (must include a `Blocked:` line), `[x]` closed.
- Hygiene: once a closed issue's consequences are reflected in code/tests and in user-facing docs, remove the entry from this file. Git history remains the record. (Recurring runbooks below are the exception: keep them open.)

## BugFixes

- [ ] [B001] (P0) Audit and harden SQL queries for security, correctness, and performance.
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


## Improvements

## Maintenance

## Features

## Planning
*do not implement yet*

