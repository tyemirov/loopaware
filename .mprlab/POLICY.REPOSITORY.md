# Repository Policy Sections

## Selected Manifest Contract

Apply this section when the task changes or validates a selected application manifest.

- Keep the selected application manifest versionless.
- Keep `owner`, `release`, and `lifecycle` as the required baseline fields.
- Use the fixed `artifacts`, `publications`, and `targets` catalogs.
- Use explicit identity lists and ordered command lists.
- Declare CI and execution policy in the selected manifest's `lifecycle` field.
- Apply the current iOS baseline below before the manifest preservation rules.
- After this cutover, accept each manifest that an earlier versionless gateway accepted.
- After this cutover, keep each accepted field name, type, requirement, and function.
- A field added to an existing shape must be optional.
- A field added to an existing shape must have one canonical default.
- Normalize that default before manifest identity calculation.
- Add a declaration kind only with one closed shape.
- Reject unknown fields and `schema_version`.

### Current iOS Baseline

The operator selected a fresh iOS contract on September 10, 2026.
Previous iOS adapter acceptance does not constrain this contract.

- Require `build_system: local` and a repository-relative `.mjs` path in `build.ios`.
- Build and sign iOS artifacts locally through the shared native builder.
- Publish the sealed IPA through the shared App Store Connect adapter.
- Reject iOS `.sh` adapters, Xcode Cloud requests, and cloud publication receipts.
- Do not add compatibility dispatch, obsolete aliases, or migration requirements for the removed iOS contract.
- Preserve the declared TestFlight or App Store destination and its export restriction.

This baseline does not authorize removal of stored releases, credentials, or deployment state.

## Static Website Hosting

Apply this section to deployment or publication work for a browser frontend.

- Use GitHub Pages as the production host for each deployable browser frontend.
- Declare the browser frontend with a `targets.github_pages.sites[]` declaration in `.mprlab/deploy/resources.yml`.
- Use `gh-pages` as the publication branch.
- The GitHub Pages repository can differ from the application repository.
- Keep API and service routes on hostnames that differ from the website hostname.
- Reserve the GitHub Pages domain and its `www` hostname for GitHub Pages.
- Treat a container as an artifact source only when its static output goes to GitHub Pages.
- Verify publication through the public website and `/.mprlab-release.json`.
- Run the Governor check after each selected manifest change and before each release, publish, or deploy operation.

## Gateway Lifecycle Execution

Apply this section when the repository uses the selected-manifest lifecycle or owns Gateway infrastructure operations.

- Use the installed `mprlab-gateway` runtime for application and infrastructure operations.
- Preserve `make release && make publish && make deploy` as the application operator command.
- Pass the selected application Git root through `--app-root`.
- Use `MPRLAB_GATEWAY_EXECUTABLE` for an explicit installed command.
- Preserve application source validation, release policy, resource ownership, private inputs, and sealed receipts.
- Keep each operation and its child calls on one captured runtime package and dependency environment.
- Keep Gateway inventory and private config under `MPRLAB_GATEWAY_OPERATOR_ROOT`.
- Use `$HOME/.config/mprlab-gateway` as the default operator root.
- Select an explicit version with `mprlab-gateway install --version <version>` to change the active runtime.
- Use Gateway source for repository CI, release preparation, and Gateway publication.
- Keep installation, application operations, and operator-controlled production activation as separate gates.

## Repository Guide Requirements

These guides retain repository-specific product or operational contracts.
Preserve those contracts during shared-governance updates. Apply current agent procedures from the core policy.

- Read and retain `.mprlab/AGENTS.APPLE.md` for its task-specific scope.
