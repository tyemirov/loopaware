# npm Publication Credentials

## Canonical Input

The npm publication input is `NPM_API_KEY`.
The `react-native-client` resource declares `@loopaware/react-native` at `https://registry.npmjs.org/`.
The selected manifest declares `.mprlab/deploy/.env` as its `environment_file`.

The lifecycle Make targets export `configs/.env.loopaware` before Gateway captures the selected manifest.
Gateway preserves a process value when the selected environment file has the same assignment.
Thus the exported `NPM_API_KEY` supplies this input for the lifecycle Make targets.
Gateway captures the resolved input once. The npm publisher reads that capture.
It does not read another credential file after capture.

Keep the real value in the existing private file.
Preserve other assignments when you edit that file.
Keep the value outside tracked files and command arguments.
The tracked `configs/.env.loopaware.example` file documents the name with an unusable value.
Keep that example separate from operational inputs.

Gateway removes inherited npm authentication overrides before it starts npm.
It supplies the captured credential through standard input as npm user configuration.
`NPM_TOKEN`, `NODE_AUTH_TOKEN`, `npm login`, and the operator's `.npmrc` do not select this publication input.
Keep the existing input name and credential location.

## Loading And Authentication

Credential loading and registry authentication are separate results.
A nonempty local value proves presence, not registry acceptance or package publication authority.
Gateway reports a private-input error if the required captured assignment is absent or invalid.
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
A failed lifecycle stage removes local pipeline receipts under the current Gateway contract.
Use a full canonical lifecycle after the failure is corrected.

## npm Version Verification

npm can accept an archive with HTTP 202 before its version becomes available to `npm view`.
During this interval, the version read can return `E404`.
An accepted upload does not prove that publication is completed.

LoopAware declares a separate policy for `app_publish.npm.version_verify` in `.mprlab/deploy/resources.yml`.
Its finite `attempt_limit: 1443` covers the four-hour publication time limit with the declared intervals.
Gateway rejects `attempt_limit: 0` at the lifecycle manifest boundary.

Gateway continues until the version becomes available or the remaining four-hour publication time limit ends.
A shorter parent time limit stops the read operation earlier.
The intervals remain 2, 5, and then 10 seconds between attempts.
Terminal errors, including `E401` and `E403`, stop the operation immediately.

With immediate responses, attempt 1442 occurs at 14,397 seconds and attempt 1443 would occur at 14,407 seconds.
The four-hour time limit stops observation before attempt 1443.
Slower responses can cause the time limit to stop observation earlier.

The archive read uses the same finite policy as the version check.
The initial version read and upload retain their existing attempt limits.
Successful publication still requires the exact version, tag, and archive checks.

On October 13, 2026, npm accepted version `1.1.8`, but four verification attempts stopped at 23:26:38 PDT.
npm made the version available at 23:27:36 PDT.
The public archive matched the submitted archive.

To do a test of the deployment policy, select one installed Gateway executable:

```bash
MPRLAB_GATEWAY_EXECUTABLE=/absolute/runtime/bin/mprlab-gateway make deployment-npm-visibility-check
```

The test uses that selected executable without another runtime selection.
The test requires `uv`. It installs isolated Python test dependencies.
It first calls `app-plan-release` on an independent Git fixture with the unchanged application manifest and a local origin.
The controller must accept the manifest and capture the source before the injected Ansible child starts.
The child reports its arguments instead of executing the Ansible plan.
This check proves controller acceptance, not execution of the full plan.
The negative case requires rejection of a zero attempt limit before child execution.

The behavior tests use the installed Gateway action, policy resolver, and retry CLI with a controlled clock and provider responses.
They verify delayed success, time limits, terminal errors, and the attempt limits for other operations.
They do not read credentials, upload packages, or contact npm.
All test commands use `CI=1` and closed standard input.

The tests belong in `tests/deployment`.
Gateway permits only the tracked `resources.yml` beneath `.mprlab/deploy`.
Do not put tests, scripts, or other tracked files in that reserved directory.
The envelope tests execute the installed surface reader and assertion through real Ansible.
They check the actual Git index and reject an additional tracked file in an independent fixture.
Stage file moves before this check because Gateway reads tracked paths from the Git index.
These checks prove envelope validation, not execution of the full lifecycle.

The canonical `ci-mobile` target includes this deployment integration test.
The hosted Mobile job installs the selected Gateway and `uv==0.12.19` before this target.
The other CI jobs do not require these tools.

## npm Archive Visibility

The `app_publish.npm.archive_read` operation checks the exact package archive after the version and tag checks.
It uses the version-check intervals and `attempt_limit: 1443` within the unchanged four-hour publication time limit.
A shorter parent time limit still stops the operation earlier.
Terminal authentication and permission errors stop immediately.
The initial version read and upload keep their previous policies.

On October 27, 2026, publication and version checks passed before an archive read returned `ETARGET` at 20:45:34 UTC.
Public metadata and the archive returned HTTP 200 at 20:46:42 UTC.
These observations do not establish when archive visibility recovered.
The Gateway correction for B685 classifies the archive-specific `ETARGET` as retryable.
B147 qualification of that correction requires the published, installed Gateway release that contains it.

The public policy tests use controlled time and provider responses with the real installed Gateway.
They verify archive recovery after four attempts, finite exhaustion, shorter parent limits, and terminal errors.
They also verify that the initial version read and upload retain their previous policies.
