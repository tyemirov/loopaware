# Release, publication, and deployment

[Project overview](../README.md)

## Lifecycle and destinations

The selected manifest declares these destinations:

- GitHub Pages for the static frontend at `https://loopaware.mprlab.com`.
- The API service at `https://loopaware-api.mprlab.com`.
- GHCR for the backend image and npm for `@loopaware/react-native`.
- Internal TestFlight and Google Play tracks for the native operator app.

A declared destination describes the operation target.
Confirm publication and deployment from their operation results.

## Release, Publish, Deploy

LoopAware declares its complete production shape in `.mprlab/deploy/resources.yml`. That permanent versionless manifest is the only tracked production lifecycle declaration. It declares backend services, retained data, runtime endpoints, public routing, and GitHub Pages delivery.
It also declares the React Native package, native mobile artifacts, and private input references.

Run the complete lifecycle from this repository:

```bash
make deploy
```

Gateway v9 prepares the release, publishes its artifacts, deploys the runtime, and verifies the declared surfaces.
The optional `make release` and `make publish` targets execute their named phases.
Each target calls the installed `mprlab-gateway` command with `release`, `publish`, or `deploy`.
The wrapper passes the exact application Git root through `--app-root`.

## Gateway execution

Set `MPRLAB_GATEWAY_EXECUTABLE` to select an explicit installed executable.
The wrapper passes caller environment values unchanged.
Gateway reads only the environment file selected by the manifest.
Caller values take precedence.

Each Gateway call uses `CI=1` and closed standard input.
The selected manifest disables the lifecycle CI stage.
Run `make ci` separately for source validation.
Gateway owns lifecycle validation, artifact preparation, publication, deployment, and verification.

LoopAware keeps its native builders and application inputs.

The full pipeline has a completion limit of five hours and 30 minutes.
CI and release each keep a 30-minute budget. Publication keeps four hours.
The total includes a 20-minute activation allowance and ten minutes for orchestration.

Gateway uses the same `app-deploy` budget for the complete pipeline and runtime activation.

## Private inputs

The selected manifest binds the npm publication credential as `NPM_API_KEY`.
Gateway supplies this captured value to npm through standard input.
See [npm publication credentials](npm-publication.md) for the input contract and authentication diagnosis.

The selected manifest declares `.mprlab/deploy/.env` through `environment.source`.
Gateway resolves that file and process values during manifest processing for release, publication, and deployment.
Each operation uses its captured inputs.
The `environment.private_inputs` catalog binds private input references.

Keep secret values outside the manifest and public artifacts.

## Native preparation

Run `make mobile-prepare-store` after a native configuration or dependency change.
This command generates the production native projects under `mobile/prepared/`.
Commit the prepared files with their source changes.
Release verifies application identity, native configuration, and dependency versions before the native build.

The adapter stages the current captured product source with the prepared native inputs.

## iOS build contract

iOS releases use the shared Gateway native builder on a Mac.
The selected manifest uses `mobile/scripts/build-store-artifact.mjs` for Android and iOS.
Gateway exports the signed IPA only for App Store Connect and preserves the internal TestFlight restriction.
No physical iOS device or registered test device is required.

Use Xcode 26.6, CocoaPods 1.17, and Node.js 24 or later on the build Mac.
The iOS app version comes from `ios.version` in `mobile/app.config.js`.
The generated plist reads `MARKETING_VERSION` and `CURRENT_PROJECT_VERSION` from the native build settings.
The release timestamp allocates its integer build number as UTC seconds since January 1, 2020.

The Android adapter keeps its native AAB builder and UTC CalVer contract.
Application versions can differ from the repository release version.
Expo CLI runs during preparation and local development only.

The iOS app declares `ios.config.usesNonExemptEncryption: false` for its current use of Apple system services.
Expo writes this declaration as Boolean false under `ITSAppUsesNonExemptEncryption` in the native Info.plist.
Apple reads the uploaded declaration for TestFlight export compliance.
Native preparation verifies both source and plist declarations before the build.

The Release metadata cleanup phase preserves this declaration.

The prepared Apple project contains the shared scheme and native dependency inputs.
Gateway installs the locked dependencies and supplies `NODE_ENV=production` to each native phase.
`mobile/plugins/withStoreBuild.cjs` owns the native bundle phase and its Node environment input.
The plugin removes absent test targets from the shared scheme and generates strict Podfile property parsing.

It explicitly selects React Native and Expo source dependencies, including when the process inherits prebuilt flags.
The current Expo 57 source config enables the iOS scene lifecycle.
`expo-build-properties` generates `EXExpoAppSceneDelegate` and `ExpoReactNativeFactoryProvider`.
The native source dependency flags remain explicit.

`make mobile-scene-check` verifies this contract through the real Expo generator.
Android preparation runs `mobile/scripts/prepare-android-build-policy.mjs` after Expo.
The generated `gradlew` supplies `-Pkotlin.compiler.execution.strategy=in-process`.
The JVM metaspace limit is `1g`.

Preparation keeps the heap, parallel execution, workers, ABI selection, and signing inputs unchanged.
`make mobile-android-policy-check` verifies real launcher arguments and repeated preparation.
`make mobile-prepare-store` runs `make mobile-podfile-config-check` after native preparation.
This check uses actual CocoaPods for valid, missing, and malformed properties, plus an unavailable prebuilt service.

Use `pod install` from `mobile/prepared/ios` for native dependency preparation.
CocoaPods owns the generated dependency metadata.
The preparation script configures the release metadata phase through `mobile/scripts/fix-ios-project-warnings.mjs`.
The phase uses Apple Foundation through `mobile/scripts/strip-release-metadata.swift` and the declared Xcode toolchain.

It rejects missing or invalid property lists and preserves custom app values and the original file format.
It removes only the exact Expo development service entries and default description.
`make mobile-release-metadata-check` executes the generated phase against seven cases on macOS.
Other platforms check the declared input dependency and skip the six macOS execution cases.

Native preparation and release validation include this target.

## Signing inputs

Keep the Android signing files under the ignored `configs/signing/` directory.
Supply their paths and passwords through the caller environment or the selected `.mprlab/deploy/.env` file.
The Android key must match `mobile/android-release-identity.json`.
The release adapter uses `MPRLAB_APP_ROOT` to resolve private signing files from the selected checkout.

It reads the registered certificate identity from the captured source snapshot.
Gateway reads the selected repository input for iOS signing and store API authentication.

The private configuration supplies these inputs:

- `LOOPAWARE_ANDROID_KEYSTORE`, `LOOPAWARE_ANDROID_STORE_PASSWORD`, `LOOPAWARE_ANDROID_KEY_ALIAS`, and `LOOPAWARE_ANDROID_KEY_PASSWORD`.
- `LOOPAWARE_APPLE_TEAM`, `LOOPAWARE_APPLE_PROFILE`, and `LOOPAWARE_APPLE_IDENTITY`.
- `LOOPAWARE_APPLE_CERTIFICATE_BASE64` and `LOOPAWARE_APPLE_CERTIFICATE_PASSWORD`.
- `APP_STORE_CONNECT_API_KEY_ID`, `APP_STORE_CONNECT_API_ISSUER_ID`, and `APP_STORE_CONNECT_API_KEY_PATH`.
- `GOOGLE_PLAY_SERVICE_ACCOUNT_KEY_PATH`, `NPM_API_KEY`, and `GH_TOKEN`.
- `JAVA_HOME` and `ANDROID_HOME` for the Android toolchain.

The Apple certificate value contains the base64 PKCS#12 identity, including its private key.
The profile and certificate identifier must match that identity and the selected application.
The profile must permit App Store distribution.
Gateway creates a temporary keychain, imports the identity, and removes its temporary state after the build.

Authorized build operators must receive these private inputs separately from the Git clone.

## Validation and operation evidence

Run `make mobile-release-check` to validate prepared inputs, native adapter forwarding, and Android signing behavior.
The local provider tests use controlled dependencies.
Actual native signing and store submission remain separate acceptance results.

For a non-mutating inspection, use the installed Gateway's `app-plan-release`, `app-plan-publish`, or `app-plan-deploy` command.
Pass this repository's Git root through `--app-root`.
Production activation remains an operator action.
