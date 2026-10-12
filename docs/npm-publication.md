# npm Publication Credentials

## Canonical Input

LoopAware stores its npm publication credential in the ignored `configs/.env.loopaware` file.
The exact assignment name is `NPM_API_KEY`.
The `react-native-client` resource declares `@loopaware/react-native` at `https://registry.npmjs.org/`.
The manifest owner, `loopaware`, selects `configs/.env.loopaware`.

Keep the real value in that private file.
Preserve existing assignments when you prepare or edit the private file.
Do not put it in a README, issue, command argument, or tracked config file.
The tracked `configs/.env.loopaware.example` file documents the name with an unusable value.
Do not copy or load the example as an operational input.

Gateway reads `NPM_API_KEY` directly from the selected application file during npm publication.
It removes inherited npm authentication overrides before it starts npm.
It supplies the credential through standard input as npm user configuration.
The credential does not need an export from the operator shell.

These inputs do not select the npm publication credential:

- `NPM_TOKEN` or `NODE_AUTH_TOKEN` in the operator shell.
- A token from `npm login` or the operator's `.npmrc` file.
- `.mprlab/deploy/.env`, which supplies deployment values.
- `configs/.env.loopaware.computercat`, which supplies the ComputerCat service configuration.

Keep one canonical assignment. Do not add a second name or another credential file.

## Loading And Authentication

Credential loading and registry authentication are separate results.
A nonempty local value proves presence, not registry acceptance or package publication authority.
Gateway reports a private-input error if the canonical file or required assignment is absent or invalid.
An npm HTTP response occurs after this local input check.

Gateway v6.0.0 restored `npm whoami` and account-access probes before publication.
The earlier B058 correction removed account probes from the application publication process.
An HTTP 401 from `whoami` does not mean that Gateway missed the local file.

The Gateway source correction for B650 removes identity, collaborator, organization, and package-settings probes.
Its preflight validates the canonical local input and reports `credential: loaded`.
This result does not claim registry authentication or publication authority.
The actual `npm publish` request establishes authority for an absent package version.
Gateway preserves the native npm authentication and permission errors.
Such an error can occur after other resource publication effects.

Public package metadata, `npm ping`, and `npm publish --dry-run` do not prove package write authority.
An account's collaborator access also does not prove the supplied granular token's package permissions.
An existing exact package version can pass artifact verification without a new upload.
This result does not prove that the current token can publish.

Examine the selected input and Gateway version before you change the credential.
Do not replace a token solely because a command with another authentication source fails.
Use the normal lifecycle after credential correction.
Do not claim successful publication from local input validation or a read-only check.

npm distinguishes package permissions from organization permissions.
A stage-only token cannot publish a version directly.
Current npm rules also restrict account and package governance actions for tokens that bypass 2FA.
An older package-settings check is not a substitute for the current publication contract.
See [npm access tokens](https://docs.npmjs.com/about-access-tokens/) and
[token creation](https://docs.npmjs.com/creating-and-viewing-access-tokens/).

## Observed Result On October 12, 2026

The local canonical file contained a nonempty `NPM_API_KEY`.
The selected Gateway v6.0.0 source reads that assignment and passes it to npm through standard input.
A diagnostic with that stored value reproduced HTTP 401 from `npm whoami`.
No credential value was displayed or changed.

`npm ping` succeeded with both the stored value and a deliberately invalid diagnostic token.
The public package-status query also succeeded.
These results do not establish that the stored token can publish the package.
The cause of the identity endpoint rejection remains unconfirmed.
The source correction removes the unrelated account-access requirement.
It does not establish that the stored credential can publish.
The installed Gateway v6.0.0 runtime still contains the account probes.
The corrected Gateway runtime must be available and installed before another operational attempt.

## Application Lifecycle

Use the repository lifecycle from the primary checkout:

```bash
make release && make publish && make deploy
```

Release prepares the artifacts. Publication exposes the prepared artifacts. Deployment activates the published version.
Credential repair does not authorize publication or deployment.
Keep the existing release record when you diagnose a publication failure.
