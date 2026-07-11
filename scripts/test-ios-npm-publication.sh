#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
temporary_directory="$(mktemp -d)"
trap 'rm -rf "${temporary_directory}"' EXIT

fake_bin="${temporary_directory}/bin"
mkdir -p "${fake_bin}"

ios_directory="${temporary_directory}/ios"
ios_log="${ios_directory}/xcrun.log"
ios_ipa="${ios_directory}/loopaware-ios.ipa"
ios_manifest="${ios_directory}/loopaware-ios.json"
ios_api_key="${ios_directory}/AuthKey_TESTKEY.p8"
mkdir -p "${ios_directory}"
printf 'fixture signed IPA payload\n' >"${ios_ipa}"
printf 'fixture App Store Connect API key\n' >"${ios_api_key}"

python3 - "${ios_manifest}" "${ios_ipa}" <<'PY_IOS_MANIFEST'
import hashlib
import json
import pathlib
import sys

manifest_path = pathlib.Path(sys.argv[1])
ipa_path = pathlib.Path(sys.argv[2]).resolve()
manifest_path.write_text(
    json.dumps(
        {
            "schema": "loopaware.mobile-ios-archive.v1",
            "status": "passed",
            "app": {
                "bundleIdentifier": "com.mprlab.loopaware",
                "version": "2026.7.11",
                "buildNumber": "205891200",
            },
            "ipa": {
                "path": str(ipa_path),
                "sha256": hashlib.sha256(ipa_path.read_bytes()).hexdigest(),
                "sizeBytes": ipa_path.stat().st_size,
            },
            "versioning": {
                "releaseTimestamp": "2026-07-11T00:00:00.000Z",
                "releaseVersion": "2026.7.11",
                "buildCode": 205891200,
                "iosBuildNumber": "205891200",
                "androidVersionCode": 205891200,
                "buildCodeSource": "calver_utc_seconds_since_2020_01_01",
            },
        }
    ),
    encoding="utf-8",
)
PY_IOS_MANIFEST

cat >"${fake_bin}/xcrun" <<'EOF_XCRUN'
#!/usr/bin/env bash
set -euo pipefail
printf '%s|API_PRIVATE_KEYS_DIR=%s\n' "$*" "${API_PRIVATE_KEYS_DIR:-<unset>}" >>"${IOS_LOG}"
case "$*" in
  "altool --list-providers --output-format json --api-key TESTKEY --api-issuer fixture-issuer")
    printf '%s\n' '{"providers":[]}'
    ;;
  "altool --validate-app "*)
    if [[ "${FAKE_IOS_VALIDATION_FAILURE:-0}" == "1" ]]; then
      printf '%s\n' 'fixture altool validation failed' >&2
      exit 42
    fi
    ;;
  "altool --upload-package "*)
    if [[ "${FAKE_IOS_UPLOAD_FAILURE:-0}" == "1" ]]; then
      printf '%s\n' 'fixture altool upload failed' >&2
      exit 43
    fi
    printf '%s\n' 'fixture altool upload accepted'
    ;;
  *)
    printf 'unexpected xcrun command: %s\n' "$*" >&2
    exit 97
    ;;
esac
EOF_XCRUN
chmod +x "${fake_bin}/xcrun"

ios_common_args=(
  --mobile-dir "${repo_root}/mobile"
  --release-timestamp "2026-07-11T00:00:00Z"
  --asc-api-key-id TESTKEY
  --asc-api-issuer-id fixture-issuer
  --asc-api-key-path "${ios_api_key}"
  --asc-app-id 6788555440
)

PATH="${fake_bin}:${PATH}" IOS_LOG="${ios_log}" \
  node "${repo_root}/mobile/scripts/submit-ios.mjs" \
    "${ios_common_args[@]}" \
    --preflight-only >"${ios_directory}/credentials.json"
grep -Fq '"credentialAccess": "verified"' "${ios_directory}/credentials.json"
expected_provider_command="altool --list-providers --output-format json --api-key TESTKEY --api-issuer fixture-issuer|API_PRIVATE_KEYS_DIR=${ios_directory}"
[[ "$(<"${ios_log}")" == "${expected_provider_command}" ]]

: >"${ios_log}"
PATH="${fake_bin}:${PATH}" IOS_LOG="${ios_log}" \
  node "${repo_root}/mobile/scripts/submit-ios.mjs" \
    "${ios_common_args[@]}" \
    --manifest "${ios_manifest}" \
    --ipa "${ios_ipa}" \
    --provider-public-id fixture-provider \
    --dry-run >"${ios_directory}/validation.json"
grep -Fq '"appValidation": "passed"' "${ios_directory}/validation.json"
expected_validation_command="altool --validate-app ${ios_ipa} --platform ios --apple-id 6788555440 --bundle-id com.mprlab.loopaware --bundle-version 205891200 --bundle-short-version-string 2026.7.11 --api-key TESTKEY --api-issuer fixture-issuer --provider-public-id fixture-provider|API_PRIVATE_KEYS_DIR=${ios_directory}"
[[ "$(<"${ios_log}")" == "${expected_validation_command}" ]]
[[ "$(<"${ios_log}")" != *"--upload-package"* ]]

: >"${ios_log}"
PATH="${fake_bin}:${PATH}" IOS_LOG="${ios_log}" \
  node "${repo_root}/mobile/scripts/submit-ios.mjs" \
    "${ios_common_args[@]}" \
    --manifest "${ios_manifest}" \
    --ipa "${ios_ipa}" \
    --provider-public-id fixture-provider >"${ios_directory}/upload.json"
grep -Fq 'fixture altool upload accepted' "${ios_directory}/upload.json"
grep -Fq '"status": "submitted"' "${ios_directory}/upload.json"
grep -Fq '"tool": "xcrun altool"' "${ios_directory}/upload.json"
expected_upload_command="altool --upload-package ${ios_ipa} --platform ios --apple-id 6788555440 --bundle-id com.mprlab.loopaware --bundle-version 205891200 --bundle-short-version-string 2026.7.11 --api-key TESTKEY --api-issuer fixture-issuer --provider-public-id fixture-provider|API_PRIVATE_KEYS_DIR=${ios_directory}"
[[ "$(<"${ios_log}")" == "${expected_upload_command}" ]]

: >"${ios_log}"
set +e
ios_upload_failure_output="$({
  PATH="${fake_bin}:${PATH}" IOS_LOG="${ios_log}" FAKE_IOS_UPLOAD_FAILURE=1 \
    node "${repo_root}/mobile/scripts/submit-ios.mjs" \
      "${ios_common_args[@]}" \
      --manifest "${ios_manifest}" \
      --ipa "${ios_ipa}" \
      --provider-public-id fixture-provider
} 2>&1)"
ios_upload_failure_status=$?
set -e
[[ "${ios_upload_failure_status}" -eq 2 ]]
[[ "${ios_upload_failure_output}" == *"fixture altool upload failed"* ]]
[[ "${ios_upload_failure_output}" == *"command failed with exit 43"* ]]
[[ "$(<"${ios_log}")" == "${expected_upload_command}" ]]

: >"${ios_log}"
set +e
ios_failure_output="$({
  PATH="${fake_bin}:${PATH}" IOS_LOG="${ios_log}" FAKE_IOS_VALIDATION_FAILURE=1 \
    node "${repo_root}/mobile/scripts/submit-ios.mjs" \
      "${ios_common_args[@]}" \
      --manifest "${ios_manifest}" \
      --ipa "${ios_ipa}" \
      --provider-public-id fixture-provider \
      --dry-run
} 2>&1)"
ios_failure_status=$?
set -e
[[ "${ios_failure_status}" -eq 2 ]]
[[ "${ios_failure_output}" == *"fixture altool validation failed"* ]]
[[ "$(<"${ios_log}")" == "${expected_validation_command}" ]]

: >"${ios_log}"
set +e
ios_legacy_output="$({
  PATH="${fake_bin}:${PATH}" IOS_LOG="${ios_log}" \
    node "${repo_root}/mobile/scripts/submit-ios.mjs" \
      "${ios_common_args[@]}" \
      --apple-id legacy@example.com \
      --preflight-only
} 2>&1)"
ios_legacy_status=$?
set -e
[[ "${ios_legacy_status}" -eq 2 ]]
[[ "${ios_legacy_output}" == *"unknown option: --apple-id"* ]]
[[ ! -s "${ios_log}" ]]

npm_repository="${temporary_directory}/npm-repository"
npm_log="${npm_repository}/npm.log"
npm_artifact_directory="${npm_repository}/.git/mprlab-release"
npm_payload_directory="${npm_artifact_directory}/payloads/release-assets"
npm_package_directory="${temporary_directory}/npm-package/package"
npm_tarball="${npm_payload_directory}/loopaware-react-native-0.1.0.tgz"
npm_tarball_argument=".git/mprlab-release/payloads/release-assets/loopaware-react-native-0.1.0.tgz"
npm_published_state="${npm_repository}/published-integrity.state"
npm_latest_state="${npm_repository}/latest.state"
mkdir -p "${npm_repository}/scripts/release" "${npm_repository}/configs" "${npm_payload_directory}" "${npm_package_directory}"
git -C "${npm_repository}" init -b master >/dev/null
cp "${repo_root}/scripts/publish-react-native.sh" "${npm_repository}/scripts/publish-react-native.sh"
cp "${repo_root}/scripts/release/load_release_env.sh" "${npm_repository}/scripts/release/load_release_env.sh"
cp "${repo_root}/scripts/release/parse_release_env.py" "${npm_repository}/scripts/release/parse_release_env.py"
cat >"${npm_repository}/scripts/release/release_helper.py" <<'EOF_NPM_HELPER'
#!/usr/bin/env bash
set -euo pipefail
[[ "$*" == "verify-release-artifact" ]]
EOF_NPM_HELPER
chmod +x "${npm_repository}/scripts/publish-react-native.sh" "${npm_repository}/scripts/release/release_helper.py" "${npm_repository}/scripts/release/parse_release_env.py"
: >"${npm_repository}/configs/.env.loopaware"
cat >"${npm_package_directory}/package.json" <<'EOF_PACKAGE_JSON'
{
  "name": "@loopaware/react-native",
  "version": "0.1.0",
  "publishConfig": {
    "access": "public",
    "registry": "https://registry.npmjs.org/"
  }
}
EOF_PACKAGE_JSON
COPYFILE_DISABLE=1 tar -czf "${npm_tarball}" -C "${temporary_directory}/npm-package" package
npm_tarball_sha256="$(shasum -a 256 "${npm_tarball}" | awk '{print $1}')"
npm_prepared_integrity="$(node - "${npm_tarball}" <<'NODE_INTEGRITY'
const fs = require("node:fs");
const crypto = require("node:crypto");
const archive = fs.readFileSync(process.argv[2]);
process.stdout.write(`sha512-${crypto.createHash("sha512").update(archive).digest("base64")}`);
NODE_INTEGRITY
)"
python3 - "${npm_artifact_directory}/manifest.json" "${npm_tarball_sha256}" <<'PY_NPM_MANIFEST'
import json
import pathlib
import sys

pathlib.Path(sys.argv[1]).write_text(
    json.dumps(
        {
            "payloads": [
                {
                    "path": "payloads/release-assets/loopaware-react-native-0.1.0.tgz",
                    "sha256": sys.argv[2],
                }
            ]
        }
    ),
    encoding="utf-8",
)
PY_NPM_MANIFEST

cat >"${fake_bin}/npm" <<'EOF_NPM'
#!/usr/bin/env bash
set -euo pipefail
printf '%s|NPM_CONFIG_DRY_RUN=%s|npm_config_dry_run=%s\n' \
  "$*" "${NPM_CONFIG_DRY_RUN:-<unset>}" "${npm_config_dry_run:-<unset>}" >>"${NPM_LOG}"
case "$*" in
  "whoami --registry https://registry.npmjs.org/")
    printf '%s\n' 'fixture-user'
    ;;
  "access list packages fixture-user --json --registry https://registry.npmjs.org/")
    printf '{"@loopaware/react-native":"%s"}\n' "${FAKE_NPM_PERMISSION:-read-write}"
    ;;
  "access get status @loopaware/react-native --json --registry https://registry.npmjs.org/")
    printf '%s\n' '{"@loopaware/react-native":"public"}'
    ;;
  "access set status=public @loopaware/react-native --json --registry https://registry.npmjs.org/")
    if [[ "${FAKE_NPM_WRITE_FAILURE:-0}" == "1" ]]; then
      printf '%s\n' 'npm error code E403 fixture npm write denial' >&2
      exit 1
    fi
    printf '%s\n' '{"@loopaware/react-native":"public"}'
    ;;
  "view @loopaware/react-native@0.1.0 dist.integrity --json --registry https://registry.npmjs.org/")
    if [[ -f "${NPM_PUBLISHED_STATE}" ]]; then
      printf '"%s"\n' "$(<"${NPM_PUBLISHED_STATE}")"
      exit 0
    fi
    case "${FAKE_NPM_VIEW_MODE:-missing}" in
      missing)
        printf '%s\n' 'npm error code E404' >&2
        exit 1
        ;;
      error)
        printf '%s\n' 'npm error code E500 fixture registry failure' >&2
        exit 1
        ;;
      published)
        printf '"%s"\n' "${FAKE_NPM_PUBLISHED_INTEGRITY:?}"
        ;;
      *)
        printf 'unexpected fake npm view mode: %s\n' "${FAKE_NPM_VIEW_MODE}" >&2
        exit 97
        ;;
    esac
    ;;
  "view @loopaware/react-native name --json --registry https://registry.npmjs.org/")
    if [[ "${FAKE_NPM_PACKAGE_MODE:-exists}" == "missing" ]]; then
      printf '%s\n' 'npm error code E404' >&2
      exit 1
    fi
    printf '%s\n' '"@loopaware/react-native"'
    ;;
  "view @loopaware/react-native dist-tags --json --registry https://registry.npmjs.org/")
    if [[ -f "${NPM_LATEST_STATE}" ]]; then
      latest="$(<"${NPM_LATEST_STATE}")"
    else
      latest="${FAKE_NPM_LATEST_VERSION:-0.0.9}"
    fi
    printf '{"latest":"%s"}\n' "${latest}"
    ;;
  "dist-tag add @loopaware/react-native@0.1.0 latest --registry https://registry.npmjs.org/")
    printf '%s\n' '0.1.0' >"${NPM_LATEST_STATE}"
    ;;
  "publish "*)
    [[ -z "${NPM_CONFIG_DRY_RUN:-}" && -z "${npm_config_dry_run:-}" ]] || {
      printf '%s\n' 'npm dry-run environment leaked into canonical publication command' >&2
      exit 96
    }
    if [[ "$*" == *"--dry-run=false"* ]]; then
      printf '%s\n' "${FAKE_NPM_PREPARED_INTEGRITY}" >"${NPM_PUBLISHED_STATE}"
      printf '%s\n' '0.1.0' >"${NPM_LATEST_STATE}"
    fi
    ;;
  *)
    printf 'unexpected npm command: %s\n' "$*" >&2
    exit 97
    ;;
esac
EOF_NPM
chmod +x "${fake_bin}/npm"

run_npm_publication() {
  local mode="$1"
  (
    cd "${npm_repository}"
    PATH="${fake_bin}:${PATH}" \
      NPM_LOG="${npm_log}" \
      NPM_PUBLISHED_STATE="${npm_published_state}" \
      NPM_LATEST_STATE="${npm_latest_state}" \
      FAKE_NPM_PREPARED_INTEGRITY="${npm_prepared_integrity}" \
      RELEASE_ENV_FILE="${npm_repository}/configs/.env.loopaware" \
      NPM_CONFIG_DRY_RUN=true \
      npm_config_dry_run=true \
      env -u NPM_API_KEY -u NODE_AUTH_TOKEN ./scripts/publish-react-native.sh ${mode}
  )
}

: >"${npm_log}"
set +e
npm_permission_output="$(
  FAKE_NPM_PERMISSION=read-only run_npm_publication --preflight-only 2>&1
)"
npm_permission_status=$?
set -e
[[ "${npm_permission_status}" -ne 0 ]]
[[ "${npm_permission_output}" == *"does not have read-write access"* ]]
[[ "$(wc -l <"${npm_log}" | tr -d ' ')" == "4" ]]

: >"${npm_log}"
set +e
npm_bootstrap_output="$(
  FAKE_NPM_VIEW_MODE=missing \
  FAKE_NPM_PACKAGE_MODE=missing \
    run_npm_publication --preflight-only 2>&1
)"
npm_bootstrap_status=$?
set -e
[[ "${npm_bootstrap_status}" -ne 0 ]]
[[ "${npm_bootstrap_output}" == *"must be bootstrapped once before the canonical lifecycle can prove write authority"* ]]
[[ "$(wc -l <"${npm_log}" | tr -d ' ')" == "2" ]]

: >"${npm_log}"
npm_preflight_output="$(
  FAKE_NPM_VIEW_MODE=missing run_npm_publication --preflight-only
)"
[[ "${npm_preflight_output}" == *"existing package remained public and no npm version was published"* ]]
[[ "$(wc -l <"${npm_log}" | tr -d ' ')" == "9" ]]
npm_preflight_publish_command="$(tail -n 1 "${npm_log}")"
[[ "${npm_preflight_publish_command}" == "publish ${npm_tarball_argument} --dry-run --registry https://registry.npmjs.org/ --access public --tag latest|NPM_CONFIG_DRY_RUN=<unset>|npm_config_dry_run=<unset>" ]]

: >"${npm_log}"
set +e
npm_write_output="$({
  FAKE_NPM_VIEW_MODE=missing \
  FAKE_NPM_WRITE_FAILURE=1 \
    run_npm_publication --preflight-only
} 2>&1)"
npm_write_status=$?
set -e
[[ "${npm_write_status}" -ne 0 ]]
[[ "${npm_write_output}" == *"fixture npm write denial"* ]]
[[ "$(wc -l <"${npm_log}" | tr -d ' ')" == "7" ]]
[[ "$(<"${npm_log}")" != *"publish ${npm_tarball_argument}"* ]]

: >"${npm_log}"
set +e
npm_lookup_error_output="$(
  FAKE_NPM_VIEW_MODE=error run_npm_publication --preflight-only 2>&1
)"
npm_lookup_error_status=$?
set -e
[[ "${npm_lookup_error_status}" -ne 0 ]]
[[ "${npm_lookup_error_output}" == *"npm registry lookup failed for @loopaware/react-native@0.1.0"* ]]
[[ "${npm_lookup_error_output}" == *"E500 fixture registry failure"* ]]
[[ "$(wc -l <"${npm_log}" | tr -d ' ')" == "1" ]]

: >"${npm_log}"
npm_existing_output="$(
  FAKE_NPM_VIEW_MODE=published \
  FAKE_NPM_PUBLISHED_INTEGRITY="${npm_prepared_integrity}" \
  FAKE_NPM_LATEST_VERSION=0.1.0 \
    run_npm_publication --preflight-only
)"
[[ "${npm_existing_output}" == *"has the prepared integrity"* ]]
[[ "${npm_existing_output}" == *"latest already points to 0.1.0"* ]]
[[ "$(wc -l <"${npm_log}" | tr -d ' ')" == "8" ]]

: >"${npm_log}"
set +e
npm_downgrade_output="$(
  FAKE_NPM_VIEW_MODE=published \
  FAKE_NPM_PUBLISHED_INTEGRITY="${npm_prepared_integrity}" \
  FAKE_NPM_LATEST_VERSION=0.2.0 \
    run_npm_publication --preflight-only 2>&1
)"
npm_downgrade_status=$?
set -e
[[ "${npm_downgrade_status}" -ne 0 ]]
[[ "${npm_downgrade_output}" == *"refusing to move latest backward"* ]]
[[ "$(wc -l <"${npm_log}" | tr -d ' ')" == "5" ]]
[[ "$(<"${npm_log}")" != *"access set status=public"* ]]

: >"${npm_log}"
set +e
npm_mismatch_output="$(
  FAKE_NPM_VIEW_MODE=published \
  FAKE_NPM_PUBLISHED_INTEGRITY="sha512-fixture-mismatch" \
    run_npm_publication --preflight-only 2>&1
)"
npm_mismatch_status=$?
set -e
[[ "${npm_mismatch_status}" -ne 0 ]]
[[ "${npm_mismatch_output}" == *"already exists with different content"* ]]
[[ "$(wc -l <"${npm_log}" | tr -d ' ')" == "1" ]]

: >"${npm_log}"
rm -f "${npm_published_state}" "${npm_latest_state}"
npm_publish_output="$(
  FAKE_NPM_VIEW_MODE=missing run_npm_publication ""
)"
[[ "${npm_publish_output}" == *"Publishing prepared @loopaware/react-native@0.1.0"* ]]
[[ "${npm_publish_output}" == *"Published @loopaware/react-native@0.1.0 with matching integrity, public visibility, and latest dist-tag."* ]]
[[ "$(wc -l <"${npm_log}" | tr -d ' ')" == "12" ]]
npm_actual_publish_command="$(grep '^publish ' "${npm_log}")"
[[ "${npm_actual_publish_command}" == "publish ${npm_tarball_argument} --dry-run=false --registry https://registry.npmjs.org/ --access public --tag latest|NPM_CONFIG_DRY_RUN=<unset>|npm_config_dry_run=<unset>" ]]

: >"${npm_log}"
rm -f "${npm_published_state}" "${npm_latest_state}"
npm_retag_output="$(
  FAKE_NPM_VIEW_MODE=published \
  FAKE_NPM_PUBLISHED_INTEGRITY="${npm_prepared_integrity}" \
  FAKE_NPM_LATEST_VERSION=0.0.9 \
    run_npm_publication ""
)"
[[ "${npm_retag_output}" == *"Updating @loopaware/react-native@latest to 0.1.0"* ]]
grep -Fq 'dist-tag add @loopaware/react-native@0.1.0 latest --registry https://registry.npmjs.org/' "${npm_log}"
[[ "$(<"${npm_latest_state}")" == "0.1.0" ]]

echo "iOS and npm publication contract checks passed"
