// @ts-check
import { readFile } from "node:fs/promises";
import { resolve, dirname, join, isAbsolute } from "node:path";
import { withAppleSigning } from "./apple-signing.mjs";
import { androidSigningEnvironment } from "./android-signing.mjs";
import { runNativeBuild } from "./native-build-process.mjs";
import { createMobileCalVerVersion } from "./mobile-calver-version.mjs";

const acceptedOptions = new Set(["--mobile-dir", "--output", "--release-timestamp", "--manifest"]);
const options = new Map();
const argumentsList = process.argv.slice(2);
for (let index = 0; index < argumentsList.length; index += 2) {
  const name = argumentsList[index];
  const value = argumentsList[index + 1];
  if (!acceptedOptions.has(name) || options.has(name) || !value) throw new Error(`Invalid release option: ${name}`);
  options.set(name, value);
}
for (const name of ["--mobile-dir", "--output", "--release-timestamp"]) {
  if (!options.has(name)) throw new Error(`Release requires ${name}`);
}
const sourceRoot = resolve(options.get("--mobile-dir"));
const output = resolve(options.get("--output"));
const platform = output.endsWith(".ipa") ? "ios" : output.endsWith(".aab") ? "android" : undefined;
if (!platform) throw new Error("Release output must be an IPA or AAB.");
if (options.has("--manifest") && resolve(options.get("--manifest")) !== join(dirname(output), `${platform}.json`)) throw new Error("Release manifest must use the gateway platform path.");
if (!/^v?\d+\.\d+\.\d+$/.test(process.env.MPRLAB_ARTIFACT_VERSION ?? "")) throw new Error("Release requires the allocated MPRLAB_ARTIFACT_VERSION.");
const timestamp = options.get("--release-timestamp");
const milliseconds = Date.parse(timestamp);
if (!Number.isFinite(milliseconds) || new Date(milliseconds).toISOString().replace(".000Z", "Z") !== timestamp) throw new Error("Release timestamp must use canonical UTC seconds.");
const versioning = createMobileCalVerVersion(timestamp);
const gateway = process.env.MPRLAB_GATEWAY_EXECUTABLE;
if (!gateway || !isAbsolute(gateway)) throw new Error("Release requires the authoritative gateway executable.");
const config = JSON.parse(await readFile(join(sourceRoot, "app.config.snapshot.json"), "utf8")).expo;
const request = {
  schema_version: 1, source_root: sourceRoot, platform, output,
  application_identifier: platform === "ios" ? config.ios.bundleIdentifier : config.android.package,
  version: versioning.releaseVersion, build_number: String(versioning.buildCode), release_timestamp: timestamp,
  preparation_manifest: "native-preparation.json", verify_script: "scripts/verify-store-preparation.mjs",
  ...(platform === "android" ? { android: {
    module: "app", version_name_environment: "MPRLAB_MOBILE_VERSION_NAME", version_code_environment: "MPRLAB_MOBILE_VERSION_CODE",
    signing_environment: ["LOOPAWARE_ANDROID_KEYSTORE", "LOOPAWARE_ANDROID_STORE_PASSWORD", "LOOPAWARE_ANDROID_KEY_ALIAS", "LOOPAWARE_ANDROID_KEY_PASSWORD"]
  } } : { ios: {
    workspace: "ios/LoopAware.xcworkspace", scheme: "LoopAware", app_name: "LoopAware", entry_file: "index.ts",
    team_environment: "LOOPAWARE_APPLE_TEAM", profile_environment: "LOOPAWARE_APPLE_PROFILE", identity_environment: "LOOPAWARE_APPLE_IDENTITY", export_intent: "app-store"
  } })
};
const repositoryRoot = resolve(import.meta.dirname, "../..");
const controller = new AbortController();
const interrupt = () => controller.abort("SIGINT");
const terminate = () => controller.abort("SIGTERM");
process.on("SIGINT", interrupt);
process.on("SIGTERM", terminate);
try {
  if (platform === "ios") {
    process.exitCode = await withAppleSigning({ repositoryRoot, applicationIdentifier: request.application_identifier, signal: controller.signal }, async signing => {
      const signedRequest = { ...request, ios: { ...("ios" in request ? request.ios : {}), keychain_environment: signing.keychainEnvironment } };
      return runNativeBuild(gateway, signedRequest, signing.environment, controller.signal);
    });
  } else {
    const environment = await androidSigningEnvironment(repositoryRoot);
    process.exitCode = await runNativeBuild(gateway, request, environment, controller.signal);
  }
} catch (error) {
  process.stderr.write(`${error instanceof Error ? error.message : String(error)}\n`);
  process.exitCode = controller.signal.aborted ? (controller.signal.reason === "SIGINT" ? 130 : 143) : 2;
} finally {
  process.removeListener("SIGINT", interrupt);
  process.removeListener("SIGTERM", terminate);
}
