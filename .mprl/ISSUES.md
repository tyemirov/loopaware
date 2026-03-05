# ISSUES

Working backlog for this repository. Keep it current and small. Use @issues-md-format.md for the canonical format.

- Status markers: `[ ]` open, `[!]` blocked (must include a `Blocked:` line), `[x]` closed.
- Hygiene: once a closed issue's consequences are reflected in code/tests and in user-facing docs, remove the entry from this file. Git history remains the record. (Recurring runbooks below are the exception: keep them open.)

## BugFixes

- [ ] [B001] (P0) Audit and optimize SQL queries for performance and security in statistics and visit tracking.
  ### Summary
  The current implementation of site statistics and visit tracking relies on SQL queries that may become performance bottlenecks as the volume of data grows. Specifically, several analytical endpoints fetch large amounts of data into memory for post-processing rather than utilizing database-level aggregation. This issue tracks the audit and optimization of these queries, the addition of necessary composite indexes, and a verification of security best practices.
  
  ### Analysis
  *   **In-Memory Aggregation Bottlenecks:** Both `VisitAttribution` and `VisitEngagement` in `internal/api/site_stats.go` fetch all visits for a site (filtered by `is_bot` and optionally a date range) into memory to calculate distributions and breakdowns. For sites with high traffic, this will lead to excessive memory consumption, slow response times, and potential OOM (Out of Memory) crashes. These should be refactored to use SQL `GROUP BY` and aggregate functions where possible.
  *   **Missing Composite Indexes:** Many queries filter by `site_id`, `is_bot`, and `occurred_at` simultaneously. While these fields have individual indexes, SQLite would benefit from composite indexes like `(site_id, is_bot, occurred_at)` to satisfy these queries without additional filtering or sorting overhead.
  *   **SQL Security:** While GORM generally handles parameter binding correctly, an audit of any raw SQL fragments (e.g., `topPagesSelectStatement` in `site_stats.go`) is required to ensure no unsanitized inputs are ever concatenated into query strings.
  *   **Schema Utilization:** The project includes a `SiteVisitRollup` model, but it is not fully utilized across all statistics endpoints. We should investigate if pre-aggregated data can replace real-time scans for the `VisitTrend` and `VisitEngagement` metrics.
  
  ### Deliverables
  *   **Performance Optimization:**
      *   Refactor `VisitAttribution` to perform source/medium/campaign extraction using SQL string functions or by processing visits in smaller, manageable chunks.
      *   Refactor `VisitEngagement` to use SQL aggregation for metrics like `ReturningVisitorCount` and `AveragePagesPerVisitor` instead of loading all `visitor_id` values.
      *   Introduce composite indexes in `internal/model/visit.go` for optimized filtering: `(site_id, is_bot, occurred_at)` and `(site_id, is_bot, visitor_id)`.
  *   **Security Audit:**
      *   Verify all `database.Raw` and `database.Exec` calls, as well as raw SQL fragments in GORM `Select` or `Group` clauses, to ensure they use parameter binding.
      *   Ensure that all user-provided strings used in queries (like `siteID` or filter parameters) are validated or sanitized before execution.
  *   **Validation:**
      *   Execute `make test` and `make test-unit` to ensure no regressions in statistics calculation.
      *   Optionally provide a benchmarking report showing the performance impact of composite indexes on tables with 100k+ rows.


## Improvements

## Maintenance

## Features

## Planning
*do not implement yet*

