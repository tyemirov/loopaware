// @ts-check
/// <reference types="node" />

import fs from "node:fs";
import path from "node:path";

const repoRoot = path.resolve(import.meta.dirname, "..");

const makefileSource = readText("Makefile");
const ciWorkflowSource = readText(".github/workflows/ci.yml");
const macosReleaseContractJobSource = workflowJob("release-contract-macos");
const dockerignoreSource = readText(".dockerignore");
const readmeSource = readText("README.md");
const normalizedReadmeSource = readmeSource.replace(/\s+/g, " ");
const releaseSource = readText("scripts/release.sh");
const repositoryIdentitySource = readText("scripts/release/repository_identity.sh");
const prepareReleaseSource = readText("scripts/release/prepare_release.sh");
const publishReleaseSource = readText("scripts/publish-release.sh");
const publishOrchestratorSource = readText("scripts/publish.sh");
const releasePreflightSource = readText("scripts/release-preflight.sh");
const publishPreflightSource = readText("scripts/publish-preflight.sh");
const publishMobileSource = readText("scripts/publish-mobile.sh");
const publishReactNativeSource = readText("scripts/publish-react-native.sh");
const releaseEnvParserSource = readText("scripts/release/parse_release_env.py");
const releaseEnvLoaderSource = readText("scripts/release/load_release_env.sh");
const deploySource = readText("scripts/deploy.sh");
const appAnsibleRunnerSource = readText("scripts/run-app-ansible-deploy.sh");
const appDeployComposeSource = readText(".mprlab/deploy/docker-compose.yml");
const appDeployPlaybookSource = readText(".mprlab/deploy/ansible/playbooks/deploy.yml");
const appDeployValidateSource = readText(".mprlab/deploy/ansible/tasks/validate.yml");
const appDeployPreflightSource = readText(".mprlab/deploy/ansible/tasks/preflight.yml");
const appDeployTaskSource = readText(".mprlab/deploy/ansible/tasks/deploy.yml");
const appDeployVerifySource = readText(".mprlab/deploy/ansible/tasks/verify.yml");
const pagesDeploySource = readText("scripts/release/deploy_pages_artifact.sh");
const containerPublishSource = readText("scripts/release/publish_container_artifacts.sh");
const containerPrepareSource = readText("scripts/release/prepare_container_artifact.sh");
const dockerIdentitySource = readText("scripts/release/docker_identity.sh");
const stagedArtifactVerifierSource = readText("scripts/release/verify_staged_artifacts.py");
const lifecycleLockSource = readText("scripts/release/with_lifecycle_lock.sh");
const lifecycleRunnerSource = readText("scripts/release/run_lifecycle.sh");
const publicationAttestationSource = readText("scripts/release/record_publication.sh");
const releaseHelperSource = readText("scripts/release/release_helper.py");
const integrationRunnerSource = readText("tests/scripts/run-integration.sh");
const deployResourcesSource = readText(".mprlab/deploy/resources.yml");
const androidPublishSource = readText("mobile/scripts/publish-android-play.mjs");
const iosSubmitSource = readText("mobile/scripts/submit-ios.mjs");
const iosBuildSource = readText("mobile/scripts/build-ios-archive.mjs");
const androidBuildSource = readText("mobile/scripts/build-android-bundle.mjs");
const mobilePackage = JSON.parse(readText("mobile/package.json"));
const mobilePackageLock = JSON.parse(readText("mobile/package-lock.json"));
const releaseToolFiles = [
  "prepare_release.sh",
  "publish_release.sh",
  "release_helper.py",
  "prepare_pages_artifact.sh",
  "deploy_pages_artifact.sh",
  "prepare_container_artifact.sh",
  "publish_container_artifacts.sh",
  "docker_identity.sh",
  "verify_staged_artifacts.py",
  "with_lifecycle_lock.sh",
  "parse_release_env.py",
  "load_release_env.sh",
  "repository_identity.sh",
  "run_lifecycle.sh",
  "record_publication.sh",
];
const releaseBoundarySources = [
  makefileSource,
  releaseSource,
  publishReleaseSource,
  publishOrchestratorSource,
  publishReactNativeSource,
];
const pythonReleaseHelperCallers = [
  releasePreflightSource,
  publishMobileSource,
  publishReactNativeSource,
  prepareReleaseSource,
  containerPublishSource,
  publishReleaseSource,
];

assert(
  releaseHelperSource.startsWith("#!/usr/bin/env python3\n") &&
    !releaseHelperSource.includes("uv run --script") &&
    pythonReleaseHelperCallers.every((source) => !source.includes("UV_CACHE_DIR")),
  "release_helper_must_use_the_standard_python_runtime_without_uv",
);
assert(
  ciWorkflowSource.includes("PYTHON_VERSION: '3.11'") &&
    ciWorkflowSource.includes("uses: actions/setup-python@v6") &&
    ciWorkflowSource.includes("python-version: ${{ env.PYTHON_VERSION }}") &&
    (ciWorkflowSource.match(/- 'scripts\/\*\*'/g) || []).length === 2,
  "ci_must_pin_python_and_run_for_release_script_changes",
);
assert(
  macosReleaseContractJobSource.includes("name: Release contract (macOS)") &&
    macosReleaseContractJobSource.includes("runs-on: macos-15") &&
    macosReleaseContractJobSource.includes("uses: actions/setup-node@v6") &&
    macosReleaseContractJobSource.includes("uses: actions/setup-python@v6") &&
    macosReleaseContractJobSource.includes("brew install bash coreutils") &&
    macosReleaseContractJobSource.includes('echo "$(brew --prefix)/bin" >> "$GITHUB_PATH"') &&
    macosReleaseContractJobSource.includes('echo "$(brew --prefix coreutils)/libexec/gnubin" >> "$GITHUB_PATH"') &&
    macosReleaseContractJobSource.includes("run: make release-workflow-check"),
  "release_workflow_contract_must_run_on_the_production_macos_toolchain",
);

assert(
  makefileSource.includes("release-workflow-check:") &&
    makefileSource.includes("node scripts/validate-release-workflow.mjs"),
  "release_workflow_check_missing_makefile_target",
);
assert(
  makefileSource.includes("lint-js: client-react-native-check mobile-check release-workflow-check"),
  "release_workflow_check_missing_ci_wiring",
);
assert(
  makefileSource.includes("bash scripts/test-release-tooling.sh"),
  "release_pages_contract_check_missing",
);
assert(
  makefileSource.includes("bash scripts/test-deploy-dry-run.sh"),
  "deploy_dry_run_contract_check_missing",
);
assert(
  makefileSource.includes("bash scripts/test-publish-preflight.sh"),
  "publish_preflight_contract_check_missing",
);
assert(
  makefileSource.includes("bash scripts/test-staged-release-artifacts.sh"),
  "staged_release_artifact_contract_check_missing",
);
assert(
  makefileSource.includes("bash scripts/test-ios-npm-publication.sh"),
  "ios_npm_publication_contract_check_missing",
);
assert(
  makefileSource.includes("bash scripts/test-lifecycle-orchestration.sh"),
  "lifecycle_orchestration_contract_check_missing",
);
assert(
  makefileSource.includes("override RELEASE_TOOL_DIR := $(abspath $(CURDIR)/scripts/release)"),
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
assert(!prepareReleaseSource.includes("RELEASE_HELPER"), "release_helper_override_forbidden");
assert(!prepareReleaseSource.includes("RELEASE_BUMP") && !prepareReleaseSource.includes("RELEASE_SCHEME"), "release_selection_environment_overrides_forbidden");
assert(!prepareReleaseSource.includes("    --scheme)"), "release_calver_scheme_override_forbidden");
assert(!readText("scripts/release/publish_release.sh").includes("RELEASE_HELPER"), "publish_release_helper_override_forbidden");
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
assert(
  releaseSource.includes("assert_canonical_github_origin") &&
    publishReleaseSource.includes("assert_canonical_github_origin") &&
    repositoryIdentitySource.includes('"git@github.com:${repository}"|"git@github.com:${repository}.git"') &&
    repositoryIdentitySource.includes('"https://github.com/${repository}"|"https://github.com/${repository}.git"'),
  "release_and_publish_must_verify_the_canonical_repository_identity",
);
assert(
  releaseSource.includes("assert_no_github_repository_override") &&
    publishReleaseSource.includes("assert_no_github_repository_override") &&
    deploySource.includes("assert_no_github_repository_override") &&
    repositoryIdentitySource.includes("GH_REPO is not supported by the canonical lifecycle") &&
    repositoryIdentitySource.includes("GH_HOST must be github.com") &&
    repositoryIdentitySource.includes("origin pushurl overrides are not supported") &&
    repositoryIdentitySource.includes("remote get-url --all origin") &&
    repositoryIdentitySource.includes("remote get-url --push --all origin") &&
    repositoryIdentitySource.includes("effective origin fetch URL") &&
    repositoryIdentitySource.includes("effective origin push URL"),
  "lifecycle_github_repository_must_be_explicit_and_non_overridable",
);
assert(
  releaseSource.includes('assert_remote_default_and_release_tags "${repo_root}" LoopAware') &&
    repositoryIdentitySource.includes("git -C \"${directory}\" ls-remote --symref origin") &&
    repositoryIdentitySource.includes('[[ "${local_head}" == "${remote_default_sha}" ]]') &&
    repositoryIdentitySource.includes('pending_local_tag=""') &&
    repositoryIdentitySource.includes('tag --delete "${local_tag}"') &&
    repositoryIdentitySource.includes("fetch --force --no-tags --no-write-fetch-head") &&
    repositoryIdentitySource.includes('[[ "${local_tag_commit}" == "${remote_tag_commit}" ]]'),
  "release_must_synchronize_and_pin_remote_release_state_before_preparation",
);
assert(
  makefileSource.includes("override RELEASE_ENV_FILE := $(CURDIR)/configs/.env.loopaware") &&
    makefileSource.includes("override RELEASE_ENV_FILE := $(value RELEASE_ENV_FILE)"),
  "release_makefile_missing_env_file_default",
);
assert(makefileSource.includes("export RELEASE_ENV_FILE"), "release_makefile_must_export_env_file_without_recipe_interpolation");
assert(
  makefileSource.includes(
    "override RELEASE_ARTIFACT_TARGETS := mobile-release-artifacts client-react-native-artifact container-artifacts pages-artifact",
  ),
  "release_missing_canonical_artifact_contract",
);
assert(
  releaseEnvParserSource.includes("EXPORTED_KEYS = {") &&
    releaseEnvParserSource.includes("APPLICATION_KEYS = {") &&
    releaseEnvParserSource.includes("key {key} is not part of the release env contract") &&
    releaseEnvParserSource.includes("duplicate key") &&
    releaseEnvParserSource.includes("shlex.quote(values[key])") &&
    !releaseEnvParserSource.includes('"PATH",') &&
    !releaseEnvParserSource.includes('"MAKEFLAGS",') &&
    !releaseEnvParserSource.includes('"BASH_ENV",') &&
    releaseEnvLoaderSource.includes('"${loader_directory}/parse_release_env.py" "${env_file}" >"${export_file}"') &&
    releaseEnvLoaderSource.includes('source "${export_file}"'),
  "release_env_must_be_parsed_as_allowlisted_data_before_export",
);
for (const source of [releaseSource, releasePreflightSource, publishMobileSource, publishReactNativeSource]) {
  assert(source.includes("load_release_env_file"), "release_env_consumer_missing_strict_loader");
  assert(!source.includes('source "${env_file}"'), "release_env_must_not_be_sourced_as_shell_code");
}
assert(makefileSource.includes("override DOCKER_IMAGE := ghcr.io/tyemirov/loopaware"), "release_missing_canonical_container_image");
assert(makefileSource.includes("override PUBLISH_PLATFORMS := linux/amd64"), "release_missing_canonical_container_platform");
assert(makefileSource.includes("override PAGES_URL := https://loopaware.mprlab.com/"), "release_missing_canonical_pages_url");
assert(makefileSource.includes("override PAGES_BRANCH := gh-pages"), "release_missing_canonical_pages_branch");
assert(makefileSource.includes("override PAGES_DOMAIN := loopaware.mprlab.com"), "release_missing_canonical_pages_domain");
assert(makefileSource.includes("override MOBILE_RESOLVED_RELEASE_TIMESTAMP :="), "release_mobile_timestamp_must_not_be_overridable");
assert(
  makefileSource.includes("override MOBILE_GOOGLE_IOS_REDIRECT_URI := com.googleusercontent.apps.281540686395-8a90ldjnklddl0qpoc8ur6620lguv7mg:/oauth2redirect/google") &&
    makefileSource.includes("override MOBILE_IOS_DEVELOPMENT_TEAM := Z9ZW6HDGML"),
  "release_mobile_identity_must_be_canonical",
);
assert(
  deployResourcesSource.includes("image: ghcr.io/tyemirov/loopaware:latest") &&
    deployResourcesSource.includes("target: pages-deploy") &&
    deployResourcesSource.includes("url: https://loopaware.mprlab.com/"),
  "deploy_resource_inventory_drifted_from_lifecycle_contract",
);
assert(makefileSource.includes("mobile-release-artifacts:"), "release_missing_mobile_artifact_target");
assert(makefileSource.includes("client-react-native-artifact"), "release_missing_react_native_package_artifact");
assert(
  makefileSource.includes("override CLIENT_REACT_NATIVE_NPM := npm") &&
    makefileSource.includes("override CLIENT_REACT_NATIVE_NPM_COMMAND := env -u NO_COLOR npm") &&
    makefileSource.includes("override MOBILE_NPM := npm") &&
    makefileSource.includes("override MOBILE_NPM_COMMAND := env -u NO_COLOR npm") &&
    publishReactNativeSource.includes('npm_command="npm"'),
  "release_and_publish_package_managers_must_be_canonical",
);
assert(
  makefileSource.includes("override NPM_CONFIG_CACHE := $(CURDIR)/.cache/npm") &&
    makefileSource.includes("export NPM_CONFIG_CACHE") &&
    makefileSource.includes("override GOCACHE := $(CURDIR)/.cache/go-build") &&
    makefileSource.includes("export GOCACHE"),
  "make_owned_build_commands_must_use_repository_caches",
);
assert(
  makefileSource.includes('git archive --output "$$archive" "$$RELEASE_SOURCE_COMMIT:clients/react-native"') &&
    makefileSource.includes('git archive --output "$$archive" "$$RELEASE_SOURCE_COMMIT:mobile"') &&
    makefileSource.includes('git archive --output "$$archive" "$$RELEASE_SOURCE_COMMIT"') &&
    makefileSource.includes('git archive --output "$$archive" "$$RELEASE_SOURCE_COMMIT:web"') &&
    makefileSource.includes("--platforms linux/amd64 --pull"),
  "release_artifacts_must_build_from_the_exact_source_commit",
);
assert(
  makefileSource.includes("export LOOPAWARE_MOBILE_API_BASE_URL") &&
    makefileSource.includes("export LOOPAWARE_MOBILE_TAUTH_BASE_URL") &&
    makefileSource.includes("export LOOPAWARE_MOBILE_TAUTH_TENANT_ID") &&
    makefileSource.includes("export LOOPAWARE_MOBILE_GOOGLE_IOS_REDIRECT_URI") &&
    makefileSource.includes("export LOOPAWARE_MOBILE_RELEASE_TIMESTAMP") &&
    iosBuildSource.includes('name.startsWith("EXPO_PUBLIC_")') &&
    androidBuildSource.includes('name.startsWith("EXPO_PUBLIC_")') &&
    iosBuildSource.includes("runtimeConfig: appConfig.runtimeConfig") &&
    androidBuildSource.includes("runtimeConfig: appConfig.runtimeConfig") &&
    stagedArtifactVerifierSource.includes("iOS build manifest has a noncanonical runtime configuration") &&
    stagedArtifactVerifierSource.includes("Android build manifest has a noncanonical runtime configuration") &&
    stagedArtifactVerifierSource.includes("iOS build manifest has a noncanonical signing configuration"),
  "release_mobile_build_environment_must_be_canonical",
);
assert(
  mobilePackage.devDependencies?.["pod-install"] === "1.1.0" &&
    mobilePackageLock.packages?.[""]?.devDependencies?.["pod-install"] === "1.1.0" &&
    mobilePackageLock.packages?.["node_modules/pod-install"]?.version === "1.1.0" &&
    iosBuildSource.includes('run(["npm", "ci", "--include=dev"]') &&
    iosBuildSource.includes('run(["npx", "--no-install", "pod-install", "ios"]'),
  "release_ios_archive_must_install_and_use_locked_pod_install",
);
const mobileArtifactRecipe = makeRecipe("mobile-release-artifacts");
assert(
  mobileArtifactRecipe.includes("set -e;") &&
    mobileArtifactRecipe.indexOf("build-ios-archive.mjs") < mobileArtifactRecipe.indexOf("build-android-bundle.mjs"),
  "release_mobile_artifact_builders_must_fail_fast_in_ios_then_android_order",
);
const reactNativeArtifactRecipe = makeRecipe("client-react-native-artifact");
assert(
  reactNativeArtifactRecipe.includes("set -e;") &&
    reactNativeArtifactRecipe.includes('source_dir="$$(mktemp -d)"') &&
    reactNativeArtifactRecipe.includes('git archive --output "$$archive" "$$RELEASE_SOURCE_COMMIT:clients/react-native"') &&
    reactNativeArtifactRecipe.includes('(cd "$$source_dir" && env -u NO_COLOR npm ci --legacy-peer-deps)') &&
    reactNativeArtifactRecipe.includes('(cd "$$source_dir" && env -u NO_COLOR npm run typecheck)') &&
    reactNativeArtifactRecipe.includes('(cd "$$source_dir" && env -u NO_COLOR npm run build)') &&
    reactNativeArtifactRecipe.includes('(cd "$$source_dir" && env -u NO_COLOR npm run verify-package)') &&
    !reactNativeArtifactRecipe.includes('npm --prefix "$$source_dir"') &&
    reactNativeArtifactRecipe.includes('npm pack --ignore-scripts --pack-destination "$$asset_dir"'),
  "react_native_release_package_must_fail_fast_in_clean_exact_commit_checkout",
);
assert(
  !makefileSource.includes("loopaware-ios.json $(MOBILE_IOS_ARCHIVE_ARGS)") &&
    !makefileSource.includes("loopaware-android.aab $(MOBILE_ANDROID_BUNDLE_ARGS)") &&
    makefileSource.includes("MOBILE_IOS_ARCHIVE_ARGS is not supported") &&
    makefileSource.includes("MOBILE_ANDROID_BUNDLE_ARGS is not supported") &&
    makefileSource.includes("PAGES_DEPLOY_ARGS is not supported"),
  "canonical_mobile_release_outputs_must_not_accept_appended_overrides",
);
for (const target of [
  "build-ios",
  "mobile-android-bundle",
  "mobile-release-artifacts",
  "submit-ios",
  "submit-android",
  "publish-mobile",
]) {
  assert(
    !/\$\((?:APP_STORE_CONNECT|MOBILE|LOOPAWARE_MOBILE|ANDROID_(?:HOME|SDK_ROOT|STUDIO_JAVA_HOME))/.test(makeRecipe(target)),
    `canonical_mobile_recipe_must_not_interpolate_sensitive_make_variables:${target}`,
  );
}
assert(
  dockerignoreSource.includes("configs/AuthKey_*.p8") &&
    dockerignoreSource.includes("configs/client_secret_*.json") &&
    dockerignoreSource.includes("mobile/android/") &&
    dockerignoreSource.includes("mobile/ios/"),
  "docker_context_secret_and_native_output_exclusions_missing",
);
assert(
  prepareReleaseSource.includes('RELEASE_SOURCE_COMMIT="${source_commit}"') &&
    makefileSource.includes('org.opencontainers.image.revision=$$RELEASE_SOURCE_COMMIT') &&
    makefileSource.includes('org.opencontainers.image.version=$$RELEASE_VERSION') &&
    makefileSource.includes("org.opencontainers.image.source=https://github.com/tyemirov/loopaware"),
  "release_container_provenance_labels_missing",
);
assert(
  prepareReleaseSource.includes("canonical_artifact_targets=") &&
    prepareReleaseSource.includes("verify_staged_artifacts.py") &&
    releasePreflightSource.includes("verify_staged_artifacts.py") &&
    stagedArtifactVerifierSource.includes("exact canonical nine-file set") &&
    stagedArtifactVerifierSource.includes("mobile artifacts do not use the staging release timestamp"),
  "release_must_verify_the_exact_staged_artifact_contract",
);
assert(
  prepareReleaseSource.includes('MOBILE_RELEASE_TIMESTAMP="${release_timestamp}"') &&
    releasePreflightSource.includes('MOBILE_RELEASE_TIMESTAMP="${release_timestamp}"'),
  "release_must_pass_the_owned_mobile_timestamp_as_a_recursive_make_assignment",
);
assert(
  prepareReleaseSource.includes("LoopAware releases must use deployable stable vMAJOR.MINOR.PATCH versions"),
  "release_must_select_only_deployable_stable_versions",
);
assert(
  prepareReleaseSource.includes("head_release_tags=()") &&
    prepareReleaseSource.includes("multiple stable release tags point at HEAD") &&
    prepareReleaseSource.includes("prepared release commit must have exactly one source parent") &&
    prepareReleaseSource.includes('[[ "${prepared_changed_files}" == "CHANGELOG.md" ]]') &&
    prepareReleaseSource.includes('"${helper}" verify-release-artifact') &&
    prepareReleaseSource.includes("prepared release manifest does not contain the exact canonical nine-file payload set") &&
    prepareReleaseSource.includes('echo "release_already_prepared=true"') &&
    releasePreflightSource.includes('grep -Fxq "release_already_prepared=true"'),
  "release_rerun_must_recognize_only_an_exact_prepared_release",
);
assert(
  prepareReleaseSource.includes('insert-changelog --version "${next_version}" --notes-file "${notes_file}"') &&
    releaseHelperSource.includes("RELEASE_COMMIT_SUBJECT_RE.fullmatch(subject) is None") &&
    releaseHelperSource.includes("release notes heading does not match the selected changelog version") &&
    releaseHelperSource.includes("for section_start, section_end in reversed(stale_sections)") &&
    releaseHelperSource.includes('changelog.add_argument("--version", required=True)'),
  "release_changelog_must_replace_the_selected_unpublished_version_canonically",
);
assert(
  !prepareReleaseSource.includes("RELEASE_CI_TIMEOUT") &&
    prepareReleaseSource.includes('echo "==> [release] Running make ci"\nmake ci') &&
    releasePreflightSource.includes('echo "==> [release-preflight] Running the release CI gate"\nmake ci') &&
    prepareReleaseSource.includes("git var GIT_AUTHOR_IDENT") &&
    prepareReleaseSource.includes("git var GIT_COMMITTER_IDENT") &&
    prepareReleaseSource.includes('git commit --no-verify --no-gpg-sign -m "Release ${next_version}"') &&
    prepareReleaseSource.includes('git tag --no-sign -a "${next_version}"'),
  "release_dry_run_and_release_must_share_deterministic_ci_and_git_inputs",
);
assert(
  makefileSource.includes("RELEASE_ARGS is not supported") &&
    makefileSource.includes("PUBLISH_RELEASE_ARGS is not supported") &&
    makefileSource.includes("DEPLOY_ARGS is not supported") &&
    !makefileSource.includes("$(RELEASE_ARGS)") &&
    !makefileSource.includes("$(PUBLISH_RELEASE_ARGS)") &&
    !makefileSource.includes("$(DEPLOY_ARGS)"),
  "lifecycle_targets_must_reject_raw_shell_arguments",
);
const publishPreflightIndex = publishOrchestratorSource.indexOf("./scripts/publish-preflight.sh");
const publishReleaseIndex = publishOrchestratorSource.indexOf("./scripts/publish-release.sh");
const publishContainerIndex = publishOrchestratorSource.indexOf("./scripts/release/publish_container_artifacts.sh");
const publishMobileIndex = publishOrchestratorSource.indexOf("./scripts/publish-mobile.sh");
const publishReactNativeIndex = publishOrchestratorSource.indexOf("./scripts/publish-react-native.sh");
const publicationAttestationIndex = publishOrchestratorSource.lastIndexOf("./scripts/release/record_publication.sh");
assert(publishPreflightIndex >= 0, "publish_missing_preflight");
assert(publishReleaseIndex > publishPreflightIndex, "publish_must_preflight_before_github_mutation");
assert(publishContainerIndex > publishReleaseIndex, "publish_container_order_invalid");
assert(publishReactNativeIndex > publishContainerIndex, "publish_react_native_order_invalid");
assert(publishMobileIndex > publishReactNativeIndex, "store_uploads_must_be_the_last_publication_stage");
assert(publicationAttestationIndex > publishMobileIndex, "publication_attestation_must_follow_every_provider_stage");
assert(
  publishOrchestratorSource.includes("expected_manifest_sha256") &&
    publishOrchestratorSource.includes("assert_manifest_unchanged") &&
    publishOrchestratorSource.includes("prepared release manifest changed during publication"),
  "publish_must_pin_one_release_manifest_across_all_stages",
);
const canonicalLifecycleRecipes = new Map([
  ["release", "@/bin/sh ./scripts/release/run_lifecycle.sh ./scripts/release/with_lifecycle_lock.sh release ./scripts/release.sh"],
  ["release-dry-run", "@/bin/sh ./scripts/release/run_lifecycle.sh ./scripts/release/with_lifecycle_lock.sh release-dry-run ./scripts/release-preflight.sh"],
  ["publish-preflight", "@/bin/sh ./scripts/release/run_lifecycle.sh ./scripts/release/with_lifecycle_lock.sh publish-preflight ./scripts/publish-preflight.sh"],
  ["publish-dry-run", "@/bin/sh ./scripts/release/run_lifecycle.sh ./scripts/release/with_lifecycle_lock.sh publish-dry-run ./scripts/publish-preflight.sh"],
  ["publish", "@/bin/sh ./scripts/release/run_lifecycle.sh ./scripts/release/with_lifecycle_lock.sh publish ./scripts/publish.sh"],
  ["deploy", "@/bin/sh ./scripts/release/run_lifecycle.sh ./scripts/release/with_lifecycle_lock.sh deploy ./scripts/deploy.sh"],
  ["deploy-dry-run", "@/bin/sh ./scripts/release/run_lifecycle.sh ./scripts/release/with_lifecycle_lock.sh deploy-dry-run ./scripts/deploy.sh --dry-run"],
]);
assert(
  lifecycleLockSource.includes('lock_dir="${git_common_dir}/mprlab-lifecycle.lock"') &&
    lifecycleLockSource.includes('if ! mkdir "${lock_dir}"'),
  "canonical_lifecycle_lock_missing",
);
assert(
  deploySource.includes('fi\n\nassert_loopaware_unchanged "${loopaware_remote_default_sha}"') &&
    deploySource.match(/assert_loopaware_unchanged "\$\{loopaware_remote_default_sha\}"/g)?.length >= 5,
  "deploy_must_reassert_loopaware_after_the_app_owned_backend_handoff",
);
assert(
  makefileSource.includes("override SHELL := /bin/sh") &&
    makefileSource.includes("lifecycle targets reject Make's no-execute mode") &&
    makefileSource.includes("lifecycle targets reject Make's ignore-errors mode") &&
    makefileSource.includes("override MOBILE_RELEASE_TIMESTAMP := $(value MOBILE_RELEASE_TIMESTAMP)") &&
    makefileSource.includes("override ANDROID_SDK_ROOT := $(value ANDROID_SDK_ROOT)") &&
    lifecycleRunnerSource.includes("BASH_ENV ENV NODE_OPTIONS NODE_PATH PYTHONHOME PYTHONPATH DOCKER_HOST DOCKER_CONTEXT DOCKER_TLS_VERIFY DOCKER_CERT_PATH") &&
    lifecycleRunnerSource.includes("$1 ~ /^BASH_FUNC_/") &&
    lifecycleRunnerSource.includes('$1 == "SHELLOPTS"') &&
    lifecycleRunnerSource.includes('$1 == "BASHOPTS"') &&
    lifecycleRunnerSource.includes("lifecycle requires Bash from a canonical system or Homebrew path") &&
    lifecycleRunnerSource.includes("Bash 4 or newer"),
  "lifecycle_must_reject_make_and_runtime_startup_bypasses",
);
for (const [target, expectedRecipe] of canonicalLifecycleRecipes) {
  const recipe = makeRecipe(target);
  assert(recipe.trim() === expectedRecipe, `canonical_lifecycle_recipe_drifted:${target}`);
  assert(!recipe.includes("$("), `canonical_lifecycle_recipe_must_not_interpolate_make_variables:${target}`);
}
assert(makefileSource.includes("./scripts/publish-mobile.sh"), "publish_missing_mobile_upload");
assert(makefileSource.includes("./scripts/publish-react-native.sh"), "publish_missing_react_native_package_upload");
assert(!releaseSource.includes("git push"), "release_must_not_push_git_refs");
assert(
  releaseHelperSource.includes('["git", "push", "--atomic", args.remote, *push_refspecs]') &&
    releaseHelperSource.includes('push_refspecs.append(f"HEAD:refs/heads/{default_branch}")') &&
    releaseHelperSource.includes('push_refspecs.append(f"refs/tags/{version}:refs/tags/{version}")'),
  "publication_must_push_the_release_branch_and_tag_atomically",
);
assert(!releaseSource.includes("submit-mobile"), "release_must_not_upload_mobile_stores");
assert(!releaseSource.includes("gh "), "release_must_not_call_github");

const submitIosBlock = makefileSource.slice(
  makefileSource.indexOf("submit-ios: mobile-check"),
  makefileSource.indexOf("submit-android: mobile-check"),
);
assert(!submitIosBlock.includes("build-ios"), "publish_ios_must_consume_prepared_artifact");
assert(
  submitIosBlock.indexOf("submit-ios.mjs --mobile-dir mobile --dry-run") <
    submitIosBlock.lastIndexOf("submit-ios.mjs --mobile-dir mobile"),
  "publish_ios_must_validate_exact_artifact_before_upload",
);
const submitAndroidBlock = makefileSource.slice(
  makefileSource.indexOf("submit-android: mobile-check"),
  makefileSource.indexOf("submit-mobile:"),
);
assert(!submitAndroidBlock.includes("mobile-android-bundle"), "publish_android_must_consume_prepared_artifact");

const deployImageVerifyIndex = deploySource.indexOf('release_digest="$(image_digest');
assert(deployImageVerifyIndex >= 0, "deploy_missing_image_verification");
assert(!deploySource.includes("make ci"), "deploy_must_not_run_ci_or_rebuild_artifacts");
assert(!deploySource.includes("SKIP_CI"), "deploy_must_not_expose_legacy_ci_toggle");
assert(!deploySource.includes("publish_container_artifacts"), "deploy_must_not_publish_containers");
assert(
  deploySource.includes('"${repo_root}/scripts/release/deploy_pages_artifact.sh"'),
  "deploy_missing_owned_pages_activation",
);
assert(
  deploySource.includes("is not published in the registry; run make publish"),
  "deploy_missing_publish_recovery_message",
);
assert(deploySource.includes("2>&1"), "deploy_image_inspect_must_capture_errors");
assert(makefileSource.includes("deploy-dry-run:"), "deploy_dry_run_missing_makefile_target");
assert(deploySource.includes("--dry-run"), "deploy_missing_dry_run_option");
assert(
  deploySource.includes("LOOPAWARE_DEPLOY_IMAGE_REF is derived from the published release and cannot be overridden") &&
    appAnsibleRunnerSource.includes("ANSIBLE_CONFIG is owned by the LoopAware deployment controller") &&
    appAnsibleRunnerSource.includes("use LOOPAWARE_ANSIBLE_INVENTORY for the canonical app deployment inventory"),
  "deploy_app_owned_inputs_must_be_derived_and_non_overridable",
);
const pagesPreflightIndex = deploySource.indexOf('\n  --verify-only\n');
const appBackendHandoffIndex = deploySource.indexOf('"${repo_root}/scripts/run-app-ansible-deploy.sh"');
const dryRunExitIndex = deploySource.indexOf('if [[ "${DRY_RUN}" == "true" ]]; then\n  echo "LoopAware deploy dry run passed');
const pagesActivationIndex = deploySource.lastIndexOf('"${repo_root}/scripts/release/deploy_pages_artifact.sh"');
assert(pagesPreflightIndex >= 0, "deploy_missing_pages_artifact_preflight");
assert(appBackendHandoffIndex > pagesPreflightIndex, "deploy_must_validate_pages_before_backend_mutation");
assert(dryRunExitIndex > appBackendHandoffIndex, "deploy_dry_run_exit_missing_after_app_owned_preflight");
assert(pagesActivationIndex > dryRunExitIndex, "deploy_dry_run_must_exit_before_pages_activation");
assert(
  deploySource.includes("partial deploy flags are not supported by the canonical lifecycle"),
  "canonical_deploy_must_reject_partial_flags",
);
assert(
  deploySource.includes('[[ "${tag_sha}" == "${head_sha}" ]]'),
  "deploy_must_require_release_tag_at_head",
);
assert(
  deploySource.includes('[[ "${#head_release_tags[@]}" -eq 1 ]]'),
  "deploy_must_require_exactly_one_release_tag_at_head",
);
assert(
  deploySource.includes('assert_clean_default_branch "${repo_root}" LoopAware'),
  "deploy_must_require_the_clean_loopaware_default_branch_checkout",
);
assert(
  deploySource.includes('assert_canonical_github_origin "${repo_root}" LoopAware "tyemirov/loopaware"'),
  "deploy_must_verify_the_canonical_loopaware_repository_identity",
);
assert(!deploySource.includes("    --tag)"), "deploy_manual_tag_selection_forbidden");
assert(deploySource.includes("DEPLOY_TAG is not supported"), "deploy_tag_environment_override_must_fail_closed");
assert(
  deployResourcesSource.includes("type: ansible_task_bundle") &&
    deployResourcesSource.includes("validate: .mprlab/deploy/ansible/tasks/validate.yml") &&
    deployResourcesSource.includes("preflight: .mprlab/deploy/ansible/tasks/preflight.yml") &&
    deployResourcesSource.includes("deploy: .mprlab/deploy/ansible/tasks/deploy.yml") &&
    deployResourcesSource.includes("verify: .mprlab/deploy/ansible/tasks/verify.yml") &&
    appAnsibleRunnerSource.includes("ansible-core==2.19.8") &&
    appAnsibleRunnerSource.includes('ansible-playbook --inventory localhost, "${repo_root}/.mprlab/deploy/ansible/playbooks/preflight-local.yml"') &&
    appAnsibleRunnerSource.includes('ansible-playbook "${become_flags[@]}" --inventory "${inventory_path}" "${repo_root}/.mprlab/deploy/ansible/playbooks/deploy.yml"'),
  "deploy_missing_app_owned_ansible_task_bundle",
);
assert(
  !makefileSource.includes("GATEWAY_DIR") &&
    !fs.existsSync(path.join(repoRoot, "deploy")) &&
    !deploySource.includes("mprlab-gateway") &&
    !deploySource.includes("GATEWAY_DIR") &&
    !appAnsibleRunnerSource.includes("mprlab-gateway") &&
    appDeployComposeSource.includes("${LOOPAWARE_DEPLOY_IMAGE_REF:?") &&
    appDeployPlaybookSource.includes("Run the app-owned preflight entrypoint") &&
    appDeployPlaybookSource.includes("Run the app-owned deploy entrypoint") &&
    appDeployPlaybookSource.includes("Run the app-owned verify entrypoint") &&
    appDeployValidateSource.includes("go\n      - run\n      - ./cmd/configaudit") &&
    appDeployPreflightSource.includes("Verify LoopAware, TAuth, and Pinguin identities and authenticated canaries") &&
    appDeployTaskSource.includes("Activate the exact LoopAware backend release") &&
    appDeployVerifySource.includes("Require the exact running LoopAware backend"),
  "deploy_must_be_owned_and_executable_from_the_loopaware_checkout",
);
assert(
  deploySource.includes("verify_published_image_provenance") &&
    deploySource.includes("verify_release_container_descriptor") &&
    deploySource.includes('exact_image_ref="${IMAGE_REPOSITORY}@${release_digest}"') &&
    deploySource.includes("published image must contain exactly one manifest") &&
    deploySource.includes("does not match prepared descriptor"),
  "deploy_missing_immutable_image_provenance_contract",
);
assert(
  releaseHelperSource.includes('container_descriptor = "payloads/containers/loopaware/container.json"') &&
    deploySource.includes("--pattern container.json") &&
    deploySource.includes('"${release_artifact_directory}/container.json"'),
  "github_release_must_publish_and_deploy_the_container_descriptor",
);
assert(
  !deploySource.includes('make --no-print-directory pages-deploy'),
  "deploy_must_call_owned_pages_deployer_without_make_variable_override",
);
assert(pagesDeploySource.includes("--verify-only"), "pages_deployer_missing_verify_only_option");
assert(
  pagesDeploySource.includes("--artifact-dir <path>") &&
    pagesDeploySource.includes('gh release download "${version}" --repo tyemirov/loopaware') &&
    pagesDeploySource.includes("gh api repos/tyemirov/loopaware/pages") &&
    deploySource.match(/--artifact-dir "\$\{release_artifact_directory\}"/g)?.length === 2 &&
    deploySource.includes('gh release download "${TAG}" --repo tyemirov/loopaware') &&
    deploySource.includes("gh repo view tyemirov/loopaware"),
  "deploy_must_download_release_assets_once_and_pin_explicit_github_repositories",
);
assert(
  publishOrchestratorSource.includes("publication_attestation_path") &&
    publishOrchestratorSource.includes("no provider upload was repeated") &&
    publicationAttestationSource.includes("mprlab.loopaware-publication.v1") &&
    publicationAttestationSource.includes("app-store-connect-upload-accepted") &&
    publicationAttestationSource.includes("release_manifest_sha256") &&
    publicationAttestationSource.includes("gh release upload") &&
    releaseHelperSource.includes('publication_attestation = artifact_path / "publication.json"') &&
    deploySource.includes("verify_publication_attestation") &&
    deploySource.includes("--pattern publication.json"),
  "deploy_must_require_a_complete_publication_attestation",
);
assert(
  pagesDeploySource.indexOf('if [[ "${verify_only}" == "true" ]]') < pagesDeploySource.indexOf('git clone --no-checkout'),
  "pages_verify_only_must_exit_before_clone_or_push",
);
assert(
  !pagesDeploySource.includes("readarray -t release_values < <("),
  "pages_manifest_read_must_not_use_hanging_process_substitution",
);
assert(
  pagesDeploySource.includes('expected_source_commit="$(git rev-parse "${version}^{commit}^")"') &&
    pagesDeploySource.includes('git archive "${source_commit}:web"') &&
    pagesDeploySource.includes('diff -r "${expected_site_directory}" "${site_directory}"'),
  "pages_verify_only_missing_exact_source_content_contract",
);
assert(makefileSource.includes("release-dry-run:"), "release_dry_run_missing");
assert(
    makefileSource.includes("publish-dry-run:") &&
    makeRecipe("publish-dry-run").trim() ===
      "@/bin/sh ./scripts/release/run_lifecycle.sh ./scripts/release/with_lifecycle_lock.sh publish-dry-run ./scripts/publish-preflight.sh",
  "publish_dry_run_missing",
);
assert(
  releasePreflightSource.includes("Building disposable release artifacts") &&
    releasePreflightSource.includes("make --no-print-directory") &&
    releasePreflightSource.includes('"${artifact_target_list[@]}"'),
  "release_dry_run_must_build_disposable_artifacts",
);
assert(
  publishPreflightSource.indexOf("./scripts/publish-release.sh --dry-run") <
    publishPreflightSource.indexOf("publish_container_artifacts.sh --preflight-only") &&
    publishPreflightSource.indexOf("publish_container_artifacts.sh --preflight-only") <
      publishPreflightSource.indexOf("./scripts/publish-mobile.sh --preflight-only") &&
    publishPreflightSource.indexOf("./scripts/publish-mobile.sh --preflight-only") <
      publishPreflightSource.indexOf("./scripts/publish-react-native.sh --preflight-only"),
  "publish_preflight_stage_order_invalid",
);
assert(
  publishPreflightSource.includes("expected_manifest_sha256") &&
    publishPreflightSource.includes("assert_manifest_unchanged") &&
    publishPreflightSource.includes("prepared release manifest changed during publication preflight"),
  "publish_preflight_must_pin_one_manifest_between_provider_checks",
);
assert(
  publishPreflightSource.includes("gh repo view tyemirov/loopaware") &&
    deploySource.includes("gh repo view tyemirov/loopaware"),
  "github_permission_checks_must_name_the_canonical_repository",
);
assert(
  publishMobileSource.indexOf("preflight_mobile_publication") < publishMobileSource.indexOf('echo "==> [publish] Uploading'),
  "mobile_publish_must_preflight_both_stores_before_upload",
);
assert(
  publishMobileSource.includes('node "${ios_submit_script}"') &&
    publishMobileSource.includes('node "${android_publish_script}"') &&
    !publishMobileSource.includes("make --no-print-directory submit-ios") &&
    !publishMobileSource.includes("make --no-print-directory submit-android"),
  "mobile_publication_must_invoke_fixed_scripts_without_make_recipe_interpolation",
);
assert(
  androidPublishSource.includes("verifyAndroidPublisherAccess") &&
    androidPublishSource.includes("track update authority") &&
    androidPublishSource.includes("inspect existing Android Publisher bundles") &&
    androidPublishSource.includes("assertPublishableVersionCode") &&
    androidPublishSource.includes("validatedTrackState") &&
    androidPublishSource.includes("existingTrackState.releases") &&
    androidPublishSource.includes("changed metadata for an existing release") &&
    androidPublishSource.includes("assertBundleSha256") &&
    androidPublishSource.includes('method: "PUT"') &&
    androidPublishSource.includes("verifyCommittedAndroidPublication") &&
    androidPublishSource.includes("create Android Publisher verification edit") &&
    androidPublishSource.includes("delete Android Publisher verification edit") &&
    androidPublishSource.includes("committed Android track is missing retained versionCode") &&
    androidPublishSource.includes("failed edit cleanup"),
  "android_publication_must_preflight_inventory_preserve_track_and_postverify_commit",
);
assert(
  androidPublishSource.includes('changesInReviewBehavior: "ERROR_IF_IN_REVIEW"') &&
    androidPublishSource.includes("unknown option: --") &&
    !androidPublishSource.includes('options.get("track")') &&
    !androidPublishSource.includes('options.get("quota-project")'),
  "android_publication_must_reject_unknown_destinations_and_pending_review_mutation",
);
assert(
  iosSubmitSource.includes('packageCommand("--validate-app"') &&
    iosSubmitSource.includes('packageCommand("--upload-package"') &&
    !iosSubmitSource.includes("--preflight-only") &&
    !iosSubmitSource.includes("--list-providers"),
  "ios_publish_preflight_must_validate_exact_app_before_upload",
);
assert(containerPublishSource.includes("--preflight-only"), "container_publish_preflight_missing");
assert(
  containerPublishSource.split("docker login ghcr.io").length - 1 === 1 &&
    containerPublishSource.includes('--username "${registry_username}" --password-stdin') &&
    containerPublishSource.split('docker push "${platform_ref}"').length - 1 === 1 &&
    !containerPublishSource.includes("verify_ghcr_push_access") &&
    !containerPublishSource.includes("WWW-Authenticate") &&
    !containerPublishSource.includes("blobs/uploads/") &&
    !containerPublishSource.includes("ghcr.io/token") &&
    !containerPublishSource.includes("Authorization: Bearer") &&
    !containerPublishSource.includes("curl "),
  "container_publication_must_use_standard_docker_login_and_push",
);
assert(
  containerPublishSource.includes("verify_prepared_container_archive") &&
    containerPublishSource.includes("verify_container_archive_loadability") &&
    containerPublishSource.includes("loaded_container_image_id") &&
    containerPublishSource.includes("docker save --output") &&
    containerPublishSource.includes("prepared container image id") &&
    containerPublishSource.includes("https://github.com/tyemirov/loopaware"),
  "container_publish_must_verify_archive_identity_and_labels_before_push",
);
assert(
  containerPrepareSource.includes("container_archive_image_id.py") &&
    containerPublishSource.includes("container_archive_image_id.py") &&
    !containerPrepareSource.includes("{{.Id}}"),
  "container_descriptor_identity_must_use_saved_archive_config_digest",
);
assert(
  containerPublishSource.includes('sources+=("${image}@${push_platform_digest}")') &&
    containerPublishSource.includes("published_linux_amd64_digest") &&
    containerPublishSource.includes("does not match pushed digest"),
  "container_publish_must_pin_and_verify_the_registry_push_digest",
);
assert(
  containerPrepareSource.includes("{{.Os}}/{{.Architecture}}") &&
    containerPublishSource.includes("remote_version_platform_digest") &&
    containerPublishSource.includes("existing version index must contain exactly one manifest") &&
    containerPublishSource.includes("remote_single_platform_index_digest") &&
    containerPublishSource.includes("attestation-manifest") &&
    containerPublishSource.includes("existing mutable index must contain exactly one deployable manifest") &&
    containerPublishSource.includes("published image must contain exactly one manifest") &&
    containerPublishSource.includes("Preserving immutable existing") &&
    containerPublishSource.includes("loaded image platform") &&
    stagedArtifactVerifierSource.includes("container archive image config is not linux/amd64"),
  "container_version_and_platform_publication_must_be_immutable",
);
assert(
  pagesDeploySource.includes("PAGES_VERIFY_ATTEMPTS must be an integer from 1 through 120") &&
    pagesDeploySource.includes("PAGES_VERIFY_DELAY_SECONDS must be an integer from 0 through 300"),
  "pages_verification_arithmetic_inputs_must_be_bounded_decimals",
);
assert(
  containerPrepareSource.includes("assert_local_docker_endpoint") &&
    containerPublishSource.includes("assert_local_docker_endpoint") &&
    dockerIdentitySource.includes("DOCKER_HOST DOCKER_CONTEXT DOCKER_TLS_VERIFY DOCKER_CERT_PATH") &&
    dockerIdentitySource.includes("docker context inspect") &&
    dockerIdentitySource.includes("unix://*|npipe://*") &&
    dockerIdentitySource.includes("canonical lifecycle requires a local Docker endpoint"),
  "container_lifecycle_must_reject_remote_docker_endpoints",
);
assert(
  integrationRunnerSource.includes("rejects inherited ${variable_name}") &&
    integrationRunnerSource.includes("docker context inspect") &&
    integrationRunnerSource.includes("requires a local Docker context") &&
    integrationRunnerSource.includes('export LOOPAWARE_BASE_URL=http://localhost:8090') &&
    integrationRunnerSource.includes('export LOOPAWARE_ENV_FILE=${test_config_dir}/loopaware.env'),
  "integration_runner_must_be_isolated_from_production_ambient_state",
);
assert(
  releaseHelperSource.includes("existing GitHub Release metadata is immutable") &&
    releaseHelperSource.includes("existing GitHub Release asset is immutable") &&
    releaseHelperSource.includes("release_asset_plan") &&
    !releaseHelperSource.includes('"--clobber"'),
  "github_release_metadata_and_assets_must_be_immutable",
);
assert(
  publishReactNativeSource.includes("npm registry lookup failed") &&
    !publishReactNativeSource.includes('view "${package_spec}" dist.integrity --json 2>/dev/null || true'),
  "npm_publication_lookup_must_fail_closed",
);
assert(
  publishReactNativeSource.includes("must be bootstrapped once before the canonical lifecycle can prove write authority") &&
    publishReactNativeSource.includes("access set status=public") &&
    publishReactNativeSource.includes("verify_public_status") &&
    publishReactNativeSource.includes("refusing to move latest backward") &&
    publishReactNativeSource.includes("npm post-publication visibility verification") &&
    publishReactNativeSource.includes("dist-tag add"),
  "npm_preflight_must_prove_existing_package_write_authority_or_fail_closed",
);
assert(
  publishReactNativeSource.includes("prepared React Native package name is not @loopaware/react-native") &&
    publishReactNativeSource.includes('canonical_registry="https://registry.npmjs.org/"') &&
    publishReactNativeSource.includes("React Native publication arguments are not part of the canonical lifecycle contract") &&
    publishReactNativeSource.includes("--dry-run=false") &&
    publishReactNativeSource.includes("--tag latest"),
  "react_native_publication_destination_must_be_canonical",
);
assert(
  publishMobileSource.includes("verify-release-artifact"),
  "mobile_publication_must_reverify_the_outer_release_manifest",
);
assert(
  publishMobileSource.includes("prepared mobile artifacts do not share one versioning identity") &&
    publishMobileSource.includes("prepared mobile artifact timestamp does not match the outer release manifest") &&
    publishMobileSource.includes("do not blindly retry") &&
    iosSubmitSource.includes("does not match the publication release identity") &&
    androidPublishSource.includes("does not match the publication release identity"),
  "mobile_publication_must_pin_one_timestamp_and_report_partial_outcomes",
);
assert(
  !androidPublishSource.includes("process.env.MOBILE_ANDROID_PLAY_TRACK") &&
    !androidPublishSource.includes("process.env.MOBILE_ANDROID_PLAY_STATUS") &&
    !androidPublishSource.includes("process.env.GOOGLE_CLOUD_QUOTA_PROJECT") &&
    androidPublishSource.includes("delete failed Android Publisher release edit"),
  "android_publication_destination_and_cleanup_contract_missing",
);
for (const source of [pagesDeploySource, publishMobileSource, publishReactNativeSource, containerPublishSource]) {
  assert(!source.match(/(?:readarray|mapfile)[^\n]*< <\(/), "release_script_hanging_process_substitution_forbidden");
}

assert(normalizedReadmeSource.includes("`make release` prepares"), "readme_missing_local_release_contract");
assert(normalizedReadmeSource.includes("`make publish` publishes"), "readme_missing_publish_contract");
assert(normalizedReadmeSource.includes("`make deploy-dry-run`"), "readme_missing_deploy_dry_run_contract");

console.log("release workflow validation passed");

/**
 * @param {string} relativePath
 * @returns {string}
 */
function readText(relativePath) {
  return fs.readFileSync(path.join(repoRoot, relativePath), "utf8");
}

/**
 * @param {string} target
 * @returns {string}
 */
function makeRecipe(target) {
  const lines = makefileSource.split("\n");
  const targetIndex = lines.findIndex((line) => line.startsWith(`${target}:`));
  assert(targetIndex >= 0, `make_target_missing:${target}`);
  const recipeLines = [];
  for (let index = targetIndex + 1; index < lines.length && lines[index].startsWith("\t"); index += 1) {
    recipeLines.push(lines[index].slice(1));
  }
  assert(recipeLines.length > 0, `make_recipe_missing:${target}`);
  return recipeLines.join("\n");
}

/**
 * @param {string} jobName
 * @returns {string}
 */
function workflowJob(jobName) {
  const lines = ciWorkflowSource.split("\n");
  const jobIndex = lines.findIndex((line) => line === `  ${jobName}:`);
  assert(jobIndex >= 0, `workflow_job_missing:${jobName}`);
  const jobLines = [];
  for (let index = jobIndex; index < lines.length; index += 1) {
    if (index > jobIndex && /^  [a-z0-9-]+:$/.test(lines[index])) break;
    jobLines.push(lines[index]);
  }
  return jobLines.join("\n");
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
