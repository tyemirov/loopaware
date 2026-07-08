// @ts-check
/// <reference types="node" />

import fs from "node:fs";
import path from "node:path";
import crypto from "node:crypto";

const repoRoot = path.resolve(import.meta.dirname, "../..");
const mobileRoot = path.resolve(repoRoot, "mobile");

const requiredFiles = [
  "mobile/App.tsx",
  "mobile/app.config.js",
  "mobile/src/api.ts",
  "mobile/src/auth.ts",
  "mobile/src/config.ts",
  "mobile/src/types.ts",
  "mobile/assets/icon.png",
  "mobile/scripts/build-android-bundle.mjs",
  "mobile/scripts/build-ios-archive.mjs",
  "mobile/scripts/fix-ios-project-warnings.mjs",
  "mobile/scripts/mobile-calver-version.mjs",
  "mobile/scripts/native-build-fingerprint.mjs",
  "mobile/scripts/publish-android-play.mjs",
  "mobile/scripts/prepare-android-project.mjs",
  "mobile/scripts/resolve-metro-port.mjs",
  "mobile/scripts/submit-ios.mjs",
  "mobile/scripts/test-api-boundaries.mjs",
];

for (const requiredFile of requiredFiles) {
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

assert(!fs.existsSync(path.join(mobileRoot, "app.json")), "mobile_config_legacy_app_json: use app.config.js");
assert(!fs.existsSync(path.join(mobileRoot, "eas.json")), "mobile_config_legacy_eas_json: use local store build and submit scripts");

const packageJSON = readJSON("mobile/package.json");
assert(packageJSON.dependencies?.expo === "~56.0.12", "mobile_config_expo_patch_version_outdated");
assert(packageJSON.overrides?.uuid === "^11.1.1", "mobile_config_missing_uuid_audit_override");

const requiredDependencies = [
  "expo-auth-session",
  "expo-constants",
  "expo-crypto",
  "expo-dev-client",
  "expo-secure-store",
  "expo-system-ui",
  "expo-web-browser",
  "react-native-safe-area-context",
];
for (const dependencyName of requiredDependencies) {
  assert(packageJSON.dependencies?.[dependencyName], `mobile_config_missing_dependency: ${dependencyName}`);
}

const requiredScripts = [
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
];
for (const scriptName of requiredScripts) {
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
assert(appConfigSource.includes("com.mprlab.loopaware"), "mobile_config_missing_bundle_identifier");
assert(appConfigSource.includes("loopAware"), "mobile_config_missing_runtime_extra");
assert(appConfigSource.includes("expo-web-browser"), "mobile_config_missing_web_browser_plugin");
assert(appConfigSource.includes("expo-secure-store"), "mobile_config_missing_secure_store_plugin");
assert(appConfigSource.includes("expo-system-ui"), "mobile_config_missing_system_ui_plugin");
assert(appConfigSource.includes("LOOPAWARE_MOBILE_GOOGLE_IOS_REDIRECT_URI"), "mobile_config_missing_ios_google_redirect_uri_env");
assert(appConfigSource.includes("LOOPAWARE_MOBILE_VERSION"), "mobile_config_missing_calver_version_env");
assert(appConfigSource.includes("NODE_ENV") && appConfigSource.includes("mobilePlugins"), "mobile_config_missing_production_plugin_gate");
assert(appConfigSource.includes("LOOPAWARE_MOBILE_IOS_BUILD_NUMBER"), "mobile_config_missing_ios_build_number_env");
assert(appConfigSource.includes("LOOPAWARE_MOBILE_ANDROID_VERSION_CODE"), "mobile_config_missing_android_version_code_env");
assert(appConfigSource.includes("redirectUriScheme"), "mobile_config_missing_ios_google_redirect_scheme_registration");
assert(appConfigSource.includes("optionalPositiveInteger"), "mobile_config_missing_store_version_validation");
assert(appConfigSource.includes("optionalCalVerVersion"), "mobile_config_missing_calver_validation");
assert(appConfigSource.includes("D4AF37"), "mobile_config_missing_loopaware_gold_brand_color");
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
const releaseScriptSource = readText("scripts/release.sh");
for (const target of [
  "mobile-install",
  "mobile-check",
  "run-ios",
  "run-android",
  "build-ios",
  "build-android",
  "mobile-android-bundle",
  "submit-ios",
  "submit-android",
  "submit-mobile",
]) {
  assert(makefile.includes(`${target}:`), `mobile_makefile_missing_target: ${target}`);
}
assert(makefile.includes("mobile-check"), "mobile_makefile_missing_ci_gate");
assert(!makefile.includes("MOBILE_EAS"), "mobile_makefile_must_not_use_eas");
assert(!makefile.includes("eas submit"), "mobile_makefile_must_not_use_eas_submit");
assert(!makefile.includes("eas build"), "mobile_makefile_must_not_use_eas_build");
assert(!makefile.includes("MOBILE_IOS_BUILD_NUMBER"), "mobile_makefile_must_not_request_ios_build_number");
assert(!makefile.includes("MOBILE_ANDROID_VERSION_CODE"), "mobile_makefile_must_not_request_android_version_code");
assert(makefile.includes("MOBILE_RELEASE_TIMESTAMP"), "mobile_makefile_missing_release_timestamp");
assert(makefile.includes("MOBILE_RESOLVED_RELEASE_TIMESTAMP"), "mobile_makefile_missing_resolved_release_timestamp");
assert(makefile.includes("MOBILE_IOS_ASC_APP_ID ?="), "mobile_makefile_missing_ios_asc_app_id_variable");
assert(
  makefile.includes('MOBILE_IOS_ASC_APP_ID="$(MOBILE_IOS_ASC_APP_ID)"'),
  "mobile_makefile_missing_ios_asc_app_id_env",
);
assert(
  makefile.includes('APP_STORE_CONNECT_API_KEY_ID="$(APP_STORE_CONNECT_API_KEY_ID)"'),
  "mobile_makefile_submit_missing_app_store_key_id_env",
);
assert(
  makefile.includes('APP_STORE_CONNECT_API_ISSUER_ID="$(APP_STORE_CONNECT_API_ISSUER_ID)"'),
  "mobile_makefile_submit_missing_app_store_issuer_env",
);
assert(makefile.includes('MOBILE_RELEASE_TIMESTAMP="$(MOBILE_RELEASE_TIMESTAMP)"'), "mobile_makefile_release_missing_mobile_timestamp_env");
assert(makefile.includes('APP_STORE_CONNECT_API_KEY_PATH="$(APP_STORE_CONNECT_API_KEY_PATH)"'), "mobile_makefile_release_missing_app_store_key_path_env");
assert(makefile.includes('ANDROID_SDK_ROOT="$(ANDROID_SDK_ROOT)"'), "mobile_makefile_release_missing_android_sdk_env");
assert(makefile.includes("MOBILE_IOS_ARCHIVE_SCRIPT"), "mobile_makefile_missing_ios_archive_script");
assert(makefile.includes('node "$(MOBILE_IOS_ARCHIVE_SCRIPT)"'), "mobile_makefile_missing_ios_archive_command");
assert(makefile.includes('node "$(MOBILE_ANDROID_BUNDLE_SCRIPT)"'), "mobile_makefile_missing_android_bundle_script");
assert(makefile.includes("MOBILE_IOS_SUBMIT_SCRIPT"), "mobile_makefile_missing_ios_submit_script");
assert(makefile.includes("MOBILE_ANDROID_PUBLISH_SCRIPT"), "mobile_makefile_missing_android_publish_script");
assert(makefile.includes('node "$(MOBILE_IOS_SUBMIT_SCRIPT)"'), "mobile_makefile_missing_ios_submit_command");
assert(makefile.includes('node "$(MOBILE_ANDROID_PUBLISH_SCRIPT)"'), "mobile_makefile_missing_android_publish_command");
assert(makefile.includes("test:api-boundaries"), "mobile_makefile_missing_api_boundary_check");
assert(makefile.includes("MOBILE_NPM_COMMAND ?= env -u NO_COLOR $(MOBILE_NPM)"), "mobile_makefile_missing_no_color_warning_guard");
assert(makefile.includes("MOBILE_NATIVE_BUILD_FINGERPRINT"), "mobile_makefile_missing_native_build_fingerprint");
assert(makefile.includes("Development build missing or stale"), "mobile_makefile_missing_stale_native_build_guard");
assert(!makefile.includes('@mkdir -p "$(dir $(ANDROID_LOCAL_PROPERTIES))"'), "mobile_makefile_creates_partial_android_project");
assert(makefile.includes("MOBILE_GOOGLE_IOS_REDIRECT_URI"), "mobile_makefile_missing_ios_google_redirect_uri_variable");
assert(makefile.includes('LOOPAWARE_MOBILE_IOS_BUNDLE_IDENTIFIER="$(MOBILE_IOS_BUNDLE_IDENTIFIER)"'), "mobile_makefile_missing_ios_bundle_env");
assert(makefile.includes('LOOPAWARE_MOBILE_ANDROID_PACKAGE="$(MOBILE_ANDROID_PACKAGE)"'), "mobile_makefile_missing_android_package_env");
assert(makefile.includes('LOOPAWARE_MOBILE_RELEASE_TIMESTAMP="$(MOBILE_RESOLVED_RELEASE_TIMESTAMP)"'), "mobile_makefile_missing_release_timestamp_env");
assert(
  makefile.includes('LOOPAWARE_MOBILE_GOOGLE_IOS_REDIRECT_URI="$(MOBILE_GOOGLE_IOS_REDIRECT_URI)"'),
  "mobile_makefile_missing_ios_google_redirect_uri_env",
);
assert(
  makefile.includes('EXPO_PACKAGER_PROXY_URL="http://localhost:$${metro_port}"'),
  "mobile_makefile_missing_ios_localhost_proxy_url",
);
assert(releaseScriptSource.includes("submit-mobile"), "mobile_release_missing_combined_store_submit");
assert(releaseScriptSource.includes("--skip-ios"), "mobile_release_missing_skip_ios");
assert(releaseScriptSource.includes("--skip-android"), "mobile_release_missing_skip_android");
assert(releaseScriptSource.includes("--skip-mobile"), "mobile_release_missing_skip_mobile");
assert(releaseScriptSource.includes('MOBILE_RELEASE_TIMESTAMP="${mobile_release_timestamp}"'), "mobile_release_missing_shared_store_timestamp");

const iosSubmitSource = readText("mobile/scripts/submit-ios.mjs");
assert(iosSubmitSource.includes("xcrun") && iosSubmitSource.includes("altool"), "mobile_ios_submit_missing_altool_upload");
assert(iosSubmitSource.includes("--upload-package"), "mobile_ios_submit_missing_package_upload");
assert(iosSubmitSource.includes("--platform"), "mobile_ios_submit_missing_platform");
assert(iosSubmitSource.includes("--apple-id"), "mobile_ios_submit_missing_asc_app_id_flag");
assert(iosSubmitSource.includes("--bundle-id"), "mobile_ios_submit_missing_bundle_id");
assert(iosSubmitSource.includes("--bundle-version"), "mobile_ios_submit_missing_bundle_version");
assert(iosSubmitSource.includes("--bundle-short-version-string"), "mobile_ios_submit_missing_short_version");
assert(iosSubmitSource.includes("MOBILE_IOS_ASC_APP_ID"), "mobile_ios_submit_missing_asc_app_id_env");
assert(iosSubmitSource.includes("APP_STORE_CONNECT_API_KEY_ID"), "mobile_ios_submit_missing_api_key_env");
assert(iosSubmitSource.includes("MOBILE_IOS_APP_SPECIFIC_PASSWORD"), "mobile_ios_submit_missing_app_password_env");
assert(iosSubmitSource.includes("API_PRIVATE_KEYS_DIR"), "mobile_ios_submit_missing_api_private_keys_dir");
assert(!iosSubmitSource.includes("--p8-file-path"), "mobile_ios_submit_must_not_pass_p8_file_path_to_upload");
assert(iosSubmitSource.includes("iOS IPA hash changed since build manifest"), "mobile_ios_submit_missing_hash_drift_guard");
assert(iosSubmitSource.includes("createMobileCalVerVersion"), "mobile_ios_submit_missing_calver_manifest_default");
assert(iosSubmitSource.includes("requireArchiveVersioning"), "mobile_ios_submit_missing_manifest_versioning_validation");
assert(iosSubmitSource.includes("versioning: manifest.versioning"), "mobile_ios_submit_must_report_manifest_versioning");

const iosArchiveSource = readText("mobile/scripts/build-ios-archive.mjs");
assert(iosArchiveSource.includes("xcodebuild"), "mobile_ios_archive_missing_xcodebuild");
assert(iosArchiveSource.includes("-exportArchive"), "mobile_ios_archive_missing_export_archive");
assert(iosArchiveSource.includes("app-store-connect"), "mobile_ios_archive_missing_app_store_connect_export");
assert(!iosArchiveSource.includes("process.env.MOBILE_IOS_BUILD_NUMBER"), "mobile_ios_archive_must_not_read_manual_build_number");
assert(iosArchiveSource.includes("createMobileCalVerVersion"), "mobile_ios_archive_missing_calver_versioning");
assert(iosArchiveSource.includes("LOOPAWARE_MOBILE_VERSION"), "mobile_ios_archive_missing_calver_version_env");
assert(iosArchiveSource.includes("LOOPAWARE_MOBILE_IOS_BUILD_NUMBER"), "mobile_ios_archive_missing_build_number_env");
assert(iosArchiveSource.includes("expo\", \"prebuild\""), "mobile_ios_archive_missing_expo_prebuild");
assert(iosArchiveSource.includes("pod-install"), "mobile_ios_archive_missing_pod_install");
assert(iosArchiveSource.includes("stripDevelopmentClientFromProductionArchive"), "mobile_ios_archive_missing_dev_client_strip");
assert(iosArchiveSource.includes('delete packageJSON.dependencies["expo-dev-client"]'), "mobile_ios_archive_must_strip_dev_client_dependency");
assert(iosArchiveSource.includes('"CODE_SIGN_IDENTITY="'), "mobile_ios_archive_must_clear_automatic_development_identity");
assert(
  iosArchiveSource.includes("automatic iOS signing does not accept MOBILE_IOS_SIGNING_CERTIFICATE"),
  "mobile_ios_archive_must_reject_automatic_signing_certificate",
);
assert(iosArchiveSource.includes("prepareSigningKeychain"), "mobile_ios_archive_missing_signing_keychain_prepare");
assert(iosArchiveSource.includes("MOBILE_IOS_SIGNING_KEYCHAIN_PASSWORD_FILE"), "mobile_ios_archive_missing_signing_keychain_password_file");
assert(iosArchiveSource.includes("set-key-partition-list"), "mobile_ios_archive_missing_codesign_partition_authorization");

const androidBundleSource = readText("mobile/scripts/build-android-bundle.mjs");
assert(androidBundleSource.includes("signingConfig signingConfigs.release"), "mobile_android_bundle_missing_release_signing");
assert(androidBundleSource.includes("android-release-identity.json"), "mobile_android_bundle_missing_release_identity");
assert(androidBundleSource.includes("verifyUploadKeyFingerprint"), "mobile_android_bundle_missing_upload_key_fingerprint_check");
assert(androidBundleSource.includes("uploadKey.sha256"), "mobile_android_bundle_missing_upload_key_sha256_contract");
assert(!androidBundleSource.includes("-genkeypair"), "mobile_android_bundle_must_not_generate_upload_key");
assert(
  androidBundleSource.includes("generated app bundle is signed with the Android debug certificate"),
  "mobile_android_bundle_missing_debug_signing_rejection",
);
assert(
  androidBundleSource.includes('"bundletool", "validate"'),
  "mobile_android_bundle_missing_bundletool_validation",
);
assert(androidBundleSource.includes("LOOPAWARE_ANDROID_UPLOAD"), "mobile_android_bundle_missing_upload_key_env");
assert(androidBundleSource.includes("loopaware-upload-key.jks"), "mobile_android_bundle_missing_default_upload_key");
assert(androidBundleSource.includes("LOOPAWARE_MOBILE_ANDROID_VERSION_CODE"), "mobile_android_bundle_missing_version_code_env");
assert(!androidBundleSource.includes("process.env.MOBILE_ANDROID_VERSION_CODE"), "mobile_android_bundle_must_not_read_manual_version_code");
assert(androidBundleSource.includes("createMobileCalVerVersion"), "mobile_android_bundle_missing_calver_versioning");
assert(androidBundleSource.includes("LOOPAWARE_MOBILE_VERSION"), "mobile_android_bundle_missing_calver_version_env");
assert(androidBundleSource.includes("writeBuildManifest"), "mobile_android_bundle_missing_build_manifest");

const mobileCalVerSource = readText("mobile/scripts/mobile-calver-version.mjs");
assert(mobileCalVerSource.includes("2100000000"), "mobile_calver_missing_google_play_version_code_maximum");
assert(mobileCalVerSource.includes("Date.UTC(2020, 0, 1"), "mobile_calver_missing_utc_epoch");
assert(mobileCalVerSource.includes("releaseVersion"), "mobile_calver_missing_release_version");

const androidPublishSource = readText("mobile/scripts/publish-android-play.mjs");
assert(androidPublishSource.includes("android-release-identity.json"), "mobile_android_publish_missing_release_identity");
assert(androidPublishSource.includes("releaseIdentity.googleCloudProjectId"), "mobile_android_publish_missing_release_identity_quota_project");
assert(androidPublishSource.includes("androidpublisher.googleapis.com/upload/androidpublisher/v3/applications"), "mobile_android_publish_missing_upload_api");
assert(androidPublishSource.includes("deobfuscationFiles/proguard"), "mobile_android_publish_missing_mapping_upload");
assert(androidPublishSource.includes("gcloud") && androidPublishSource.includes("application-default"), "mobile_android_publish_missing_adc_auth");
assert(androidPublishSource.includes("create Android Publisher edit"), "mobile_android_publish_missing_edit_create");
assert(androidPublishSource.includes("commit Android Publisher edit"), "mobile_android_publish_missing_edit_commit");
assert(androidPublishSource.includes("createMobileCalVerVersion"), "mobile_android_publish_missing_calver_artifact_default");
assert(androidPublishSource.includes("LOOPAWARE_MOBILE_VERSION"), "mobile_android_publish_missing_calver_app_config_env");

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

for (const envFile of [
  "configs/.env.tauth.example",
  "configs/.env.tauth.computercat.example",
  "tests/configs/tauth.env",
]) {
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
