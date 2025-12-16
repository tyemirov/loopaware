# ISSUES (Append-only section-based log)

Entries record newly discovered requests or changes, with their outcomes. No instructive content lives here. Read @NOTES.md for the process to follow when fixing issues.

Read @AGENTS.md, @ARCHITECTURE.md, @README.md, @PRD.md. Read @POLICY.md, PLANNING.md, @NOTES.md, and @ISSUES.md under issues.md/.  Start working on open issues. Work autonomously and stack up PRs. Prioritize bugfixes.

Each issue is formatted as `- [ ] [<ID>-<number>]`. When resolved it becomes `- [x] [<ID>-<number>]`.
## Features (113–199)

## Improvements (210–299)
- [ ] [LA-213] Make the tabs span wider so thay occupy all o their space and divide it in 3 equal parts ![alt text](<../image copy.png>)
- [ ] [LA-214] Add additional source origins UX to the subscriber widget ![alt text](image.png). Add an extra source origin section before the "Place this snippet on pages where you want visitors to subscribe." have + and - buttons to add input fields with extra allowed source origins.
- [ ] [LA-215] Improve instructions for the subscribe widget. copying the script probably won't be sufficient, so we shall have two pieces:
  1. <script defer src="http://localhost:8080/subscribe.js?site_id=12665b6e-78a2-421f-9149-04be800f6245"></script>
  2. The form that actually displays the subscribe fields

## BugFixes (312–399)
- [ ] [LS-317] the menu label “Built by Marco Polo Research Lab” is invisible. Consult the examples in @tools/mpr-ui/demo and be sure not to override any of the mpr-ui css with our own css ![alt text](<../image copy 2.png>)
- [ ] [LS-318] the site starts in the light switch toogle on the left which shall be light theme but it displays dark theme. FIx it and make the toggle on the left be light theme and the toggle on the right be dark theme ![alt text](<../image copy 2.png>)

## Maintenance (408–499)

### Recurring (close when done but do not remove)

- [ ] [LA-406] Cleanup:
  1. Review the completed issues and compare the code against the README.md and ARCHITECTURE.md files.
  2. Update the README.md and ARCHITECTURE.
  3. Clean up the completed issues.
  reconciled the README REST API table, subscription token routes, and dashboard feature list with the shipped behavior; expanded ARCHITECTURE.md with an overview of components and key flows.

- [ ] [LA-407] Polish:
1. Review each open issue
2. Add additional context: dependencies, documentation, execution plan, goal
3. Add priroity and deliverable. Reaarange and renumber issues as needed.

## Planning (do not implement yet)
