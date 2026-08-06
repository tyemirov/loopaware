// @ts-check
/// <reference types="node" />

import fs from "node:fs";
import path from "node:path";
import crypto from "node:crypto";

const repoRoot = path.resolve(import.meta.dirname, "../..");

for (const requiredFile of [
  "mobile/App.tsx",
  "mobile/app.config.js",
  "mobile/src/api.ts",
  "mobile/src/auth.ts",
  "mobile/src/config.ts",
  "mobile/src/types.ts",
  "mobile/assets/icon.png",
  "mobile/android-release-identity.json",
  "mobile/scripts/fix-ios-project-warnings.mjs",
  "mobile/scripts/build-android-bundle.mjs",
  "mobile/scripts/build-ios-archive.mjs",
  "mobile/scripts/native-build-fingerprint.mjs",
  "mobile/scripts/mobile-calver-version.mjs",
  "mobile/scripts/prepare-android-project.mjs",
  "mobile/scripts/publish-android-play.mjs",
  "mobile/scripts/resolve-metro-port.mjs",
  "mobile/scripts/submit-ios.mjs",
  "mobile/scripts/test-api-boundaries.mjs",
]) {
  assert(fs.existsSync(path.join(repoRoot, requiredFile)), `mobile_config_missing_file: ${requiredFile}`);
}

const androidReleaseIdentity = readJSON("mobile/android-release-identity.json");
assert(
  androidReleaseIdentity.schema === "loopaware.mobile-android-release-identity.v1",
  "mobile_android_release_identity_invalid_schema",
);
assert(androidReleaseIdentity.googleCloudProjectId === "loopaware", "mobile_android_release_identity_invalid_project_id");
assert(androidReleaseIdentity.googleCloudProjectNumber === "281540686395", "mobile_android_release_identity_invalid_project_number");
assert(androidReleaseIdentity.packageName === "com.mprlab.loopaware", "mobile_android_release_identity_invalid_package");
assert(
  androidReleaseIdentity.androidClientId === "281540686395-7lt9u98ir3oincpqhdrjur5169qel8n9.apps.googleusercontent.com",
  "mobile_android_release_identity_invalid_android_client_id",
);
assert(
  androidReleaseIdentity.playAppSigning?.sha256 === "CE:E0:CF:75:5F:CC:61:E3:38:B4:B8:E3:68:F2:7C:05:59:A3:14:CA:A0:49:9D:88:2C:01:5B:A0:6B:9B:EB:2C",
  "mobile_android_release_identity_invalid_play_signing_sha256",
);
assert(
  androidReleaseIdentity.uploadKey?.sha256 === "37:35:47:AD:E8:49:30:76:15:2C:36:03:02:46:97:97:0D:DE:5E:91:0A:27:54:7B:CA:E7:4E:25:51:DC:66:EE",
  "mobile_android_release_identity_invalid_upload_key_sha256",
);
assert(
  androidReleaseIdentity.debugKey?.sha1 === "A6:37:D9:89:83:3C:C3:7B:7F:9C:21:25:49:BC:35:16:14:F9:5B:4C",
  "mobile_android_release_identity_invalid_debug_sha1",
);

for (const obsoleteFile of ["mobile/app.json", "mobile/eas.json"]) {
  assert(!fs.existsSync(path.join(repoRoot, obsoleteFile)), `mobile_config_obsolete_file: ${obsoleteFile}`);
}

const packageJSON = readJSON("mobile/package.json");
const packageLockJSON = readJSON("mobile/package-lock.json");
assert(packageJSON.dependencies?.expo === "~57.0.9", "mobile_config_expo_patch_version_outdated");
assert(packageJSON.dependencies?.["expo-auth-session"] === "~57.0.5", "mobile_config_auth_session_patch_version_outdated");
assert(packageJSON.dependencies?.["expo-constants"] === "~57.0.8", "mobile_config_constants_patch_version_outdated");
assert(packageJSON.dependencies?.["expo-dev-client"] === "~57.0.10", "mobile_config_dev_client_patch_version_outdated");
assert(packageJSON.dependencies?.["expo-web-browser"] === "~57.0.2", "mobile_config_web_browser_patch_version_outdated");
assert(packageJSON.devDependencies?.typescript === "~7.0.2", "mobile_config_typescript_version_outdated");
assert(
  packageJSON.devDependencies?.["@typescript/typescript6"] === "6.0.2",
  "mobile_config_typescript_compiler_api_version_outdated",
);
assert(packageJSON.overrides?.uuid === "^11.1.1", "mobile_config_missing_uuid_audit_override");
for (const [dependencyName, secureVersion] of Object.entries({
  "ajv@8.11.0": "8.20.0",
  "brace-expansion": "5.0.9",
  "diff@7.0.0": "8.0.4",
  "js-yaml": "4.3.0",
  "joi@17.11.0": "17.13.4",
  "minimatch@5.1.2": "5.1.9",
  postcss: "8.5.25",
  "shell-quote": "1.10.0",
  "tar@7.5.19": "7.5.22",
  "ts-deepmerge@6.2.0": "8.0.0",
  "yaml@2.6.0": "2.9.0",
})) {
  assert(packageJSON.overrides?.[dependencyName] === secureVersion, `mobile_config_insecure_override: ${dependencyName}`);
}
assert(!packageJSON.devDependencies?.["eas-cli"], "mobile_config_eas_cli_must_be_absent");
assert(packageJSON.devDependencies?.["pod-install"] === "1.1.0", "mobile_config_missing_locked_pod_install");
assert(!packageLockJSON.packages?.[""]?.devDependencies?.["eas-cli"], "mobile_config_lock_contains_eas_cli");
assert(!packageLockJSON.packages?.["node_modules/eas-cli"], "mobile_config_lock_contains_eas_cli_package");
assert(
  packageLockJSON.packages?.[""]?.devDependencies?.["pod-install"] === "1.1.0" &&
    packageLockJSON.packages?.["node_modules/pod-install"]?.version === "1.1.0",
  "mobile_config_lock_missing_pod_install",
);
assert(
  packageLockJSON.packages?.[""]?.devDependencies?.typescript === "~7.0.2" &&
    packageLockJSON.packages?.["node_modules/typescript"]?.version === "7.0.2",
  "mobile_config_lock_missing_typescript",
);
assert(
  packageLockJSON.packages?.[""]?.devDependencies?.["@typescript/typescript6"] === "6.0.2" &&
    packageLockJSON.packages?.["node_modules/@typescript/typescript6"]?.version === "6.0.2",
  "mobile_config_lock_missing_typescript_compiler_api",
);

for (const dependencyName of [
  "expo-auth-session",
  "expo-constants",
  "expo-crypto",
  "expo-dev-client",
  "expo-secure-store",
  "expo-system-ui",
  "expo-web-browser",
  "react-native-safe-area-context",
]) {
  assert(packageJSON.dependencies?.[dependencyName], `mobile_config_missing_dependency: ${dependencyName}`);
}

for (const scriptName of [
  "android",
  "android:prepare-native",
  "android:dev-build",
  "ios",
  "ios:prepare-native",
  "ios:dev-build",
  "start",
  "test:api-boundaries",
  "typecheck",
  "validate-config",
]) {
  assert(packageJSON.scripts?.[scriptName], `mobile_config_missing_script: ${scriptName}`);
}
assert(
  packageJSON.scripts?.["android:prepare-native"]?.includes("prepare-android-project.mjs"),
  "mobile_config_missing_android_prepare_script",
);
assert(
  packageJSON.scripts?.["android:dev-build"]?.includes("android:prepare-native") &&
    packageJSON.scripts?.["android:dev-build"]?.includes("--no-install"),
  "mobile_config_missing_android_dev_build_prepare_contract",
);
assert(
  packageJSON.scripts?.["ios:prepare-native"]?.includes("fix-ios-project-warnings.mjs"),
  "mobile_config_missing_ios_warning_fix_script",
);

const appConfigSource = readText("mobile/app.config.js");
for (const expectedText of [
  "com.mprlab.loopaware",
  "loopAware",
  "expo-web-browser",
  "expo-secure-store",
  "expo-system-ui",
  "LOOPAWARE_MOBILE_GOOGLE_IOS_REDIRECT_URI",
  "LOOPAWARE_MOBILE_VERSION",
  "NODE_ENV",
  "mobilePlugins",
  "LOOPAWARE_MOBILE_IOS_BUILD_NUMBER",
  "LOOPAWARE_MOBILE_ANDROID_VERSION_CODE",
  "redirectUriScheme",
  "optionalPositiveInteger",
  "optionalCalVerVersion",
  "D4AF37",
]) {
  assert(appConfigSource.includes(expectedText), `mobile_config_missing_app_config_contract: ${expectedText}`);
}
assert(!appConfigSource.includes("android-icon-background.png"), "mobile_config_legacy_adaptive_icon_background_image");
assert(!appConfigSource.includes("edgeToEdgeEnabled"), "mobile_config_legacy_edge_to_edge_enabled");

const expectedLoopAwareIconHash = "6a6a558580003e70cd75ba46f968bc22e40caa4064bd47b7d3fb7413b3eff49b";
assert(fileHash("mobile/assets/icon.png") === expectedLoopAwareIconHash, "mobile_config_icon_not_loopaware_logo");

const apiSource = readText("mobile/src/api.ts");
assert(apiSource.includes('credentials: "include"'), "mobile_api_missing_cookie_credentials");
assert(apiSource.includes('"X-TAuth-Tenant"'), "mobile_api_missing_tauth_tenant_header");
assert(apiSource.includes("/api/sites"), "mobile_api_missing_sites_endpoint");
assert(apiSource.includes("/api/reports/traffic/portfolio"), "mobile_api_missing_portfolio_endpoint");
assert(apiSource.includes("/visits/stats"), "mobile_api_missing_traffic_stats_endpoint");
assert(apiSource.includes("/sentry/issues"), "mobile_api_missing_sentry_endpoint");
assert(apiSource.includes("readCollection"), "mobile_api_missing_collection_boundary_normalizer");
assert(apiSource.includes("mobile_api_invalid_collection"), "mobile_api_missing_invalid_collection_error");

const authSource = readText("mobile/src/auth.ts");
assert(authSource.includes("expo-auth-session"), "mobile_auth_missing_auth_session");
assert(authSource.includes("expo-web-browser"), "mobile_auth_missing_web_browser");
assert(authSource.includes("/auth/google/native") || apiSource.includes("/auth/google/native"), "mobile_auth_missing_native_tauth_flow");

const nativeBuildFingerprintSource = readText("mobile/scripts/native-build-fingerprint.mjs");
for (const envName of [
  "LOOPAWARE_MOBILE_ANDROID_PACKAGE",
  "LOOPAWARE_MOBILE_IOS_BUNDLE_IDENTIFIER",
  "LOOPAWARE_MOBILE_GOOGLE_IOS_REDIRECT_URI",
  "LOOPAWARE_MOBILE_GOOGLE_IOS_CLIENT_ID",
  "TAUTH_TENANT_GOOGLE_IOS_REDIRECT_URI_LOOPAWARE",
  "TAUTH_TENANT_GOOGLE_IOS_CLIENT_ID_LOOPAWARE",
]) {
  assert(nativeBuildFingerprintSource.includes(envName), `mobile_native_fingerprint_missing_env: ${envName}`);
}
const makefile = readText("Makefile");
for (const target of ["mobile-install", "mobile-check", "mobile-start", "run-ios", "run-android", "release publish deploy"]) {
  assert(makefile.includes(`${target}:`), `mobile_makefile_missing_target: ${target}`);
}
assert(makefile.includes('gateway_root="$$(dirname "$${application_root}")/mprlab-gateway"'), "lifecycle_gateway_wrapper_missing");
assert(makefile.includes('"app-$@"'), "lifecycle_gateway_phase_forwarding_missing");
assert(makefile.includes("mobile-check"), "mobile_makefile_missing_ci_gate");
assert(makefile.includes("test:api-boundaries"), "mobile_makefile_missing_api_boundary_check");
assert(makefile.includes("override MOBILE_NPM_COMMAND := env -u NO_COLOR npm"), "mobile_makefile_missing_canonical_no_color_command");
assert(makefile.includes("MOBILE_NATIVE_BUILD_FINGERPRINT"), "mobile_makefile_missing_native_build_fingerprint");
assert(makefile.includes("Development build missing or stale"), "mobile_makefile_missing_stale_native_build_guard");
assert(!makefile.includes('@mkdir -p "$(dir $(ANDROID_LOCAL_PROPERTIES))"'), "mobile_makefile_creates_partial_android_project");
assert(makefile.includes("MOBILE_GOOGLE_IOS_REDIRECT_URI"), "mobile_makefile_missing_ios_google_redirect_uri_variable");
assert(makefile.includes('LOOPAWARE_MOBILE_IOS_BUNDLE_IDENTIFIER="$(MOBILE_IOS_BUNDLE_IDENTIFIER)"'), "mobile_makefile_missing_ios_bundle_env");
assert(makefile.includes('LOOPAWARE_MOBILE_ANDROID_PACKAGE="$(MOBILE_ANDROID_PACKAGE)"'), "mobile_makefile_missing_android_package_env");
assert(
  makefile.includes('LOOPAWARE_MOBILE_GOOGLE_IOS_REDIRECT_URI="$(MOBILE_GOOGLE_IOS_REDIRECT_URI)"'),
  "mobile_makefile_missing_ios_google_redirect_uri_env",
);
assert(
  makefile.includes('EXPO_PACKAGER_PROXY_URL="http://localhost:$${metro_port}"'),
  "mobile_makefile_missing_ios_localhost_proxy_url",
);
for (const obsoleteTarget of ["release-dry-run:", "publish-dry-run:", "deploy-dry-run:", "build-ios:", "build-android:", "submit-ios:", "submit-android:"]) {
  assert(!makefile.includes(obsoleteTarget), `mobile_makefile_obsolete_target: ${obsoleteTarget}`);
}

const workflow = readText(".github/workflows/ci.yml");
assert(workflow.includes("mobile/**"), "mobile_ci_missing_path_filter");
assert(workflow.includes("mobile/package-lock.json"), "mobile_ci_missing_lockfile_cache");

const tauthConfig = readText("configs/config.tauth.yml");
assert(tauthConfig.includes("google_native_clients"), "mobile_tauth_missing_native_clients");
assert(tauthConfig.includes("TAUTH_TENANT_GOOGLE_IOS_CLIENT_ID_LOOPAWARE"), "mobile_tauth_missing_ios_client_placeholder");
assert(tauthConfig.includes("TAUTH_TENANT_GOOGLE_ANDROID_CLIENT_ID_LOOPAWARE"), "mobile_tauth_missing_android_client_placeholder");

const androidPrepareSource = readText("mobile/scripts/prepare-android-project.mjs");
assert(androidPrepareSource.includes("expo"), "mobile_android_prepare_missing_expo_prebuild");
assert(androidPrepareSource.includes("--platform") && androidPrepareSource.includes("android"), "mobile_android_prepare_missing_platform");
assert(androidPrepareSource.includes("local.properties"), "mobile_android_prepare_missing_local_properties_write");

for (const envFile of ["configs/.env.tauth.example", "configs/.env.tauth.computercat.example", "tests/configs/tauth.env"]) {
  const envSource = readText(envFile);
  for (const envName of [
    "TAUTH_TENANT_GOOGLE_IOS_CLIENT_ID_LOOPAWARE",
    "TAUTH_TENANT_GOOGLE_IOS_REDIRECT_URI_LOOPAWARE",
    "TAUTH_TENANT_GOOGLE_ANDROID_CLIENT_ID_LOOPAWARE",
    "TAUTH_TENANT_GOOGLE_ANDROID_REDIRECT_URI_LOOPAWARE",
  ]) {
    assert(envSource.includes(envName), `mobile_tauth_env_missing: ${envFile}:${envName}`);
  }
}

/**
 * @param {string} relativePath
 * @returns {string}
 */
function readText(relativePath) {
  return fs.readFileSync(path.join(repoRoot, relativePath), "utf8");
}

/**
 * @param {string} relativePath
 * @returns {any}
 */
function readJSON(relativePath) {
  return JSON.parse(readText(relativePath));
}

/**
 * @param {string} relativePath
 * @returns {string}
 */
function fileHash(relativePath) {
  return crypto.createHash("sha256").update(fs.readFileSync(path.join(repoRoot, relativePath))).digest("hex");
}

/**
 * @param {unknown} condition
 * @param {string} message
 */
function assert(condition, message) {
  if (!condition) {
    throw new Error(message);
  }
}
