// @ts-check
/// <reference types="node" />

import fs from "node:fs";
import path from "node:path";

const repoRoot = path.resolve(import.meta.dirname, "..");

const makefileSource = readText("Makefile");
const readmeSource = readText("README.md");
const normalizedReadmeSource = readmeSource.replace(/\s+/g, " ");
const releaseSource = readText("scripts/release.sh");
const publishReleaseSource = readText("scripts/publish-release.sh");
const publishReactNativeSource = readText("scripts/publish-react-native.sh");
const deploySource = readText("scripts/deploy.sh");
const releaseToolFiles = [
  "prepare_release.sh",
  "publish_release.sh",
  "release_helper.py",
  "prepare_pages_artifact.sh",
  "deploy_pages_artifact.sh",
  "prepare_container_artifact.sh",
  "publish_container_artifacts.sh",
];
const releaseBoundarySources = [
  makefileSource,
  releaseSource,
  publishReleaseSource,
  publishReactNativeSource,
];

assert(
  makefileSource.includes("release-workflow-check:") &&
    makefileSource.includes("node scripts/validate-release-workflow.mjs"),
  "release_workflow_check_missing_makefile_target",
);
assert(
  makefileSource.includes("@$(MAKE) release-workflow-check"),
  "release_workflow_check_missing_ci_wiring",
);
assert(
  makefileSource.includes("bash scripts/test-release-tooling.sh"),
  "release_pages_contract_check_missing",
);
assert(
  makefileSource.includes("RELEASE_TOOL_DIR := $(abspath $(CURDIR)/scripts/release)"),
  "release_tool_directory_must_be_repository_owned",
);
for (const source of releaseBoundarySources) {
  assert(!source.includes("agentSkills/gitrelease/scripts"), "release_sibling_tooling_path_forbidden");
}
for (const releaseToolFile of releaseToolFiles) {
  assert(
    fs.existsSync(path.join(repoRoot, "scripts/release", releaseToolFile)),
    "release_owned_tool_missing:" + releaseToolFile,
  );
}
assert(!releaseSource.includes("RELEASE_PIPELINE"), "release_pipeline_override_forbidden");
assert(
  releaseSource.includes('pipeline="${repo_root}/scripts/release/prepare_release.sh"'),
  "release_missing_owned_prepare_pipeline",
);
assert(releaseSource.includes('[[ -x "${pipeline}" ]]'), "release_pipeline_must_fail_fast");
assert(releaseSource.includes('exec "${pipeline}" "$@"'), "release_pipeline_must_forward_arguments");
assert(!publishReleaseSource.includes("PUBLISH_RELEASE_PIPELINE"), "publish_release_pipeline_override_forbidden");
assert(
  publishReleaseSource.includes('pipeline="${repo_root}/scripts/release/publish_release.sh"'),
  "publish_release_missing_owned_pipeline",
);
assert(publishReleaseSource.includes('[[ -x "${pipeline}" ]]'), "publish_release_pipeline_must_fail_fast");
assert(publishReleaseSource.includes('exec "${pipeline}" "$@"'), "publish_release_pipeline_must_forward_arguments");
assert(
  publishReactNativeSource.includes('helper="${repo_root}/scripts/release/release_helper.py"'),
  "publish_react_native_missing_owned_release_helper",
);
assert(releaseSource.includes("configs/.env.loopaware"), "release_missing_default_env_file");
assert(makefileSource.includes("RELEASE_ENV_FILE ?= $(CURDIR)/configs/.env.loopaware"), "release_makefile_missing_env_file_default");
assert(makefileSource.includes('RELEASE_ENV_FILE="$(RELEASE_ENV_FILE)"'), "release_makefile_must_pass_env_file");
assert(makefileSource.includes("RELEASE_ARTIFACT_TARGETS ?= mobile-release-artifacts"), "release_missing_mobile_artifact_contract");
assert(makefileSource.includes("mobile-release-artifacts:"), "release_missing_mobile_artifact_target");
assert(makefileSource.includes("client-react-native-artifact"), "release_missing_react_native_package_artifact");
assert(makefileSource.includes("publish: publish-release"), "publish_missing_prepared_release_dependency");
assert(makefileSource.includes("./scripts/publish-mobile.sh"), "publish_missing_mobile_upload");
assert(makefileSource.includes("./scripts/publish-react-native.sh"), "publish_missing_react_native_package_upload");
assert(!releaseSource.includes("git push"), "release_must_not_push_git_refs");
assert(!releaseSource.includes("submit-mobile"), "release_must_not_upload_mobile_stores");
assert(!releaseSource.includes("gh "), "release_must_not_call_github");

const submitIosBlock = makefileSource.slice(
  makefileSource.indexOf("submit-ios: submit-ios-preflight"),
  makefileSource.indexOf("submit-android: mobile-check"),
);
assert(!submitIosBlock.includes("build-ios"), "publish_ios_must_consume_prepared_artifact");
const submitAndroidBlock = makefileSource.slice(
  makefileSource.indexOf("submit-android: mobile-check"),
  makefileSource.indexOf("submit-mobile:"),
);
assert(!submitAndroidBlock.includes("mobile-android-bundle"), "publish_android_must_consume_prepared_artifact");

const deployImageVerifyIndex = deploySource.indexOf('if [[ "${SKIP_IMAGE_VERIFY}" != "true"');
assert(deployImageVerifyIndex >= 0, "deploy_missing_image_verification");
assert(!deploySource.includes("make ci"), "deploy_must_not_run_ci_or_rebuild_artifacts");
assert(!deploySource.includes("SKIP_CI"), "deploy_must_not_expose_legacy_ci_toggle");
assert(!deploySource.includes("publish_container_artifacts"), "deploy_must_not_publish_containers");
assert(deploySource.includes("make --no-print-directory pages-deploy"), "deploy_missing_pages_activation");
assert(
  deploySource.includes("is not published in the registry; run make publish"),
  "deploy_missing_publish_recovery_message",
);
assert(deploySource.includes("2>&1"), "deploy_image_inspect_must_capture_errors");

assert(normalizedReadmeSource.includes("`make release` prepares"), "readme_missing_local_release_contract");
assert(normalizedReadmeSource.includes("`make publish` publishes"), "readme_missing_publish_contract");

console.log("release workflow validation passed");

/**
 * @param {string} relativePath
 * @returns {string}
 */
function readText(relativePath) {
  return fs.readFileSync(path.join(repoRoot, relativePath), "utf8");
}

/**
 * @param {unknown} condition
 * @param {string} message
 * @returns {asserts condition}
 */
function assert(condition, message) {
  if (!condition) {
    throw new Error(message);
  }
}
