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
10. Mark an issue as done ([X])in the NOTES.md after the issue is fixed: New and existing tests are passing without regressions
11. Commit the changes and push to the remote.

Do not work on all issues at once. Work at one issue at a time sequntially.

Leave Features, BugFixes, Improvements, Maintenance sections empty when all fixes are implemented but don't delete the sections themselves.

## Issues

## Features

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

## Improvements

- [x] [LA-24] favicon retrieval shall be expressed as task that works asynchronously. once favicon is retrieved, it is cahced (saved in the db) and served from the DB.
- [x] [LA-25] favicon can be retrieved from inline embeddings in the sites looking for  `<link rel="icon"` and respecting the type (e.g. https://loopaware.mprlab.com has type="image/svg+xml" )
- [x] [LA-26] The inline icons are not fetched/displayed. https://loopaware.mprlab.com defines an inline favicon but there is no favicon in Loopaware Sites panel after defining https://loopaware.mprlab.com site. Prepare integration tests that run against https://loopaware.mprlab.com and ensure that the icon is extracted and displayed
- [x] [LA-29] Move the site registration date to the same row as Owner email (it is currently in a row below it). Add `Registered at:` prefix for the date. Remove the time.
- [x] [LA-37] Add deatils to the copy text. It is barebone now. analyze the functionality, check readme and PRD and consider usefullness for the end user
- [x] [LA-38] There is no theme switch on the landing page, and the components seem to belong to different themes.
- [x] [LA-39] Add logo of the Loopaware to the header
- [x] [LA-41] Do not display 0 for Feedback messages. Only display the total number if it's larger than 0
- [ ] [LA-48] The logo in the header shall be larger and better visible
- [x] [LA-49] Remove the login button from the hero page. Only leave login button in the header

## BugFixes

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
- [ ] [LA-46] The header on the landing page should stick to the top of the page.
- [ ] [LA-47] Clicking on the logo shall not do anything (it refreshes the page now).
- [ ] [LA-50] Add logo to the LoopAware Dashboard header (on the left of the word "LoopAware" )
- [ ] [LA-51] The choice of the theme on the landing page and the dashboard should be independent


## Maintenance
