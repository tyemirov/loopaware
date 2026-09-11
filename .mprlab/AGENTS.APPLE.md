# Apple Applications

## Scope

Use this guide for the LoopAware iOS release and distribution.
Obey root `AGENTS.md` and `.mprlab/POLICY.md`.

## Canonical Flow

Build and sign iOS release artifacts locally on a Mac through Gateway.
Export only for App Store Connect.
Publish the sealed IPA through the shared Gateway adapter.
Do not use Xcode Cloud, development exports, or ad hoc exports for iOS deployment.
App Store and TestFlight distribution require no registered test device.

- Declare the `.mjs` adapter in the selected `mobile_application` resource.
- Keep application identifiers and version policy in the application source.
- Preserve the declared TestFlight or App Store destination and its export restriction.
- Use `internal-only` for internal TestFlight and `app-store` for public App Store artifacts.
- Keep simulator and device acceptance separate from release and publication.

## Ownership And Inputs

Gateway owns native execution, temporary keychains, artifact checks, store requests, and submission journals.
The application owns the prepared project, dependencies, build request, version policy, and private input names.

- Use the authoritative Gateway executable and `mobile-build-operation`.
- Keep the application adapter limited to request construction and exit-status propagation.
- Read private inputs from `configs/.env.loopaware` through Gateway.
- Supply the team signing identity as a base64 PKCS#12 value and its password.
- Supply the team, provisioning profile, certificate identifier, and App Store Connect API credentials.
- Keep secrets outside source control, artifacts, receipts, and normal output.
- Use the temporary keychain that Gateway creates and removes for each build.
- Require no personal keychain password or registered test device.
- Give authorized build operators the same private inputs through the approved private channel.

## Project And Version

- Use Xcode 26.6, CocoaPods 1.17, and Node.js 24 or later on the build Mac.
- Commit the native project, shared scheme, dependency declarations, and lockfiles.
- Generate native inputs with `make mobile-prepare-store` before release.
- Commit the preparation records with their source inputs.
- Keep Expo CLI and EAS outside release, publication, and deployment.
- Read the iOS application version from `ios.version` in `mobile/app.config.js`.
- Allocate the build number from the canonical release timestamp.
- Preserve the allocated build identity after interruption.
- Reject stale preparation or conflicting build evidence.

## Validation And Evidence

- Run the adapter regression through its selected public entry point.
- Run `make mobile-release-check` and the canonical CI checks.
- Require a successful native archive and App Store Connect export before release acceptance.
- Verify the signed identifier, app version, build number, and artifact digest.
- Publish only the sealed artifact.
- Record each store mutation intent before its remote effect.
- Read provider state before another submission after interruption.
- Keep store processing, review, public availability, and device acceptance as separate results.

A local protocol test proves the Gateway contract under controlled inputs.
An actual native build proves signing on that Mac.
A verified store record proves submission of the selected artifact.

## Apple References

- [App Store provisioning profiles](https://developer.apple.com/help/account/provisioning-profiles/create-an-app-store-provisioning-profile/)
- [Build uploads](https://developer.apple.com/help/app-store-connect/manage-builds/upload-builds/)
