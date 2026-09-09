# Apple Applications

## Scope

Use this guide for release builds and distribution of applications for Apple platforms, including iOS and macOS.
Obey root `AGENTS.md` and `.mprlab/POLICY.md`.

## Canonical Flow

Use Xcode Cloud for each Apple release build.
Use automatic signing and Apple cloud-managed certificates.
Xcode Cloud owns the build environment, signing operation, and build number.
The release operator needs account authorization and source access, not a prepared personal Mac.

- Keep one Xcode Cloud flow for each application target.
- Keep the product declarations in `.mprlab/apple-build.json`.
- Remove local release signing, certificate imports, provisioning profile installation, and Keychain commands during the application migration.
- Remove alternative release builders during the Xcode Cloud migration.
- Require a successful hosted build before release acceptance.
- Keep simulator builds and local development separate from release acceptance.
- Keep public App Store release under operator control.

## Ownership

MPRLab-Gateway owns shared App Store Connect API operations and build evidence.
Each application owns its product identifiers, native project, shared scheme, dependencies, workflow declaration, and release version.

- Use the shared Gateway operation from the application build target.
- Use Gateway's `apple-cloud-operation` command and its documented configuration flags.
- Keep the application adapter limited to command forwarding and exit-status propagation.
- Use the POSIX shell adapter template supplied with MPR Governor.
- Declare iOS store releases with `mobile_application` and its `build.ios` shell adapter.
- Declare direct macOS releases with `macos_application` and its `build.macos` shell adapter.
- Publish the notarized macOS ZIP through the shared Gateway release and publication lifecycle.
- Run the adapter through `/bin/sh` and invoke the compiled Gateway executable directly.
- Keep Node.js dependencies in the cloud build environment.
- Let Gateway read the selected repository's `configs/.env.<owner>` for cloud API credentials.
- Keep provider authentication, build requests, status checks, and artifact collection in Gateway.
- Use the existing App Store Connect API credentials through the documented private input channel.
- Keep private keys and secret values outside source control and build output.
- Require an exact match between the requested source commit and the commit that Apple builds.
- Record the Apple build identifier, build number, workflow identifier, source commit, and final result.
- Preserve the build identifier before another status check or artifact request.
- Resume the recorded build after an interruption.
- Report an uncertain build submission before another submission.

## Project And Dependency Inputs

- Put the Xcode project or workspace and its shared scheme in source control before cloud setup.
- Commit the release version before the cloud build.
- Use automatic signing in the project settings.
- Declare the Xcode version and each required tool in the workflow or repository.
- Keep application dependencies and their lockfiles in the repository contract.
- Install declared dependencies in the cloud environment through repository scripts.
- Require no dependency on another local checkout, a personal cache, or a user-installed signing identity.
- For Expo applications, generate native projects before the release build and retain the governed output in source control.
- Keep the cloud build independent of an Expo account and EAS services.

## Build And Distribution Evidence

Xcode Cloud assigns an integer build number.
App Store Connect uses that number for the uploaded build.
Use the Apple number as the canonical Apple build number in release evidence.
For iOS, a new app version can start with cloud build number `1`.
For macOS, keep build numbers increasing across app versions.

- During migration, set the next cloud build number when the existing application requires a larger number.
- For store workflows, use the build that Xcode Cloud uploads to App Store Connect.
- Retain the cloud receipt and downloaded artifacts in one release directory.
- Preserve the submission journal when a build is interrupted.
- Publish through Gateway verification of the existing Apple build.
- Require the selected source, version, build number, distribution, and processed store build to match.
- Keep TestFlight notes in `TestFlight/WhatToTest.<LOCALE>.txt` beside the native project or workspace.
- Let Xcode Cloud attach those notes to the uploaded build.
- Keep store review and public release as separate operations after a successful build.
- For direct macOS distribution, require the workflow's signed and notarized artifact.
- Download release artifacts, symbols, and logs before Apple's 30-day retention period ends.
- Verify the product identifier, release version, source commit, and provider result before release acceptance.
- Report cloud build failure with its build identifier and available diagnostics.

## Setup And Validation

1. Verify the application record and authorized Apple team.
2. Prepare the tracked project and shared scheme.
3. Configure the first Xcode Cloud workflow in Xcode.
4. Authorize access to the source repository.
5. Record the workflow declaration in the application repository.
6. Run the shared Gateway operation for an authorized source commit.
7. Verify the hosted build, retained artifacts, and App Store Connect build record when applicable.

For a behavior change, start with an integration test through the real public entry point.
Use `.mprlab/POLICY.md` for validation.
Local protocol tests prove the Gateway contract.
A successful hosted build proves Apple provider acceptance for that application.

## Apple References

- [Initial cloud setup](https://developer.apple.com/documentation/xcode/configuring-your-first-xcode-cloud-workflow)
- [Cloud build numbers](https://developer.apple.com/documentation/xcode/setting-the-next-build-number-for-xcode-cloud-builds)
- [Workflow and build API](https://developer.apple.com/documentation/appstoreconnectapi/xcode-cloud-workflows-and-builds)
- [Distribution workflows](https://developer.apple.com/documentation/xcode/creating-a-workflow-that-builds-your-app-for-distribution)
- [TestFlight notes](https://developer.apple.com/documentation/xcode/including-notes-for-testers-with-a-beta-release-of-your-app)
