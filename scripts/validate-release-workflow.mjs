// @ts-check
/// <reference types="node" />

import fs from "node:fs";
import path from "node:path";

const repoRoot = path.resolve(import.meta.dirname, "..");

const makefileSource = readText("Makefile");
const readmeSource = readText("README.md");
const normalizedReadmeSource = readmeSource.replace(/\s+/g, " ");
const releaseSource = readText("scripts/release.sh");
const deploySource = readText("scripts/deploy.sh");

assert(
  makefileSource.includes("release-workflow-check:") &&
    makefileSource.includes("node scripts/validate-release-workflow.mjs"),
  "release_workflow_check_missing_makefile_target",
);
assert(
  makefileSource.includes("@$(MAKE) release-workflow-check"),
  "release_workflow_check_missing_ci_wiring",
);

const releaseAlreadyExistsIndex = releaseSource.indexOf("release_already_exists=true");
const releaseCiIndex = releaseSource.indexOf('echo "==> [release] Running make ci"');
assert(releaseAlreadyExistsIndex >= 0, "release_missing_already_exists_noop");
assert(releaseCiIndex >= 0, "release_missing_ci_gate");
assert(releaseAlreadyExistsIndex < releaseCiIndex, "release_noop_must_precede_ci");
assert(
  releaseSource.includes('git log --format=%H "${boundary_tag}..HEAD" --'),
  "release_missing_boundary_commit_check",
);
assert(
  releaseSource.includes("Run make publish to publish or repair Docker images"),
  "release_noop_missing_publish_recovery_message",
);

const deployImageVerifyIndex = deploySource.indexOf('if [[ "${SKIP_IMAGE_VERIFY}" != "true"');
const deployCiIndex = deploySource.indexOf('if [[ "${SKIP_CI}" != "true"');
assert(deployImageVerifyIndex >= 0, "deploy_missing_image_verification");
assert(deployCiIndex >= 0, "deploy_missing_ci_gate");
assert(deployImageVerifyIndex < deployCiIndex, "deploy_image_verification_must_precede_ci");
assert(
  deploySource.includes("is not published in the registry; run make publish"),
  "deploy_missing_publish_recovery_message",
);
assert(deploySource.includes("2>&1"), "deploy_image_inspect_must_capture_errors");

assert(
  normalizedReadmeSource.includes("If `make release` finds that `HEAD` is already covered by the current release tag"),
  "readme_missing_release_noop_contract",
);
assert(
  normalizedReadmeSource.includes("Do not create an empty release to repair a missing image"),
  "readme_missing_missing_image_recovery_contract",
);

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
