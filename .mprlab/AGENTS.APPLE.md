# Apple Applications

## Scope

Use this guide for iOS and macOS application release builds and distribution.
Obey root `AGENTS.md` and `.mprlab/POLICY.md`.

## Canonical iOS Flow

Build iOS release artifacts locally on a Mac.
Use `app-store-connect` as the only iOS export method.
Sign and seal each IPA, then publish it through Gateway's App Store Connect adapter.
Use this flow for both TestFlight and App Store destinations.
Do not use development, ad hoc, or enterprise exports for iOS deployment.
Do not use Xcode Cloud for iOS release builds.
Use the same method for each application repository and each authorized team member.

- Require `build_system: local` and a repository-relative `.mjs` path in `build.ios`.
- Preserve the declared TestFlight or App Store destination and its export restriction.
- Reject iOS `.sh` adapters, cloud requests, and cloud publication receipts.
- Remove obsolete iOS cloud adapters and disable their release workflows.
- Preserve stored releases, credentials, and deployment state.
- Keep simulator builds and development separate from release acceptance.
- Require the declared Xcode toolchain and repository dependencies on the build Mac.
- Keep the build independent of a personal cache, personal signing identity, or registered test device.
- Keep public store release under the declared lifecycle authorization.

## macOS Distribution

Keep Xcode Cloud for declared macOS store builds and direct notarized downloads.
Use Gateway's shared cloud operation with `platform: MACOS`.
Keep separate named products and workflows when an application has both distribution channels.

- Use `APP_STORE_ELIGIBLE` or `INTERNAL_ONLY` for macOS store distribution.
- Use `DEVELOPER_ID` for direct notarized distribution.
- Declare direct distribution with `macos_application` and publish through GitHub Releases.
- Preserve cloud signing, notarization, provider checks, and retained build evidence for macOS.
- Keep local iOS toolchain and signing requirements out of the macOS cloud operation.

## Ownership

MPRLab-Gateway owns shared native build execution, signing, store publication, and release evidence.
Each application owns its identifiers, native project, shared scheme, dependencies, version policy, and selected private inputs.

- Keep iOS identifiers and build inputs in the application adapter and the selected deployment manifest.
- Keep macOS cloud declarations in `.mprlab/apple-build.json` and the selected deployment manifest.
- For iOS, invoke the shared Gateway builder through the repository-owned application adapter.
- For iOS, use `mobile_application`, its `build.ios` adapter, and Gateway's `mobile-build-operation` command.
- Keep the adapter limited to application request construction and exit-status propagation.
- Execute the adapter and native inputs from the selected source commit.
- Let Gateway read the selected repository's `configs/.env.<owner>` for private inputs.
- Keep provider authentication, artifact checks, submission journals, and retry decisions in Gateway.
- Keep each macOS product within its declared shared Gateway contract.

## iOS Signing Inputs

A certificate alone cannot sign an application.
A signing identity contains the certificate and its corresponding private key.
A PKCS#12 file can carry that identity between Macs.
The repository's private inputs must supply that file, its password, and the required App Store Connect API credentials.
An ignored environment file does not arrive with a Git clone.

- Declare how authorized team members receive the private inputs before release acceptance.
- Keep private keys, passwords, and secret values outside source control and build output.
- Require the same team signing identity on each build Mac.
- Use a temporary keychain that the shared builder creates and unlocks with its own generated password.
- Import the declared signing identity and permit the required Apple signing tools to use it without a prompt.
- Delete the temporary keychain and extracted private files when the build ends.
- Use App Store Connect API authentication for provisioning profile access and store publication.
- Select the provisioning profile for the declared application and distribution method.
- Do not require registered devices for App Store or TestFlight distribution.
- Stop before the build when a required private input is absent or invalid.
- Do not request account login or personal keychain passwords during a release.

## iOS Project And Dependency Inputs

- Commit the Xcode project or workspace, shared scheme, dependency declarations, and lockfiles.
- Declare the required Xcode version and other build tools in the repository.
- Install the declared dependencies through the shared build operation.
- For Expo applications, generate native projects before release and commit the governed output.
- Record the native preparation inputs and their content hashes.
- Reject stale prepared inputs before a release build.
- Keep Expo CLI and EAS outside release, publication, and deployment.
- When precompiled libraries are required, stop if those libraries are unavailable.

## Build And Distribution Evidence

- For iOS, allocate the version and build number through the repository's release policy.
- For macOS, retain the build number and build identifier from the declared cloud operation.
- For iOS, record the source commit, preparation digest, toolchain, version, build number, and artifact digest.
- For macOS, record the source commit, workflow, provider build identifier, distribution result, and artifact identity.
- Require the signed application to match the declared identifier, version, and build number.
- Seal the signed artifact before publication.
- Publish only the sealed artifact or verified cloud build to its declared destination.
- Record each submission intent before the remote operation.
- After an interruption, verify provider state before another submission.
- Reuse an exact completed artifact after its content checks pass.
- Reject conflicting source, artifact, or provider state.
- Keep store processing, review, public availability, and device acceptance as separate results.

## Validation

Use `.mprlab/POLICY.md` for integration tests and final validation.
Controlled native tools prove the shared command contract.
A successful local build with actual Apple signing inputs proves native signing for that application.
A verified store record proves submission of the selected artifact.
Require a successful iOS release from another Mac before acceptance of team portability.

## Apple References

- [Unattended code signing](https://developer.apple.com/forums/thread/712005)
- [App Store provisioning profiles](https://developer.apple.com/help/account/provisioning-profiles/create-an-app-store-provisioning-profile/)
- [Build and distribution automation](https://developer.apple.com/videos/play/wwdc2021/10204/)
- [Build uploads](https://developer.apple.com/help/app-store-connect/manage-builds/upload-builds/)
