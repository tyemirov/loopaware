# Notes

## Rules of engagement

Review the NOTES.md. Make a plan for autonomously fixing every item under Features, BugFixes, Improvements, Maintenance. Ensure no regressions. Ensure adding tests. Lean into integration tests. Fix every issue. Document the changes.

Fix issues one by one. 
1. Create a new git branch with descriptive name
2. Describe an issue through tests. Ensure that the tests are comprehensive and failing to begin with.
3. Fix the issue
4. Rerun the tests
5. Repeat 2-4 untill the issue is fixed and comprehensive tests are passing
6. Write a nice comprehensive commit message AFTER EACH issue is fixed and tested and covered with tests.
7. Optional: update the README in case the changes warrant updated documentation
8. Optional: ipdate the PRD in case the changes warrant updated product requirements
9. Optional: update the code examples in case the changes warrant updated code examples

Do not work on all issues at once. Work at one issue at a time sequntially. 

1. Remove an issue from the NOTES.md after the issue is fixed: New and existing tests are passing without regressions
2. Commit the changes and push to the remote.

Leave Features, BugFixes, Improvements, Maintenance sections empty when all fixes are implemented but don't delete the sections themselves.

## Issues

## Features

## Improvements

- [ ] [LA-24] favicon retrieval shall be expressed as task that works asynchronously. once favicon is retrieved, it is cahced (saved in the db) and served from the DB.
- [ ] [LA-25] favicon can be retrieved from inline embeddings in the sites looking for  `<link rel="icon"` and respecting the type (e.g. https://loopaware.mprlab.com has type="image/svg+xml" )

## BugFixes

- [x] [LA-23] the header of the table in Feedback messages panel doesnt respect the theme swithc and stays in light theme. it shall respect the theme switch

## Maintenance
