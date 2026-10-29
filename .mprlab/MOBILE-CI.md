# Hosted Mobile Checks

The `Mobile checks` job uses Node.js 24, Python 3.13, and Temurin Java 21.
The job installs `uv==0.12.19` with Python's pip before the deployment policy tests.
The version is fixed to the qualified [official uv release](https://github.com/astral-sh/uv/releases/tag/0.12.19).
The canonical `ci-mobile` target includes `make deployment-npm-visibility-check`.
These tests use the selected physical Gateway executable and synthetic inputs.
The checked-in Gradle wrapper supplies Gradle for the generated signing check.
After Java setup, the job installs official Google `bundletool` 1.18.3 and verifies its version.
A dependency-free npm launcher forwards arguments to the published JAR through the selected Java runtime.
The job adds its installation directory through `GITHUB_PATH`. The download uses no private credential.
The pinned artifact comes from the [official Google release](https://github.com/google/bundletool/releases/tag/1.18.3).
The SDK checks Node, npm, Java, and bundletool before the controlled Gradle boundary.
This signing fixture stops before bundle compilation and does not require Android SDK build tools.
The job installs the published Gateway `v7.0.9` package for `linux-amd64` before `make ci-mobile`.
The installer uses the synthetic signing fixture manifest and synthetic inputs.
The application deployment manifest and private inputs stay unchanged.

## Release Access

Set the repository Actions secret `MPRLAB_GATEWAY_READ_TOKEN` before the hosted job runs.
Use a credential limited to `MarcoPoloResearchLab/mprlab-gateway` with repository `Contents: read` permission.
GitHub documents this permission for [release asset downloads](https://docs.github.com/en/rest/releases/assets#get-a-release-asset).
The workflow supplies this credential only to the runtime setup step as `GH_TOKEN`.
The setup step requires this credential and does not select another token.
Keep the credential value in Actions secrets.
The configured credential expires on January 25, 2027. Rotate `MPRLAB_GATEWAY_READ_TOKEN` before that date.

## Validation

Run `make github-workflow-check` for the trusted action declarations, immutable pins, and required CI gates.
Run `make mobile-ci-gateway-check` for the synthetic download and installer argument contract.
Run `make mobile-ci-bundletool-check` for download and installation failure handling.
Run `make mobile-adapter-check` with the selected installed Gateway for the real SDK and generated Gradle boundary.
The synthetic setup test does not establish actual GitHub access or Linux runtime installation.
The hosted job must verify those operations with the approved read credential.

On October 27, 2026, the configured credential installed Gateway successfully in the hosted job.
The real local npm launcher and official JAR passed version qualification with Java 21.
Final local canonical CI passed in 705.042 seconds with all 475 browser cases.
Automatic hosted run `40573676180` passed all six matrix jobs and the required aggregate job.
Its Mobile job verified the Linux Gateway runtime and the real bundletool and generated Gradle boundary.
B145 is closed after these code validation results. These results do not establish publication or deployment.
Fix a failed hosted job before a new push. Let that push trigger CI automatically.


## First Failure

Run `make ci` or one declared `ci-*` group without parallel Make stages or keep-going options.
CI rejects those options before application recipes run. Go and Node retain their native test concurrency.
The hosted matrix stops sibling jobs after a job fails. The three integration shards remain unchanged.

Static checks qualify the repository-owned Node adapter and Go monitor through public failure fixtures.
The Go monitor uses its own selected toolchain and native JSON events.
It closes only its owned process group and retains the first failure diagnostic.
Hosted Backend checks do not require an installed Gateway for this monitor.

Mobile checks qualify pytest through its public Make recipe before the npm visibility tests.
The public tool prerequisite fixes Ansible Core 2.19.8 and pytest 9.0.2 under Python 3.13.
Its isolated environment uses public PyPI without private inputs.
Integration checks qualify browser cleanup and the shared Playwright failure limit before application browser tests.
Both browser paths use the existing browser installation target and declared channel.
Expected negative fixture outcomes do not count as failures of the outer qualification test.
The outer test stops on its first unexpected result and then completes owned cleanup.
