# Hosted Mobile Checks

The `Mobile checks` job uses Node.js 24, Python 3.13, and Temurin Java 21.
The checked-in Gradle wrapper supplies Gradle for the generated signing check.
The job installs the published Gateway `v7.0.8` package for `linux-amd64` before `make ci-mobile`.
The installer uses the synthetic signing fixture manifest and synthetic inputs.
The application deployment manifest and private inputs stay unchanged.

## Release Access

Set the repository Actions secret `MPRLAB_GATEWAY_READ_TOKEN` before the hosted job runs.
Use a credential limited to `MarcoPoloResearchLab/mprlab-gateway` with repository `Contents: read` permission.
GitHub documents this permission for [release asset downloads](https://docs.github.com/en/rest/releases/assets#get-a-release-asset).
The workflow supplies this credential only to the runtime setup step as `GH_TOKEN`.
The setup step requires this credential and does not select another token.
Keep the credential value in Actions secrets.

## Validation

Run `make github-workflow-check` for the trusted action declarations, immutable pins, and required CI gates.
Run `make mobile-ci-gateway-check` for the synthetic download and installer argument contract.
Run `make mobile-adapter-check` with the selected installed Gateway for the real SDK and generated Gradle boundary.
The synthetic setup test does not establish actual GitHub access or Linux runtime installation.
The hosted job must verify those operations with the approved read credential.

As of October 27, 2026, credential provisioning and the complete hosted setup check remain pending under `B145`.
