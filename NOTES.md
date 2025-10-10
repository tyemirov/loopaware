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

- [ ] [LA-13] Add UI validation help explaining what are the fields required to register the web site 
1. Allowed Origin (protocol, URN, port etc). Add a UX-based validation in case it's incorrect and display a helpful message (red, in the panel title). 
2. Owner email. Add a UX-based validation in case it's incorrect and display a helpful message (red, in the panel title). 
3. site name. 
Have a small question mark in a circle icon, as typical, and display a popup/tooltip when pressed

- [ ] [LA-14] Remove the need of uuid to identify the site. The site will be served based on the remote origin and matched with the registered site. Then the widget becomes simplified and is identical for any site.

- [ ] [LA-15] Minimize the submit dialog after submission. Display the message and then disappear it after a few seconds.

## Improvements

- [ ] [LA-12] The avatar route is probably unnessary, we can just replace the image if we have it in the template file rather than having a special rout to serve it

- [ ] [LA-16] Add mailto: prefix to the email displayed in the feedback panel

- [ ] [LA-17] Retrieve and demonstrate a favicon of the site next to the sie name in the sites panel

## BugFixes

## Maintenance
