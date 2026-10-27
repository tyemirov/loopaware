# Hosted Mobile Checks

The `Mobile checks` job uses Node.js 24, Python 3.13, and Temurin Java 21.
The checked-in Gradle wrapper supplies Gradle for the generated signing check.
After Java setup, the job installs official Google `bundletool` 1.18.3 and verifies its version.
A dependency-free npm launcher forwards arguments to the published JAR through the selected Java runtime.
The job adds its installation directory through `GITHUB_PATH`. The download uses no private credential.
The pinned artifact comes from the [official Google release](https://github.com/google/bundletool/releases/tag/1.18.3).
The SDK checks Node, npm, Java, and bundletool before the controlled Gradle boundary.
This signing fixture stops before bundle compilation and does not require Android SDK build tools.
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
The hosted Mobile job still requires verification from the next source push under B145.
Fix a failed hosted job before a new push. Let that push trigger CI automatically.
