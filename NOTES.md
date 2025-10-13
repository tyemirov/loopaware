# Notes

## Role

You are a staff level full stack engineer. Your task is to **re-evaluate and refactor the LoopAware repository** according to the coding standards already written in **AGENTS.md**.

## Context

* AGENTS.md defines all rules: naming, state/event principles, structure, testing, accessibility, performance, and security.
* The repo uses Alpine.js, CDN scripts only, no bundlers.
* Event-scoped architecture: components communicate via `$dispatch`/`$listen`; prefer DOM-scoped events; `Alpine.store` only for true shared domain state.
* The backend uses Go language ecosystem

## Your tasks

1. **Read AGENTS.md first** → treat it as the *authoritative style guide*.
2. **Scan the codebase** → identify violations (inline handlers, globals, duplicated strings, lack of constants, cross-component state leakage, etc.).
3. **Generate PLAN.md** → bullet list of problems and refactors needed, scoped by file. PLAN.md is a part of PR metadata. It's a transient document outlining the work on a given issue.
4. **Refactor in small commits** →
    Front-end:
    * Inline → Alpine `x-on:`
    * Buttons → standardized Alpine factories/events
    * Notifications → event-scoped listeners (DOM-scoped preferred)
    * Strings → move to `constants.js`
    * Utilities → extract into `/js/utils/`
    * Composition → normalize `/js/app.js` as Alpine composition root
    Backend:
    * Use "object-oreinted" stye of functions attached to structs
    * Prioritize data-driven solutions over imperative approach
    * Design and use shared components
5. **Tests** → Add/adjust Puppeteer tests for key flows (button → event → notification; cross-panel isolation). Prioritize end-2-end and integration tests.
6. **Docs** → Update README and MIGRATION.md with new event contracts, removed globals, and developer instructions.
7. **Timeouts**  Set a timer before running any CLI command, tests, build, git etc. If an operation takes unreasonably long without producing an output, abort it and consider a diffeernt approach. Prepend all CLI invocations with `timeout <N>s` command.

## Output requirements

* Always follow AGENTS.md rules (do not restate them, do not invent new ones).
* Output a **PLAN.md** first, then refactor step-by-step.
* Only modify necessary files.
* Descriptive identifiers, no single-letter names.
* End with a short summary of changed files and new event contracts.

**Begin by reading AGENTS.md and generating PLAN.md now.**

## Rules of engagement

Review the NOTES.md. Make a plan for autonomously fixing every item under Features, BugFixes, Improvements, Maintenance. Ensure no regressions. Ensure adding tests. Lean into integration tests. Fix every issue. Document the changes.

Fix issues one by one, working sequentially. 
1. Create a new git bracnh with descriptive name, for example `feature/LA-56-widget-defer` or `bugfix/LA-11-alpine-rehydration`. Use the taxonomy of issues as prefixes: improvement/, feature/, bugfix/, maintenace/, issue ID and a short descriptive. Respect the name limits.
2. Describe an issue through tests. 
2a. Ensure that the tests are comprehensive and failing to begin with. 
2b. Ensure AGENTS.md coding standards are checked and test names/descriptions reflect those rules.
3. Fix the issue
4. Rerun the tests
5. Repeat pp 2-4 untill the issue is fixed: 
5a. old and new comprehensive tests are passing
5b. Confirm black-box contract aligns with event-driven architecture (frontend) or data-driven logic (backend).
5c. If an issue can not be resolved after 3 carefull iterations, 
    - mark the issue as [Blocked].
    - document the reason for the bockage.
    - commit the changes into a separate branch called "blocked/<issue-id>".
    - work on the next issue from the divergence point of the previous issue.
6. Write a nice comprehensive commit message AFTER EACH issue is fixed and tested and covered with tests.
7. Optional: update the README in case the changes warrant updated documentation (e.g. have user-facing consequences)
8. Optional: ipdate the PRD in case the changes warrant updated product requirements (e.g. change product undestanding)
9. Optional: update the code examples in case the changes warrant updated code examples
10. Mark an issue as done ([X])in the NOTES.md after the issue is fixed: New and existing tests are passing without regressions
11. Commit and push the changes to the remote branch.
12. Repeat till all issues are fixed, and commits abd branches are stacked up (one starts from another).

Do not work on all issues at once. Work at one issue at a time sequntially.

Leave Features, BugFixes, Improvements, Maintenance sections empty when all fixes are implemented but don't delete the sections themselves.

## Issues

### Features

- [x] [LA-27] Design, write a copy and add a landing page at / root, with links pointing to /app. Introduce favicon, and leverage an ability of GAuss to take a login page from the app. The landing page shall be the one we feed into GAuss as a login page and that will initiate the login flow.

- [x] [LA-28] In the footer, clicking on the Marco Polo Recearch Lab in "Built by Marco Polo Recearch Lab" should display a stacked dropdown (drop up as it will always point up):
    - [Marco Polo Recearch Lab](https://mprlab.com)
    - [Gravity Notes](https://gravity.mprlab.com)
    - [LoopAware](https://loopaware.mprlab.com)
    - [Allergy Wheel](https://allergy.mprlab.com)
    - [Social Threader](https://threader.mprlab.com)
    - [RSVP](https://rsvp.mprlab.com)
    - [Countdown Calendar](https://countdown.mprlab.com)
    - [LLM Crossword](https://llm-crossword.mprlab.com)
    - [Prompt Bubbles](https://prompts.mprlab.com)
    - [Wallpapers](https://wallpapers.mprlab.com)

    Make the footer independent so that I could reuse it as a component in other projects


### Improvements

- [x] [LA-24] favicon retrieval shall be expressed as task that works asynchronously. once favicon is retrieved, it is cahced (saved in the db) and served from the DB.
- [x] [LA-25] favicon can be retrieved from inline embeddings in the sites looking for  `<link rel="icon"` and respecting the type (e.g. https://loopaware.mprlab.com has type="image/svg+xml" )
- [x] [LA-26] The inline icons are not fetched/displayed. https://loopaware.mprlab.com defines an inline favicon but there is no favicon in Loopaware Sites panel after defining https://loopaware.mprlab.com site. Prepare integration tests that run against https://loopaware.mprlab.com and ensure that the icon is extracted and displayed
- [x] [LA-29] Move the site registration date to the same row as Owner email (it is currently in a row below it). Add `Registered at:` prefix for the date. Remove the time.
- [x] [LA-37] Add deatils to the copy text. It is barebone now. analyze the functionality, check readme and PRD and consider usefullness for the end user
- [x] [LA-38] There is no theme switch on the landing page, and the components seem to belong to different themes.
- [x] [LA-39] Add logo of the Loopaware to the header
- [x] [LA-41] Do not display 0 for Feedback messages. Only display the total number if it's larger than 0
- [x] [LA-48] The logo in the header shall be larger and better visible
- [x] [LA-49] Remove the login button from the hero page. Only leave login button in the header
- [x] [LA-52] Remove the square around the logo for both the landing page and the dashboard. The logo shall be transparent. Increase the size of the logo.
- [x] [LA-53] Slim down the header and the footer
- [x] [LA-54] Add branding to the widget, saying "Bulit by Marco Polo Research Lab" with a link to https://mprlab.com. Have it in small letters under the widget.
- [x] [LA-56] The endpoint `api/me` shall return a JSON payload including user email, name and avatar. The rest of the system should be using this information when displaying user details. The login shall ba saving/updating this information. It must be a protected endpoint so that only the logged in user could get the information.
- [x] [LA-59] Define and surface descriptive error messages for the end users: when site already exists the message should say so instead of a generic "forbidden" etc
- [ ] [LA-61] Implement task based subsystem that performs non-immediate tasks such as retrieving sites favicons. The task shall be triggered using an internal schedule: check and update favicons every 24 hours.
- [ ] [LA-62] Schedule an immediate task execution for favicon retrieval on site creation or update from the user. Implement a mechanism (SSE?) to inform the site that the favicon must be retrieved from the backend in case we got a new or updated favicon. dont do anything if the favicon hasnt changed

### BugFixes

- [x] [LA-23] the header of the table in Feedback messages panel doesnt respect the theme swithc and stays in light theme. it shall respect the theme switch
- [x] [LA-26] The inline icons are not fetched/displayed. https://loopaware.mprlab.com defines an inline favicon but there is no favicon in Loopaware Sites panel after defining https://loopaware.mprlab.com site. Prepare integration tests that run against https://loopaware.mprlab.com and ensure that the icon is extracted and displayed
- [x] [LA-28] Instead of loopaware.mprlab.com use gravity.mprlab.com in the integration tests for inline favicon
- [x] [LA-30] "Site deleted" message in "Site details" panel had white background not respecting the theme. Ensure that all messaging respect the selected theme (light or dark)
- [x] [LA-31] "Site deleted" messaged in "Site details" panel never went away breaking the expected behavior of all messages to disappear after a timeout. Ensure messages disappear after a timeout.
- [x] [LA-32] The footer doesnt display the drop down with stacked links
- [x] [LA-33] The footer on the landing page is misalighed and Built by Marco Polo Resaerrch Lab is aligned to the left instead of the right
- [x] [LA-34] The footer on the landing page is giant, and shall have the same vertical height as in the dashboard
- [x] [LA-35] The cards on the landing page do not react on hover. the focus shall get to the card that the cursor is being hovered upon, and the card shall get highlighted
- [x] [LA-36] Remove Open LoopAware button, There shall be one button Login, which, in case a user is not logged in, would redirect it to goolge flow, and in case the user is logged in, would send the user to the dashboard, I think such flow is implcit (e.g. I doubt we need to have any special checks).
- [x] [LA-40] Move the registration time of the site in the site details panel to the right, making it appear on the same ro as the "Owner email" field and under "Allowed origin" field
- [x] [LA-42] Logout shall be redirecting the user to the landing page. not back to Login screen
- [x] [LA-43] The landing page misses favicon. Use     `<link rel="icon" type="image/svg+xml" href="{{.FaviconDataURI}}" />`
- [x] [LA-44] The LoopAware logo on the landing page is incorrect. Either use the SVG from the code or ![alt text](internal/httpapi/templates/logo.png)
- [x] [LA-45] The header on the landing page should stick to the top of the page.
- [x] [LA-46] The header on the landing page should stick to the top of the page.
- [x] [LA-47] Clicking on the logo shall not do anything (it refreshes the page now).
- [x] [LA-50] Add logo to the LoopAware Dashboard header (on the left of the word "LoopAware" )
- [x] [LA-51] The choice of the theme on the landing page and the dashboard should be independent
- [x] [LA-52] When logged in as a user the field to enter the site email is missing. This field must be present. The difference between a User adn and Admin is that Admin can see ALL of the sites regardless of what user has created them. User can only see and edit their own sites.
- [X] [LA-55] The sites created by the same user are not displayed when a user changes roles. All sites created by the same user regardless of its role must be displayed when the user logs in. User and admin designation must have an abstraction called role. The role has an impact on the scope. Admin role grants an ability to view all sites regardless of the user who has created them plus everything that a user can do. User role grants the permissions to add, edit, and delete the sites that the user has created.
- [x] [LA-57] In case we have records with no information about what user has created them, we shall make a one-time data migration and update such records for the creator to be temirov@gmail.com. After that we shall be sure to separate the user who has created the records, using user's email from login, and the field called owner email, which is just an information we store for now
- [x] [LA-58] When pressing tab in the Site details, the focus shall be moving between the three input fields cyclically and not to the tooltips.
- [x] [LA-60] When trying to create a site I am getting an error "Only administrator can assign different site owners". I dont understand this error. AS  user I am able to create any sites, and assign any emails to the owners of the site, same as administrator. The only difference between a User and Administrator is the number of sites that the administrator can see. Write integration tests that verify that a user can perform ALL and every action an administratort cvan perform. ensure we have a test that verifies that administrator can see sites not created by them, and the user can not see sites not created by them, ensure that we have an association between a logged user, who is a creator, and sites being created

### Maintenance
