# Migrations

## LA-60: Unified Owner Assignment

- All authenticated dashboard roles can now create sites with any valid owner email address; the system continues to
  record the authenticated creator in `creator_email`.
- No schema changes are required. Existing sites already contain the necessary fields; verify that historical records
  have `creator_email` populated before relying on creator-based scoping.
