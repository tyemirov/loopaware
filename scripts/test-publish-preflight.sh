#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
temporary_directory="$(mktemp -d)"
trap 'rm -rf "${temporary_directory}"' EXIT

fixture_repository="${temporary_directory}/loopaware"
command_log="${temporary_directory}/commands.log"
fake_bin="${temporary_directory}/bin"
mkdir -p "${fixture_repository}/scripts/release" "${fake_bin}"
cp "${repo_root}/scripts/publish-preflight.sh" "${fixture_repository}/scripts/publish-preflight.sh"
cp "${repo_root}/scripts/publish-mobile.sh" "${fixture_repository}/scripts/publish-mobile.sh"

for stage in publish-release publish-mobile publish-react-native; do
  cat >"${fixture_repository}/scripts/${stage}.sh" <<'EOF_STAGE'
#!/usr/bin/env bash
set -euo pipefail
stage="$(basename "$0" .sh)"
printf '%s|%s\n' "${stage}" "$*" >>"${COMMAND_LOG}"
[[ "${FAKE_FAIL_STAGE:-}" != "${stage}" ]] || exit 42
if [[ "${FAKE_MUTATE_STAGE:-}" == "${stage}" ]]; then
  printf ' ' >>"${PREFLIGHT_MANIFEST_PATH}"
fi
EOF_STAGE
  chmod +x "${fixture_repository}/scripts/${stage}.sh"
done
cat >"${fixture_repository}/scripts/release/publish_container_artifacts.sh" <<'EOF_CONTAINER'
#!/usr/bin/env bash
set -euo pipefail
printf 'container|%s\n' "$*" >>"${COMMAND_LOG}"
[[ "${FAKE_FAIL_STAGE:-}" != "container" ]] || exit 42
if [[ "${FAKE_MUTATE_STAGE:-}" == "container" ]]; then
  printf ' ' >>"${PREFLIGHT_MANIFEST_PATH}"
fi
EOF_CONTAINER
chmod +x "${fixture_repository}/scripts/release/publish_container_artifacts.sh"

git -C "${fixture_repository}" init -b master >/dev/null
mkdir -p "${fixture_repository}/.git/mprlab-release"
printf '%s\n' '{"schema_version":2,"artifact_kind":"mprlab.release","version":"v1.2.3"}' >"${fixture_repository}/.git/mprlab-release/manifest.json"
cat >"${fake_bin}/gh" <<'EOF_GH'
#!/usr/bin/env bash
set -euo pipefail
[[ "$*" == "repo view tyemirov/loopaware --json viewerPermission --jq .viewerPermission" ]] || { printf 'unexpected gh command: %s\n' "$*" >&2; exit 97; }
printf '%s\n' "${FAKE_GH_PERMISSION:-WRITE}"
EOF_GH
chmod +x "${fake_bin}/gh"

(
  cd "${fixture_repository}"
  PATH="${fake_bin}:${PATH}" COMMAND_LOG="${command_log}" ./scripts/publish-preflight.sh >/dev/null
)
expected_order=$'publish-release|--dry-run\ncontainer|--preflight-only\npublish-mobile|--preflight-only\npublish-react-native|--preflight-only'
[[ "$(<"${command_log}")" == "${expected_order}" ]]

preflight_manifest_path="${fixture_repository}/.git/mprlab-release/manifest.json"
cp "${preflight_manifest_path}" "${preflight_manifest_path}.original"
: >"${command_log}"
set +e
manifest_drift_output="$(
  cd "${fixture_repository}"
  PATH="${fake_bin}:${PATH}" COMMAND_LOG="${command_log}" PREFLIGHT_MANIFEST_PATH="${preflight_manifest_path}" FAKE_MUTATE_STAGE=container \
    ./scripts/publish-preflight.sh 2>&1
)"
manifest_drift_status=$?
set -e
mv "${preflight_manifest_path}.original" "${preflight_manifest_path}"
[[ "${manifest_drift_status}" -ne 0 ]]
[[ "${manifest_drift_output}" == *"prepared release manifest changed during publication preflight"* ]]
[[ "$(wc -l <"${command_log}" | tr -d ' ')" == "2" ]]

: >"${command_log}"
set +e
(
  cd "${fixture_repository}"
  PATH="${fake_bin}:${PATH}" COMMAND_LOG="${command_log}" FAKE_FAIL_STAGE="publish-mobile" \
    ./scripts/publish-preflight.sh >/dev/null 2>&1
)
failure_status=$?
set -e
[[ "${failure_status}" -ne 0 ]]
[[ "$(wc -l <"${command_log}" | tr -d ' ')" == "3" ]]
[[ "$(tail -n 1 "${command_log}")" == "publish-mobile|--preflight-only" ]]

: >"${command_log}"
set +e
(
  cd "${fixture_repository}"
  PATH="${fake_bin}:${PATH}" COMMAND_LOG="${command_log}" PUBLISH_RELEASE_ARGS="--dry-run" \
    ./scripts/publish-preflight.sh >/dev/null 2>&1
)
reserved_status=$?
set -e
[[ "${reserved_status}" -ne 0 ]]
[[ ! -s "${command_log}" ]]

: >"${command_log}"
set +e
(
  cd "${fixture_repository}"
  PATH="${fake_bin}:${PATH}" COMMAND_LOG="${command_log}" FAKE_GH_PERMISSION="READ" \
    ./scripts/publish-preflight.sh >/dev/null 2>&1
)
github_permission_status=$?
set -e
[[ "${github_permission_status}" -ne 0 ]]
[[ "$(wc -l <"${command_log}" | tr -d ' ')" == "1" ]]
[[ "$(<"${command_log}")" == "publish-release|--dry-run" ]]

mobile_repository="${temporary_directory}/mobile-fixture"
mobile_log="${temporary_directory}/mobile.log"
mkdir -p "${mobile_repository}/scripts/release" "${mobile_repository}/configs" "${mobile_repository}/bin"
cp "${repo_root}/scripts/publish-mobile.sh" "${mobile_repository}/scripts/publish-mobile.sh"
cp "${repo_root}/scripts/release/load_release_env.sh" "${mobile_repository}/scripts/release/load_release_env.sh"
cp "${repo_root}/scripts/release/parse_release_env.py" "${mobile_repository}/scripts/release/parse_release_env.py"
cat >"${mobile_repository}/scripts/release/release_helper.py" <<'EOF_MOBILE_HELPER'
#!/usr/bin/env bash
set -euo pipefail
printf 'release-helper|%s\n' "$*" >>"${MOBILE_LOG}"
[[ "$*" == "verify-release-artifact" ]]
EOF_MOBILE_HELPER
chmod +x "${mobile_repository}/scripts/release/release_helper.py"
chmod +x "${mobile_repository}/scripts/release/parse_release_env.py"
: >"${mobile_repository}/configs/.env.loopaware"
manifest_directory="${mobile_repository}/release-artifact"
payload_directory="${manifest_directory}/payloads/release-assets"
mkdir -p "${payload_directory}"
for artifact in loopaware-ios.ipa loopaware-ios.json loopaware-android.aab loopaware-android.json loopaware-android-mapping.txt; do
  printf 'fixture-%s\n' "${artifact}" >"${payload_directory}/${artifact}"
done
python3 - "${payload_directory}/loopaware-ios.json" "${payload_directory}/loopaware-android.json" <<'PY_MOBILE_VERSIONING'
import json
import pathlib
import sys

versioning = {
    "releaseTimestamp": "2026-07-11T07:00:00.000Z",
    "releaseVersion": "2026.7.11",
    "buildCode": 1,
    "iosBuildNumber": "1",
    "androidVersionCode": 1,
    "buildCodeSource": "fixture",
}
for raw_path in sys.argv[1:]:
    pathlib.Path(raw_path).write_text(json.dumps({"versioning": versioning}), encoding="utf-8")
PY_MOBILE_VERSIONING
python3 - "${manifest_directory}/manifest.json" "${payload_directory}" <<'PY'
import json
import pathlib
import sys

manifest_path = pathlib.Path(sys.argv[1])
payload_root = pathlib.Path(sys.argv[2])
manifest_path.write_text(
    json.dumps(
        {
            "release_timestamp": "2026-07-11T00:00:00-0700",
            "payloads": [
                {"path": str(path.relative_to(manifest_path.parent))}
                for path in sorted(payload_root.iterdir())
            ],
        }
    ),
    encoding="utf-8",
)
PY
cat >"${mobile_repository}/bin/make" <<'EOF_MAKE'
#!/usr/bin/env bash
set -euo pipefail
printf 'make|%s\n' "$*" >>"${MOBILE_LOG}"
[[ "$*" == "--no-print-directory mobile-check" ]]
EOF_MAKE
cat >"${mobile_repository}/bin/node" <<'EOF_NODE'
#!/usr/bin/env bash
set -euo pipefail
script="$1"
shift
script_name="$(basename "${script}")"
printf 'node|%s|%s|MOBILE_RELEASE_TIMESTAMP=%s\n' "${script_name}" "$*" "${MOBILE_RELEASE_TIMESTAMP:-<unset>}" >>"${MOBILE_LOG}"
case "${script_name}" in
  submit-ios.mjs)
    if [[ "${FAKE_MOBILE_IOS_UPLOAD_FAILURE:-0}" == "1" && "$*" != *"--dry-run"* ]]; then
      exit 41
    fi
    ;;
  publish-android-play.mjs)
    if [[ "${FAKE_MOBILE_ANDROID_UPLOAD_FAILURE:-0}" == "1" && "$*" != *"--dry-run"* ]]; then
      exit 42
    fi
    ;;
  *)
    printf 'unexpected node script: %s\n' "${script}" >&2
    exit 97
    ;;
esac
EOF_NODE
cat >"${mobile_repository}/bin/git" <<'EOF_GIT'
#!/usr/bin/env bash
set -euo pipefail
case "$*" in
  "rev-parse --show-toplevel") printf '%s\n' "${MOBILE_REPO}" ;;
  "rev-parse --git-path mprlab-release") printf '%s\n' "${MOBILE_ARTIFACT_DIR}" ;;
  *) printf 'unexpected git command: %s\n' "$*" >&2; exit 97 ;;
esac
EOF_GIT
chmod +x "${mobile_repository}/bin/make" "${mobile_repository}/bin/node" "${mobile_repository}/bin/git"

(
  cd "${mobile_repository}"
  PATH="${mobile_repository}/bin:${PATH}" MOBILE_LOG="${mobile_log}" MOBILE_REPO="${mobile_repository}" MOBILE_ARTIFACT_DIR="${manifest_directory}" RELEASE_ENV_FILE="${mobile_repository}/configs/.env.loopaware" \
    ./scripts/publish-mobile.sh --preflight-only >/dev/null
)
[[ "$(wc -l <"${mobile_log}" | tr -d ' ')" == "4" ]]
[[ "$(sed -n '1p' "${mobile_log}")" == 'release-helper|verify-release-artifact' ]]
[[ "$(sed -n '2p' "${mobile_log}")" == 'make|--no-print-directory mobile-check' ]]
sed -n '3p' "${mobile_log}" | grep -Fq 'node|submit-ios.mjs|--dry-run'
sed -n '4p' "${mobile_log}" | grep -Fq 'node|publish-android-play.mjs|--dry-run'
sed -n '3,4p' "${mobile_log}" | grep -Fq 'MOBILE_RELEASE_TIMESTAMP=2026-07-11T00:00:00-0700'
sed -n '3,4p' "${mobile_log}" | grep -Fq -- "--manifest ${payload_directory}/loopaware-ios.json"
sed -n '3,4p' "${mobile_log}" | grep -Fq -- "--build-manifest ${payload_directory}/loopaware-android.json"

cp "${manifest_directory}/manifest.json" "${manifest_directory}/manifest.json.original"
python3 - "${manifest_directory}/manifest.json" <<'PY_OUTER_TIMESTAMP_DRIFT'
import json
import pathlib
import sys

path = pathlib.Path(sys.argv[1])
manifest = json.loads(path.read_text(encoding="utf-8"))
manifest["release_timestamp"] = "2026-07-11T01:00:00-0700"
path.write_text(json.dumps(manifest), encoding="utf-8")
PY_OUTER_TIMESTAMP_DRIFT
: >"${mobile_log}"
set +e
mobile_timestamp_output="$(
  cd "${mobile_repository}"
  PATH="${mobile_repository}/bin:${PATH}" MOBILE_LOG="${mobile_log}" MOBILE_REPO="${mobile_repository}" MOBILE_ARTIFACT_DIR="${manifest_directory}" RELEASE_ENV_FILE="${mobile_repository}/configs/.env.loopaware" \
    ./scripts/publish-mobile.sh --preflight-only 2>&1
)"
mobile_timestamp_status=$?
set -e
mv "${manifest_directory}/manifest.json.original" "${manifest_directory}/manifest.json"
[[ "${mobile_timestamp_status}" -ne 0 ]]
[[ "${mobile_timestamp_output}" == *"prepared mobile artifact timestamp does not match the outer release manifest"* ]]
[[ "$(wc -l <"${mobile_log}" | tr -d ' ')" == "1" ]]

: >"${mobile_log}"
(
  cd "${mobile_repository}"
  PATH="${mobile_repository}/bin:${PATH}" MOBILE_LOG="${mobile_log}" MOBILE_REPO="${mobile_repository}" MOBILE_ARTIFACT_DIR="${manifest_directory}" RELEASE_ENV_FILE="${mobile_repository}/configs/.env.loopaware" \
    ./scripts/publish-mobile.sh >/dev/null
)
[[ "$(wc -l <"${mobile_log}" | tr -d ' ')" == "6" ]]
[[ "$(sed -n '1p' "${mobile_log}")" == 'release-helper|verify-release-artifact' ]]
[[ "$(sed -n '2p' "${mobile_log}")" == 'make|--no-print-directory mobile-check' ]]
sed -n '3p' "${mobile_log}" | grep -Fq 'node|submit-ios.mjs|--dry-run'
sed -n '4p' "${mobile_log}" | grep -Fq 'node|publish-android-play.mjs|--dry-run'
sed -n '5p' "${mobile_log}" | grep -Fq 'node|submit-ios.mjs|--manifest'
sed -n '6p' "${mobile_log}" | grep -Fq 'node|publish-android-play.mjs|--aab'
[[ "$(sed -n '5p' "${mobile_log}")" != *"--dry-run"* ]]
[[ "$(sed -n '6p' "${mobile_log}")" != *"--dry-run"* ]]

: >"${mobile_log}"
set +e
uncertain_ios_output="$(
  cd "${mobile_repository}"
  PATH="${mobile_repository}/bin:${PATH}" MOBILE_LOG="${mobile_log}" MOBILE_REPO="${mobile_repository}" MOBILE_ARTIFACT_DIR="${manifest_directory}" RELEASE_ENV_FILE="${mobile_repository}/configs/.env.loopaware" FAKE_MOBILE_IOS_UPLOAD_FAILURE=1 \
    ./scripts/publish-mobile.sh 2>&1
)"
uncertain_ios_status=$?
set -e
[[ "${uncertain_ios_status}" -eq 41 ]]
[[ "${uncertain_ios_output}" == *"the iOS upload outcome is unknown"* ]]
[[ "${uncertain_ios_output}" == *"do not blindly retry"* ]]
[[ "$(wc -l <"${mobile_log}" | tr -d ' ')" == "5" ]]
[[ "$({ sed -n '5p' "${mobile_log}"; })" == node\|submit-ios.mjs\|* ]]

: >"${mobile_log}"
set +e
partial_mobile_output="$(
  cd "${mobile_repository}"
  PATH="${mobile_repository}/bin:${PATH}" MOBILE_LOG="${mobile_log}" MOBILE_REPO="${mobile_repository}" MOBILE_ARTIFACT_DIR="${manifest_directory}" RELEASE_ENV_FILE="${mobile_repository}/configs/.env.loopaware" FAKE_MOBILE_ANDROID_UPLOAD_FAILURE=1 \
    ./scripts/publish-mobile.sh 2>&1
)"
partial_mobile_status=$?
set -e
[[ "${partial_mobile_status}" -eq 42 ]]
[[ "${partial_mobile_output}" == *"the iOS build was accepted before Android failed"* ]]
[[ "${partial_mobile_output}" == *"prepare a new release timestamp instead of blindly retrying"* ]]
[[ "$(wc -l <"${mobile_log}" | tr -d ' ')" == "6" ]]
[[ "$({ sed -n '6p' "${mobile_log}"; })" == node\|publish-android-play.mjs\|* ]]

android_fixture="${temporary_directory}/android-publisher"
android_log="${temporary_directory}/android-publisher.log"
mkdir -p "${android_fixture}/bin"
android_aab="${android_fixture}/loopaware.aab"
android_mapping="${android_fixture}/mapping.txt"
android_manifest="${android_fixture}/manifest.json"
release_timestamp="2026-07-11T00:00:00Z"
printf 'fixture-aab\n' >"${android_aab}"
printf 'fixture-mapping\n' >"${android_mapping}"
node --input-type=module - "${repo_root}" "${release_timestamp}" "${android_aab}" "${android_mapping}" "${android_manifest}" <<'NODE_MANIFEST'
import crypto from "node:crypto";
import fs from "node:fs";
import path from "node:path";
import { pathToFileURL } from "node:url";

const [repoRoot, releaseTimestamp, aab, mapping, manifest] = process.argv.slice(2);
const versionModule = await import(pathToFileURL(path.join(repoRoot, "mobile/scripts/mobile-calver-version.mjs")));
const versioning = versionModule.createMobileCalVerVersion(releaseTimestamp);
const digest = (file) => crypto.createHash("sha256").update(fs.readFileSync(file)).digest("hex");
fs.writeFileSync(
  manifest,
  JSON.stringify({
    schema: "loopaware.mobile-android-bundle.v1",
    status: "passed",
    androidPackage: "com.mprlab.loopaware",
    versionName: versioning.releaseVersion,
    versionCode: versioning.androidVersionCode,
    sourceVersionCode: versioning.androidVersionCode,
    versionCodeSource: versioning.buildCodeSource,
    versioning,
    output: aab,
    sha256: digest(aab),
    deobfuscationFile: mapping,
    deobfuscationSha256: digest(mapping),
  }),
);
NODE_MANIFEST
cat >"${android_fixture}/bin/gcloud" <<'EOF_GCLOUD'
#!/usr/bin/env bash
set -euo pipefail
[[ "$*" == "auth application-default print-access-token" ]]
printf '%s\n' 'fixture-access-token'
EOF_GCLOUD
cat >"${android_fixture}/fake-fetch.cjs" <<'EOF_FETCH'
const fs = require("node:fs");

let editCreationCount = 0;

const jsonResponse = (payload, status = 200) =>
  new Response(JSON.stringify(payload), { status, headers: { "Content-Type": "application/json" } });

const preparedVersionCode = () => Number(process.env.ANDROID_PREPARED_VERSION_CODE);
const preparedSha256 = () => String(process.env.ANDROID_PREPARED_SHA256);

const existingBundles = () => {
  if (process.env.FAKE_ANDROID_REUSED_VERSION === "1") {
    return [{ versionCode: preparedVersionCode(), sha256: "1".repeat(64) }];
  }
  if (process.env.FAKE_ANDROID_NEWER_VERSION === "1") {
    return [{ versionCode: preparedVersionCode() + 1, sha256: "2".repeat(64) }];
  }
  return [{ versionCode: 123, sha256: "3".repeat(64) }];
};

const existingTrack = () => ({
  track: "internal",
  releases: [
    {
      name: "existing-release",
      status: process.env.FAKE_ANDROID_ACTIVE_TRACK === "1" ? "inProgress" : "completed",
      versionCodes: ["123"],
    },
  ],
});

globalThis.fetch = async (input, options = {}) => {
  const method = String(options.method || "GET");
  const url = String(input);
  const body = options.body ? Buffer.from(options.body).toString("utf8") : "";
  fs.appendFileSync(process.env.ANDROID_PUBLISHER_LOG, `${method}|${url}|${method === "PUT" ? body : ""}\n`);
  if (process.env.FAKE_ANDROID_SERVICE_DISABLED === "1" && method === "POST" && url.endsWith("/edits")) {
    return jsonResponse({ error: { code: 403, status: "PERMISSION_DENIED", details: [{ reason: "SERVICE_DISABLED" }] } }, 403);
  }
  if (method === "POST" && url.endsWith("/edits")) {
    editCreationCount += 1;
    const id = process.env.FAKE_ANDROID_ACTUAL === "1"
      ? editCreationCount === 1 ? "publish-edit" : "verification-edit"
      : "preflight-edit";
    return jsonResponse({ id });
  }
  if (process.env.FAKE_ANDROID_TRACK_FAILURE === "1" && method === "GET" && url.includes("/tracks/internal")) {
    return jsonResponse({ error: { code: 403, message: "fixture track denial" } }, 403);
  }
  if (process.env.FAKE_ANDROID_UPLOAD_FAILURE === "1" && method === "POST" && url.includes("/bundles?uploadType=media")) {
    return jsonResponse({ error: { code: 400, message: "fixture bundle rejection" } }, 400);
  }
  if (method === "GET" && (url.includes("/edits/preflight-edit/tracks/internal") || url.includes("/edits/publish-edit/tracks/internal"))) {
    return jsonResponse(existingTrack());
  }
  if (method === "GET" && url.includes("/edits/verification-edit/tracks/internal")) {
    const releases = [existingTrack().releases[0]];
    if (process.env.FAKE_ANDROID_POSTVERIFY_TRACK_MISSING !== "1") {
      releases.push({
        name: "2026.7.11",
        status: "completed",
        versionCodes: [String(preparedVersionCode())],
      });
    }
    return jsonResponse({ track: "internal", releases });
  }
  if (method === "GET" && (url.includes("/edits/preflight-edit/bundles") || url.includes("/edits/publish-edit/bundles"))) {
    return jsonResponse({ bundles: existingBundles() });
  }
  if (method === "GET" && url.includes("/edits/verification-edit/bundles")) {
    const publishedSha256 = process.env.FAKE_ANDROID_POSTVERIFY_HASH_MISMATCH === "1" ? "4".repeat(64) : preparedSha256();
    return jsonResponse({
      bundles: [...existingBundles(), { versionCode: preparedVersionCode(), sha256: publishedSha256 }],
    });
  }
  if (method === "PUT" && url.includes("/tracks/internal")) {
    if (process.env.FAKE_ANDROID_TRACK_UPDATE_FAILURE === "1") {
      return jsonResponse({ error: { code: 403, message: "fixture track update denial" } }, 403);
    }
    const payload = JSON.parse(body);
    if (url.includes("/edits/preflight-edit/")) {
      if (JSON.stringify(payload) !== JSON.stringify(existingTrack())) {
        return jsonResponse({ error: { code: 400, message: "preflight track payload drifted" } }, 400);
      }
    } else if (
      !Array.isArray(payload.releases) ||
      payload.releases.length !== 2 ||
      JSON.stringify(payload.releases[0]) !== JSON.stringify(existingTrack().releases[0]) ||
      payload.releases[1].name !== "2026.7.11" ||
      payload.releases[1].status !== "completed" ||
      JSON.stringify(payload.releases[1].versionCodes) !== JSON.stringify([String(preparedVersionCode())])
    ) {
      return jsonResponse({ error: { code: 400, message: "publication track payload drifted" } }, 400);
    }
    return jsonResponse(payload);
  }
  if (method === "POST" && url.includes("/edits/publish-edit/bundles?uploadType=media")) {
    return jsonResponse({ versionCode: preparedVersionCode(), sha256: preparedSha256() });
  }
  if (method === "POST" && url.includes("/edits/publish-edit/apks/") && url.includes("/deobfuscationFiles/proguard?uploadType=media")) {
    return jsonResponse({});
  }
  if (
    method === "POST" &&
    url.endsWith("/edits/publish-edit:commit?changesInReviewBehavior=ERROR_IF_IN_REVIEW")
  ) {
    return jsonResponse({ id: "publish-edit" });
  }
  if (method === "DELETE" && url.includes("/edits/")) {
    if (process.env.FAKE_ANDROID_DELETE_FAILURE === "1") {
      return jsonResponse({ error: { code: 500, message: "fixture cleanup denial" } }, 500);
    }
    return new Response("", { status: 200 });
  }
  return jsonResponse({ error: { code: 500, message: `unexpected request ${method} ${url}` } }, 500);
};
EOF_FETCH
chmod +x "${android_fixture}/bin/gcloud"

android_version_code="$(python3 - "${android_manifest}" <<'PY_ANDROID_VERSION_CODE'
import json
import pathlib
import sys

print(json.loads(pathlib.Path(sys.argv[1]).read_text(encoding="utf-8"))["versionCode"])
PY_ANDROID_VERSION_CODE
)"
android_aab_sha256="$(shasum -a 256 "${android_aab}" | awk '{print $1}')"
android_fixture_env=(
  PATH="${android_fixture}/bin:${PATH}"
  NODE_OPTIONS="--require=${android_fixture}/fake-fetch.cjs"
  ANDROID_PUBLISHER_LOG="${android_log}"
  ANDROID_PREPARED_VERSION_CODE="${android_version_code}"
  ANDROID_PREPARED_SHA256="${android_aab_sha256}"
)

env "${android_fixture_env[@]}" \
  node "${repo_root}/mobile/scripts/publish-android-play.mjs" \
    --mobile-dir "${repo_root}/mobile" \
    --release-timestamp "${release_timestamp}" \
    --aab "${android_aab}" \
    --mapping "${android_mapping}" \
    --build-manifest "${android_manifest}" \
    --dry-run >"${android_fixture}/success.json"
grep -Fq '"publisherAccess": "verified"' "${android_fixture}/success.json"
[[ "$(wc -l <"${android_log}" | tr -d ' ')" == "5" ]]
sed -n '1p' "${android_log}" | grep -Fq 'POST|https://androidpublisher.googleapis.com/androidpublisher/v3/applications/com.mprlab.loopaware/edits'
sed -n '2p' "${android_log}" | grep -Fq 'GET|https://androidpublisher.googleapis.com/androidpublisher/v3/applications/com.mprlab.loopaware/edits/preflight-edit/tracks/internal'
sed -n '3p' "${android_log}" | grep -Fq 'GET|https://androidpublisher.googleapis.com/androidpublisher/v3/applications/com.mprlab.loopaware/edits/preflight-edit/bundles'
sed -n '4p' "${android_log}" | grep -Fq 'PUT|https://androidpublisher.googleapis.com/androidpublisher/v3/applications/com.mprlab.loopaware/edits/preflight-edit/tracks/internal'
sed -n '4p' "${android_log}" | grep -Fq '"versionCodes":["123"]'
sed -n '5p' "${android_log}" | grep -Fq 'DELETE|https://androidpublisher.googleapis.com/androidpublisher/v3/applications/com.mprlab.loopaware/edits/preflight-edit'

: >"${android_log}"
set +e
unknown_android_option_output="$(
  env "${android_fixture_env[@]}" \
    node "${repo_root}/mobile/scripts/publish-android-play.mjs" \
      --mobile-dir "${repo_root}/mobile" \
      --release-timestamp "${release_timestamp}" \
      --aab "${android_aab}" \
      --mapping "${android_mapping}" \
      --build-manifest "${android_manifest}" \
      --dry-run-typo=true 2>&1
)"
unknown_android_option_status=$?
set -e
[[ "${unknown_android_option_status}" -eq 2 ]]
[[ "${unknown_android_option_output}" == *"unknown option: --dry-run-typo"* ]]
[[ ! -s "${android_log}" ]]

: >"${android_log}"
set +e
service_disabled_output="$(
  env "${android_fixture_env[@]}" FAKE_ANDROID_SERVICE_DISABLED=1 \
    node "${repo_root}/mobile/scripts/publish-android-play.mjs" \
      --mobile-dir "${repo_root}/mobile" \
      --release-timestamp "${release_timestamp}" \
      --aab "${android_aab}" \
      --mapping "${android_mapping}" \
      --build-manifest "${android_manifest}" \
      --dry-run 2>&1
)"
service_disabled_status=$?
set -e
[[ "${service_disabled_status}" -eq 2 ]]
[[ "${service_disabled_output}" == *"SERVICE_DISABLED"* ]]
[[ "$(wc -l <"${android_log}" | tr -d ' ')" == "1" ]]

: >"${android_log}"
set +e
track_failure_output="$(
  env "${android_fixture_env[@]}" FAKE_ANDROID_TRACK_FAILURE=1 \
    node "${repo_root}/mobile/scripts/publish-android-play.mjs" \
      --mobile-dir "${repo_root}/mobile" \
      --release-timestamp "${release_timestamp}" \
      --aab "${android_aab}" \
      --mapping "${android_mapping}" \
      --build-manifest "${android_manifest}" \
      --dry-run 2>&1
)"
track_failure_status=$?
set -e
[[ "${track_failure_status}" -eq 2 ]]
[[ "${track_failure_output}" == *"fixture track denial"* ]]
[[ "$(wc -l <"${android_log}" | tr -d ' ')" == "3" ]]
sed -n '3p' "${android_log}" | grep -Fq 'DELETE|https://androidpublisher.googleapis.com/androidpublisher/v3/applications/com.mprlab.loopaware/edits/preflight-edit'

: >"${android_log}"
set +e
track_update_failure_output="$(
  env "${android_fixture_env[@]}" FAKE_ANDROID_TRACK_UPDATE_FAILURE=1 \
    node "${repo_root}/mobile/scripts/publish-android-play.mjs" \
      --mobile-dir "${repo_root}/mobile" \
      --release-timestamp "${release_timestamp}" \
      --aab "${android_aab}" \
      --mapping "${android_mapping}" \
      --build-manifest "${android_manifest}" \
      --dry-run 2>&1
)"
track_update_failure_status=$?
set -e
[[ "${track_update_failure_status}" -eq 2 ]]
[[ "${track_update_failure_output}" == *"fixture track update denial"* ]]
[[ "$(wc -l <"${android_log}" | tr -d ' ')" == "5" ]]
sed -n '5p' "${android_log}" | grep -Fq 'DELETE|https://androidpublisher.googleapis.com/androidpublisher/v3/applications/com.mprlab.loopaware/edits/preflight-edit'

: >"${android_log}"
set +e
combined_failure_output="$(
  env "${android_fixture_env[@]}" FAKE_ANDROID_TRACK_FAILURE=1 FAKE_ANDROID_DELETE_FAILURE=1 \
    node "${repo_root}/mobile/scripts/publish-android-play.mjs" \
      --mobile-dir "${repo_root}/mobile" \
      --release-timestamp "${release_timestamp}" \
      --aab "${android_aab}" \
      --mapping "${android_mapping}" \
      --build-manifest "${android_manifest}" \
      --dry-run 2>&1
)"
combined_failure_status=$?
set -e
[[ "${combined_failure_status}" -eq 2 ]]
[[ "${combined_failure_output}" == *"fixture track denial"* ]]
[[ "${combined_failure_output}" == *"failed edit cleanup"* ]]
[[ "${combined_failure_output}" == *"fixture cleanup denial"* ]]
[[ "$(wc -l <"${android_log}" | tr -d ' ')" == "3" ]]

: >"${android_log}"
set +e
reused_version_output="$(
  env "${android_fixture_env[@]}" FAKE_ANDROID_REUSED_VERSION=1 \
    node "${repo_root}/mobile/scripts/publish-android-play.mjs" \
      --mobile-dir "${repo_root}/mobile" \
      --release-timestamp "${release_timestamp}" \
      --aab "${android_aab}" \
      --mapping "${android_mapping}" \
      --build-manifest "${android_manifest}" \
      --dry-run 2>&1
)"
reused_version_status=$?
set -e
[[ "${reused_version_status}" -eq 2 ]]
[[ "${reused_version_output}" == *"Android versionCode ${android_version_code} is already present in Google Play"* ]]
[[ "$(wc -l <"${android_log}" | tr -d ' ')" == "4" ]]
! grep -Fq 'PUT|' "${android_log}"
sed -n '4p' "${android_log}" | grep -Fq '/edits/preflight-edit'

: >"${android_log}"
set +e
newer_version_output="$(
  env "${android_fixture_env[@]}" FAKE_ANDROID_NEWER_VERSION=1 \
    node "${repo_root}/mobile/scripts/publish-android-play.mjs" \
      --mobile-dir "${repo_root}/mobile" \
      --release-timestamp "${release_timestamp}" \
      --aab "${android_aab}" \
      --mapping "${android_mapping}" \
      --build-manifest "${android_manifest}" \
      --dry-run 2>&1
)"
newer_version_status=$?
set -e
[[ "${newer_version_status}" -eq 2 ]]
[[ "${newer_version_output}" == *"must be greater than existing Google Play maximum"* ]]
[[ "$(wc -l <"${android_log}" | tr -d ' ')" == "4" ]]
! grep -Fq 'PUT|' "${android_log}"

: >"${android_log}"
set +e
active_track_output="$(
  env "${android_fixture_env[@]}" FAKE_ANDROID_ACTIVE_TRACK=1 \
    node "${repo_root}/mobile/scripts/publish-android-play.mjs" \
      --mobile-dir "${repo_root}/mobile" \
      --release-timestamp "${release_timestamp}" \
      --aab "${android_aab}" \
      --mapping "${android_mapping}" \
      --build-manifest "${android_manifest}" \
      --dry-run 2>&1
)"
active_track_status=$?
set -e
[[ "${active_track_status}" -eq 2 ]]
[[ "${active_track_output}" == *"contains inProgress release state; refusing to replace an active/manual rollout"* ]]
[[ "$(wc -l <"${android_log}" | tr -d ' ')" == "4" ]]
! grep -Fq 'PUT|' "${android_log}"

: >"${android_log}"
set +e
upload_failure_output="$(
  env "${android_fixture_env[@]}" FAKE_ANDROID_ACTUAL=1 FAKE_ANDROID_UPLOAD_FAILURE=1 \
    node "${repo_root}/mobile/scripts/publish-android-play.mjs" \
      --mobile-dir "${repo_root}/mobile" \
      --release-timestamp "${release_timestamp}" \
      --aab "${android_aab}" \
      --mapping "${android_mapping}" \
      --build-manifest "${android_manifest}" 2>&1
)"
upload_failure_status=$?
set -e
[[ "${upload_failure_status}" -eq 2 ]]
[[ "${upload_failure_output}" == *"fixture bundle rejection"* ]]
[[ "$(wc -l <"${android_log}" | tr -d ' ')" == "5" ]]
sed -n '2p' "${android_log}" | grep -Fq '/edits/publish-edit/bundles'
sed -n '3p' "${android_log}" | grep -Fq '/edits/publish-edit/tracks/internal'
sed -n '4p' "${android_log}" | grep -Fq '/edits/publish-edit/bundles?uploadType=media'
sed -n '5p' "${android_log}" | grep -Fq 'DELETE|https://androidpublisher.googleapis.com/androidpublisher/v3/applications/com.mprlab.loopaware/edits/publish-edit'

: >"${android_log}"
env "${android_fixture_env[@]}" FAKE_ANDROID_ACTUAL=1 \
  node "${repo_root}/mobile/scripts/publish-android-play.mjs" \
    --mobile-dir "${repo_root}/mobile" \
    --release-timestamp "${release_timestamp}" \
    --aab "${android_aab}" \
    --mapping "${android_mapping}" \
    --build-manifest "${android_manifest}" >"${android_fixture}/published.json"
grep -Fq '"status": "submitted"' "${android_fixture}/published.json"
grep -Fq "\"uploadedVersionCode\": ${android_version_code}" "${android_fixture}/published.json"
[[ "$(wc -l <"${android_log}" | tr -d ' ')" == "11" ]]
sed -n '4p' "${android_log}" | grep -Fq '/edits/publish-edit/bundles?uploadType=media'
sed -n '5p' "${android_log}" | grep -Fq "/edits/publish-edit/apks/${android_version_code}/deobfuscationFiles/proguard?uploadType=media"
sed -n '6p' "${android_log}" | grep -Fq 'PUT|https://androidpublisher.googleapis.com/androidpublisher/v3/applications/com.mprlab.loopaware/edits/publish-edit/tracks/internal'
sed -n '6p' "${android_log}" | grep -Fq '"name":"existing-release","status":"completed","versionCodes":["123"]'
sed -n '6p' "${android_log}" | grep -Fq "\"name\":\"2026.7.11\",\"versionCodes\":[\"${android_version_code}\"]"
sed -n '7p' "${android_log}" | grep -Fq 'POST|https://androidpublisher.googleapis.com/androidpublisher/v3/applications/com.mprlab.loopaware/edits/publish-edit:commit?changesInReviewBehavior=ERROR_IF_IN_REVIEW'
sed -n '8p' "${android_log}" | grep -Fq 'POST|https://androidpublisher.googleapis.com/androidpublisher/v3/applications/com.mprlab.loopaware/edits'
sed -n '9p' "${android_log}" | grep -Fq '/edits/verification-edit/bundles'
sed -n '10p' "${android_log}" | grep -Fq '/edits/verification-edit/tracks/internal'
sed -n '11p' "${android_log}" | grep -Fq 'DELETE|https://androidpublisher.googleapis.com/androidpublisher/v3/applications/com.mprlab.loopaware/edits/verification-edit'
! grep -Fq 'DELETE|https://androidpublisher.googleapis.com/androidpublisher/v3/applications/com.mprlab.loopaware/edits/publish-edit' "${android_log}"

: >"${android_log}"
set +e
postverification_failure_output="$(
  env "${android_fixture_env[@]}" FAKE_ANDROID_ACTUAL=1 FAKE_ANDROID_POSTVERIFY_HASH_MISMATCH=1 \
    node "${repo_root}/mobile/scripts/publish-android-play.mjs" \
      --mobile-dir "${repo_root}/mobile" \
      --release-timestamp "${release_timestamp}" \
      --aab "${android_aab}" \
      --mapping "${android_mapping}" \
      --build-manifest "${android_manifest}" 2>&1
)"
postverification_failure_status=$?
set -e
[[ "${postverification_failure_status}" -eq 2 ]]
[[ "${postverification_failure_output}" == *"was committed, but post-publication verification failed"* ]]
[[ "${postverification_failure_output}" == *"committed Android App Bundle SHA-256"* ]]
[[ "${postverification_failure_output}" == *"inspect Google Play before preparing another single-use versionCode"* ]]
[[ "$(wc -l <"${android_log}" | tr -d ' ')" == "10" ]]
sed -n '7p' "${android_log}" | grep -Fq '/edits/publish-edit:commit'
sed -n '9p' "${android_log}" | grep -Fq '/edits/verification-edit/bundles'
sed -n '10p' "${android_log}" | grep -Fq 'DELETE|https://androidpublisher.googleapis.com/androidpublisher/v3/applications/com.mprlab.loopaware/edits/verification-edit'
! grep -Fq 'DELETE|https://androidpublisher.googleapis.com/androidpublisher/v3/applications/com.mprlab.loopaware/edits/publish-edit' "${android_log}"

container_repository="${temporary_directory}/container-fixture"
container_docker_log="${temporary_directory}/container-docker.log"
container_docker_state="${temporary_directory}/container-docker.state"
container_platform_state="${temporary_directory}/container-platform.state"
container_version_state="${temporary_directory}/container-version.state"
container_latest_state="${temporary_directory}/container-latest.state"
container_bin="${container_repository}/bin"
export CONTAINER_PLATFORM_STATE="${container_platform_state}"
export CONTAINER_VERSION_STATE="${container_version_state}"
export CONTAINER_LATEST_STATE="${container_latest_state}"
mkdir -p "${container_repository}/scripts/release" "${container_bin}"
cp "${repo_root}/scripts/release/publish_container_artifacts.sh" "${container_repository}/scripts/release/publish_container_artifacts.sh"
cp "${repo_root}/scripts/release/docker_identity.sh" "${container_repository}/scripts/release/docker_identity.sh"
cp "${repo_root}/scripts/release/container_archive_image_id.py" "${container_repository}/scripts/release/container_archive_image_id.py"
cat >"${container_repository}/scripts/release/release_helper.py" <<'EOF_HELPER'
#!/usr/bin/env bash
set -euo pipefail
[[ "$*" == "verify-release-artifact" ]]
EOF_HELPER
chmod +x "${container_repository}/scripts/release/release_helper.py"
git -C "${container_repository}" init -b master >/dev/null
container_artifact_directory="${container_repository}/.git/mprlab-release"
container_payload_directory="${container_artifact_directory}/payloads/containers/loopaware"
mkdir -p "${container_payload_directory}"
container_archive="${container_payload_directory}/linux-amd64.tar"
container_inspect_image_id='sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa'
export CONTAINER_ARCHIVE="${container_archive}"
export CONTAINER_INSPECT_IMAGE_ID="${container_inspect_image_id}"
container_source_commit="2222222222222222222222222222222222222222"
container_image_id="$(python3 - "${container_archive}" "${container_source_commit}" <<'PY_CONTAINER_ARCHIVE'
import hashlib
import io
import json
import pathlib
import sys
import tarfile

archive_path = pathlib.Path(sys.argv[1])
config = {
    "os": "linux",
    "architecture": "amd64",
    "config": {
        "Labels": {
            "org.opencontainers.image.revision": sys.argv[2],
            "org.opencontainers.image.version": "v1.2.3",
            "org.opencontainers.image.source": "https://github.com/tyemirov/loopaware",
        }
    }
}
config_bytes = json.dumps(config, sort_keys=True, separators=(",", ":")).encode()
config_digest = hashlib.sha256(config_bytes).hexdigest()
manifest_bytes = json.dumps(
    [{"Config": f"{config_digest}.json", "RepoTags": ["mprlab-release.local/loopaware:v1.2.3-linux-amd64"], "Layers": []}]
).encode()
with tarfile.open(archive_path, "w") as archive:
    for name, payload in (("manifest.json", manifest_bytes), (f"{config_digest}.json", config_bytes)):
        info = tarfile.TarInfo(name)
        info.size = len(payload)
        archive.addfile(info, io.BytesIO(payload))
print(f"sha256:{config_digest}")
PY_CONTAINER_ARCHIVE
)"
container_sha256="$(shasum -a 256 "${container_archive}" | awk '{print $1}')"
python3 - "${container_artifact_directory}/manifest.json" "${container_source_commit}" <<'PY_CONTAINER_MANIFEST'
import json
import pathlib
import sys

pathlib.Path(sys.argv[1]).write_text(json.dumps({"version": "v1.2.3", "source_commit": sys.argv[2]}), encoding="utf-8")
PY_CONTAINER_MANIFEST
python3 - "${container_payload_directory}/container.json" "${container_sha256}" "${container_image_id}" <<'PY_CONTAINER_DESCRIPTOR'
import json
import pathlib
import sys

pathlib.Path(sys.argv[1]).write_text(
    json.dumps(
        {
            "schema_version": 1,
            "artifact_kind": "mprlab.container",
            "name": "loopaware",
            "image": "ghcr.io/tyemirov/loopaware",
            "version": "v1.2.3",
            "platforms": [
                {
                    "platform": "linux/amd64",
                    "token": "linux-amd64",
                    "local_ref": "mprlab-release.local/loopaware:v1.2.3-linux-amd64",
                    "image_id": sys.argv[3],
                    "archive": "payloads/containers/loopaware/linux-amd64.tar",
                    "sha256": sys.argv[2],
                }
            ],
        }
    ),
    encoding="utf-8",
)
PY_CONTAINER_DESCRIPTOR
cat >"${container_bin}/docker" <<'EOF_CONTAINER_DOCKER'
#!/usr/bin/env bash
set -euo pipefail
platform_ref='ghcr.io/tyemirov/loopaware:v1.2.3-linux-amd64'
version_ref='ghcr.io/tyemirov/loopaware:v1.2.3'
latest_ref='ghcr.io/tyemirov/loopaware:latest'
published_platform_digest='sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc'
published_index_digest='sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd'

state_for_ref() {
  case "$1" in
    "${platform_ref}") printf '%s\n' "${CONTAINER_PLATFORM_STATE}" ;;
    "${version_ref}") printf '%s\n' "${CONTAINER_VERSION_STATE}" ;;
    "${latest_ref}") printf '%s\n' "${CONTAINER_LATEST_STATE}" ;;
    *) return 1 ;;
  esac
}

read_state() {
  local state_file="$1"
  [[ -f "${state_file}" ]] || {
    printf 'manifest unknown: fixture reference is absent\n' >&2
    return 1
  }
  IFS='|' read -r state_digest state_config <"${state_file}"
  [[ "${state_digest}" =~ ^sha256:[0-9a-f]{64}$ && "${state_config}" =~ ^sha256:[0-9a-f]{64}$ ]]
}

config_for_digest() {
  local requested_digest="$1"
  local state_file
  for state_file in "${CONTAINER_VERSION_STATE}" "${CONTAINER_PLATFORM_STATE}" "${CONTAINER_LATEST_STATE}"; do
    [[ -f "${state_file}" ]] || continue
    read_state "${state_file}"
    if [[ "${state_digest}" == "${requested_digest}" ]]; then
      printf '%s\n' "${state_config}"
      return 0
    fi
  done
  printf 'manifest unknown: fixture digest is absent\n' >&2
  return 1
}

if [[ "$*" == "context show" ]]; then
  printf '%s\n' fixture
  exit 0
fi
if [[ "$*" == "context inspect fixture --format {{.Endpoints.docker.Host}}" ]]; then
  printf '%s\n' 'unix:///tmp/fixture-docker.sock'
  exit 0
fi
if [[ "$*" == "buildx version" ]]; then
  exit 0
fi
if [[ "$1" == "login" ]]; then
  token="$(cat)"
  [[ "$*" == "login ghcr.io --username fixture-user --password-stdin" && "${token}" == "fixture-token" ]] || {
    printf 'unexpected docker login: %s\n' "$*" >&2
    exit 97
  }
  if [[ "${FAKE_DOCKER_LOGIN_FAILURE:-0}" == "1" ]]; then
    printf 'fixture docker login failure\n' >&2
    exit 41
  fi
  printf 'LOGIN|%s\n' "$*" >>"${CONTAINER_DOCKER_LOG}"
  exit 0
fi
if [[ "$1 $2" == "image inspect" ]]; then
  printf 'INSPECT|%s\n' "$3" >>"${CONTAINER_DOCKER_LOG}"
  [[ -f "${CONTAINER_DOCKER_STATE}" ]] || exit 1
  if [[ "$*" == *"{{.Os}}/{{.Architecture}}"* ]]; then
    printf '%s\n' "${FAKE_DOCKER_PLATFORM:-linux/amd64}"
  else
    cat "${CONTAINER_DOCKER_STATE}"
  fi
  exit 0
fi
if [[ "$1" == "save" && "$2" == "--output" ]]; then
  printf 'SAVE|%s\n' "$4" >>"${CONTAINER_DOCKER_LOG}"
  cp "${CONTAINER_ARCHIVE}" "$3"
  exit 0
fi
if [[ "$1" == "load" && "$2" == "--input" ]]; then
  printf 'LOAD|%s\n' "$3" >>"${CONTAINER_DOCKER_LOG}"
  if [[ "${FAKE_DOCKER_LOAD_FAILURE:-0}" == "1" ]]; then
    printf 'fixture docker load failure\n' >&2
    exit 42
  fi
  printf '%s\n' "${CONTAINER_INSPECT_IMAGE_ID}" >"${CONTAINER_DOCKER_STATE}"
  printf 'Loaded image: mprlab-release.local/loopaware:v1.2.3-linux-amd64\n'
  exit 0
fi
if [[ "$1 $2 ${3:-}" == "image rm --force" ]]; then
  printf 'REMOVE|%s\n' "$4" >>"${CONTAINER_DOCKER_LOG}"
  rm -f "${CONTAINER_DOCKER_STATE}"
  exit 0
fi
if [[ "$1" == "tag" ]]; then
  printf 'TAG|%s\n' "$*" >>"${CONTAINER_DOCKER_LOG}"
  exit 0
fi
if [[ "$1" == "push" ]]; then
  printf 'PUSH|%s\n' "$*" >>"${CONTAINER_DOCKER_LOG}"
  [[ "$2" == "--platform" && "$3" == "linux/amd64" && "$4" == "${platform_ref}" ]] || {
    printf 'unexpected platform-specific push: %s\n' "$*" >&2
    exit 97
  }
  printf '%s|%s\n' "${published_platform_digest}" "${CONTAINER_EXPECTED_IMAGE_ID}" >"${CONTAINER_PLATFORM_STATE}"
  printf '%s\n' 'fixture: pushed' "digest: ${published_platform_digest} size: 1234"
  exit 0
fi
if [[ "$1 $2 $3" == "buildx imagetools create" ]]; then
  printf 'CREATE|%s\n' "$*" >>"${CONTAINER_DOCKER_LOG}"
  target_ref="$5"
  source_ref="$6"
  source_digest="${source_ref##*@}"
  [[ "$4" == "--tag" && "${source_ref}" == "ghcr.io/tyemirov/loopaware@${source_digest}" && "${source_digest}" =~ ^sha256:[0-9a-f]{64}$ ]] || {
    printf 'manifest creation did not use the immutable pushed digest: %s\n' "$*" >&2
    exit 96
  }
  source_config="$(config_for_digest "${source_digest}")"
  case "${target_ref}" in
    "${version_ref}") printf '%s|%s\n' "${source_digest}" "${source_config}" >"${CONTAINER_VERSION_STATE}" ;;
    "${latest_ref}") printf '%s|%s\n' "${source_digest}" "${source_config}" >"${CONTAINER_LATEST_STATE}" ;;
    *) printf 'unexpected imagetools target: %s\n' "${target_ref}" >&2; exit 97 ;;
  esac
  exit 0
fi
if [[ "$1 $2 $3" == "buildx imagetools inspect" ]]; then
  printf 'IMAGETOOLS_INSPECT|%s\n' "$*" >>"${CONTAINER_DOCKER_LOG}"
  inspected_ref="$4"
  if [[ "${inspected_ref}" == *@sha256:* ]]; then
    [[ "${5:-}" == "--raw" ]] || { printf 'digest inspection requires --raw\n' >&2; exit 97; }
    inspected_digest="${inspected_ref##*@}"
    inspected_config="$(config_for_digest "${inspected_digest}")"
    printf '{"schemaVersion":2,"config":{"digest":"%s"}}\n' "${inspected_config}"
    exit 0
  fi
  state_file="$(state_for_ref "${inspected_ref}")" || {
    printf 'unexpected imagetools ref: %s\n' "${inspected_ref}" >&2
    exit 97
  }
  read_state "${state_file}"
  if [[ "${5:-}" == "--raw" ]]; then
    if [[ "${inspected_ref}" == "${platform_ref}" ]]; then
      printf '{"schemaVersion":2,"config":{"digest":"%s"}}\n' "${state_config}"
    elif [[ "${inspected_ref}" == "${latest_ref}" && "${FAKE_LATEST_ATTESTATION:-0}" == "1" ]]; then
      printf '{"manifests":[{"digest":"%s","platform":{"os":"linux","architecture":"amd64"}},{"digest":"sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee","annotations":{"vnd.docker.reference.digest":"%s","vnd.docker.reference.type":"attestation-manifest"},"platform":{"os":"unknown","architecture":"unknown"}}]}\n' "${state_digest}" "${state_digest}"
    elif [[ "${inspected_ref}" == "${latest_ref}" && "${FAKE_LATEST_EXTRA_PLATFORM:-0}" == "1" ]]; then
      printf '{"manifests":[{"digest":"%s","platform":{"os":"linux","architecture":"amd64"}},{"digest":"sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee","platform":{"os":"linux","architecture":"arm64"}}]}\n' "${state_digest}"
    else
      printf '{"manifests":[{"digest":"%s","platform":{"os":"linux","architecture":"amd64"}}]}\n' "${state_digest}"
    fi
  elif [[ -z "${5:-}" ]]; then
    if [[ "${inspected_ref}" == "${platform_ref}" ]]; then
      printf '%s\n' 'Name: fixture' "Digest: ${state_digest}"
    else
      printf '%s\n' 'Name: fixture' "Digest: ${published_index_digest}"
    fi
  else
    printf 'unexpected imagetools inspect arguments: %s\n' "$*" >&2
    exit 97
  fi
  exit 0
fi
printf 'unexpected docker command: %s\n' "$*" >&2
exit 97
EOF_CONTAINER_DOCKER
cat >"${container_bin}/gh" <<'EOF_CONTAINER_GH'
#!/usr/bin/env bash
set -euo pipefail
case "$*" in
  "api user --jq .login") printf '%s\n' 'fixture-user' ;;
  "auth token") printf '%s\n' 'fixture-token' ;;
  *) printf 'unexpected gh command: %s\n' "$*" >&2; exit 97 ;;
esac
EOF_CONTAINER_GH
chmod +x "${container_bin}/docker" "${container_bin}/gh"

container_output="$(
  cd "${container_repository}"
  PATH="${container_bin}:${PATH}" CONTAINER_DOCKER_LOG="${container_docker_log}" CONTAINER_DOCKER_STATE="${container_docker_state}" CONTAINER_EXPECTED_IMAGE_ID="${container_image_id}" PUBLISH_PLATFORMS="linux/amd64" \
    ./scripts/release/publish_container_artifacts.sh --preflight-only
)"
[[ "${container_output}" == *"prepared archive loaded with its exact image id"* ]]
grep -Fqx 'LOGIN|login ghcr.io --username fixture-user --password-stdin' "${container_docker_log}"
! grep -Fq 'fixture-token' "${container_docker_log}"
! grep -Fq 'PUSH|' "${container_docker_log}"
! grep -Fq 'CREATE|' "${container_docker_log}"
grep -Fq 'LOAD|' "${container_docker_log}"
grep -Fq 'SAVE|mprlab-release.local/loopaware:v1.2.3-linux-amd64' "${container_docker_log}"
grep -Fq 'REMOVE|mprlab-release.local/loopaware:v1.2.3-linux-amd64' "${container_docker_log}"
[[ ! -f "${container_docker_state}" ]]

: >"${container_docker_log}"
set +e
container_load_output="$(
  cd "${container_repository}"
  PATH="${container_bin}:${PATH}" CONTAINER_DOCKER_LOG="${container_docker_log}" CONTAINER_DOCKER_STATE="${container_docker_state}" CONTAINER_EXPECTED_IMAGE_ID="${container_image_id}" FAKE_DOCKER_LOAD_FAILURE=1 PUBLISH_PLATFORMS="linux/amd64" \
    ./scripts/release/publish_container_artifacts.sh --preflight-only 2>&1
)"
container_load_status=$?
set -e
[[ "${container_load_status}" -ne 0 ]]
[[ "${container_load_output}" == *"prepared container archive cannot be loaded"* ]]
[[ "${container_load_output}" == *"fixture docker load failure"* ]]
grep -Fq 'LOAD|' "${container_docker_log}"

: >"${container_docker_log}"
rm -f "${container_docker_state}"
set +e
container_platform_output="$(
  cd "${container_repository}"
  PATH="${container_bin}:${PATH}" CONTAINER_DOCKER_LOG="${container_docker_log}" CONTAINER_DOCKER_STATE="${container_docker_state}" CONTAINER_EXPECTED_IMAGE_ID="${container_image_id}" FAKE_DOCKER_PLATFORM=linux/arm64 PUBLISH_PLATFORMS="linux/amd64" \
    ./scripts/release/publish_container_artifacts.sh --preflight-only 2>&1
)"
container_platform_status=$?
set -e
[[ "${container_platform_status}" -ne 0 ]]
[[ "${container_platform_output}" == *"loaded preflight image platform linux/arm64 does not match prepared linux/amd64"* ]]

: >"${container_docker_log}"
set +e
container_login_output="$(
  cd "${container_repository}"
  PATH="${container_bin}:${PATH}" CONTAINER_DOCKER_LOG="${container_docker_log}" CONTAINER_DOCKER_STATE="${container_docker_state}" CONTAINER_EXPECTED_IMAGE_ID="${container_image_id}" FAKE_DOCKER_LOGIN_FAILURE=1 PUBLISH_PLATFORMS="linux/amd64" \
    ./scripts/release/publish_container_artifacts.sh --preflight-only 2>&1
)"
container_login_status=$?
set -e
[[ "${container_login_status}" -ne 0 ]]
[[ "${container_login_output}" == *"fixture docker login failure"* ]]
! grep -Fq 'PUSH|' "${container_docker_log}"
! grep -Fq 'CREATE|' "${container_docker_log}"

: >"${container_docker_log}"
rm -f "${container_docker_state}"
container_publish_output="$(
  cd "${container_repository}"
  PATH="${container_bin}:${PATH}" CONTAINER_DOCKER_LOG="${container_docker_log}" CONTAINER_DOCKER_STATE="${container_docker_state}" CONTAINER_EXPECTED_IMAGE_ID="${container_image_id}" PUBLISH_PLATFORMS="linux/amd64" \
    ./scripts/release/publish_container_artifacts.sh
)"
[[ "${container_publish_output}" == *"Published ghcr.io/tyemirov/loopaware:v1.2.3 at sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd."* ]]
grep -Fqx 'LOGIN|login ghcr.io --username fixture-user --password-stdin' "${container_docker_log}"
grep -Fq 'PUSH|push --platform linux/amd64 ghcr.io/tyemirov/loopaware:v1.2.3-linux-amd64' "${container_docker_log}"
grep -Fq 'SAVE|mprlab-release.local/loopaware:v1.2.3-linux-amd64' "${container_docker_log}"
[[ "$(grep -c '^CREATE|' "${container_docker_log}")" == "2" ]]
grep '^CREATE|' "${container_docker_log}" | grep -Fq 'ghcr.io/tyemirov/loopaware@sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc'
[[ "$(grep '^CREATE|' "${container_docker_log}")" != *"v1.2.3-linux-amd64"* ]]

rm -f "${container_version_state}"
: >"${container_docker_log}"
container_missing_version_output="$(
  cd "${container_repository}"
  PATH="${container_bin}:${PATH}" CONTAINER_DOCKER_LOG="${container_docker_log}" CONTAINER_DOCKER_STATE="${container_docker_state}" CONTAINER_EXPECTED_IMAGE_ID="${container_image_id}" PUBLISH_PLATFORMS="linux/amd64" \
    ./scripts/release/publish_container_artifacts.sh
)"
[[ "${container_missing_version_output}" == *"Creating ghcr.io/tyemirov/loopaware:v1.2.3"* ]]
! grep -Fq 'PUSH|' "${container_docker_log}"
! grep -Fq 'LOAD|' "${container_docker_log}"
[[ "$(grep -c '^CREATE|' "${container_docker_log}")" == "2" ]]
grep '^CREATE|' "${container_docker_log}" | grep -Fq -- '--tag ghcr.io/tyemirov/loopaware:v1.2.3 ghcr.io/tyemirov/loopaware@sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc'

: >"${container_docker_log}"
container_rerun_output="$(
  cd "${container_repository}"
  PATH="${container_bin}:${PATH}" CONTAINER_DOCKER_LOG="${container_docker_log}" CONTAINER_DOCKER_STATE="${container_docker_state}" CONTAINER_EXPECTED_IMAGE_ID="${container_image_id}" PUBLISH_PLATFORMS="linux/amd64" \
    ./scripts/release/publish_container_artifacts.sh
)"
[[ "${container_rerun_output}" == *"Preserving immutable existing ghcr.io/tyemirov/loopaware:v1.2.3"* ]]
[[ "${container_rerun_output}" == *"Published ghcr.io/tyemirov/loopaware:v1.2.3 at sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd."* ]]
! grep -Fq 'PUSH|' "${container_docker_log}"
! grep -Fq 'LOAD|' "${container_docker_log}"
[[ "$(grep -c '^CREATE|' "${container_docker_log}")" == "1" ]]
grep '^CREATE|' "${container_docker_log}" | grep -Fq -- '--tag ghcr.io/tyemirov/loopaware:latest ghcr.io/tyemirov/loopaware@sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc'

: >"${container_docker_log}"
container_latest_attestation_output="$(
  cd "${container_repository}"
  PATH="${container_bin}:${PATH}" CONTAINER_DOCKER_LOG="${container_docker_log}" CONTAINER_DOCKER_STATE="${container_docker_state}" CONTAINER_EXPECTED_IMAGE_ID="${container_image_id}" FAKE_LATEST_ATTESTATION=1 PUBLISH_PLATFORMS="linux/amd64" \
    ./scripts/release/publish_container_artifacts.sh --preflight-only
)"
[[ "${container_latest_attestation_output}" == *"Verified prepared container publication inputs for loopaware:v1.2.3."* ]]
grep -Fqx 'LOGIN|login ghcr.io --username fixture-user --password-stdin' "${container_docker_log}"
! grep -Fq 'PUSH|' "${container_docker_log}"
! grep -Fq 'CREATE|' "${container_docker_log}"

: >"${container_docker_log}"
set +e
container_latest_extra_output="$(
  cd "${container_repository}"
  PATH="${container_bin}:${PATH}" CONTAINER_DOCKER_LOG="${container_docker_log}" CONTAINER_DOCKER_STATE="${container_docker_state}" CONTAINER_EXPECTED_IMAGE_ID="${container_image_id}" FAKE_LATEST_EXTRA_PLATFORM=1 PUBLISH_PLATFORMS="linux/amd64" \
    ./scripts/release/publish_container_artifacts.sh --preflight-only 2>&1
)"
container_latest_extra_status=$?
set -e
[[ "${container_latest_extra_status}" -ne 0 ]]
[[ "${container_latest_extra_output}" == *"existing mutable index must contain exactly one deployable manifest, got 2"* ]]
! grep -Fq 'PUSH|' "${container_docker_log}"
! grep -Fq 'CREATE|' "${container_docker_log}"

bad_platform_digest='sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee'
bad_config_digest='sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff'
printf '%s|%s\n' "${bad_platform_digest}" "${bad_config_digest}" >"${container_platform_state}"
: >"${container_docker_log}"
set +e
container_platform_immutability_output="$(
  cd "${container_repository}"
  PATH="${container_bin}:${PATH}" CONTAINER_DOCKER_LOG="${container_docker_log}" CONTAINER_DOCKER_STATE="${container_docker_state}" CONTAINER_EXPECTED_IMAGE_ID="${container_image_id}" PUBLISH_PLATFORMS="linux/amd64" \
    ./scripts/release/publish_container_artifacts.sh --preflight-only 2>&1
)"
container_platform_immutability_status=$?
set -e
[[ "${container_platform_immutability_status}" -ne 0 ]]
[[ "${container_platform_immutability_output}" == *"existing platform tag config"*"differs from prepared image"* ]]
! grep -Fq 'PUSH|' "${container_docker_log}"
! grep -Fq 'CREATE|' "${container_docker_log}"
printf '%s|%s\n' 'sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc' "${container_image_id}" >"${container_platform_state}"

printf '%s|%s\n' "${bad_platform_digest}" "${bad_config_digest}" >"${container_version_state}"
: >"${container_docker_log}"
set +e
container_version_immutability_output="$(
  cd "${container_repository}"
  PATH="${container_bin}:${PATH}" CONTAINER_DOCKER_LOG="${container_docker_log}" CONTAINER_DOCKER_STATE="${container_docker_state}" CONTAINER_EXPECTED_IMAGE_ID="${container_image_id}" PUBLISH_PLATFORMS="linux/amd64" \
    ./scripts/release/publish_container_artifacts.sh --preflight-only 2>&1
)"
container_version_immutability_status=$?
set -e
[[ "${container_version_immutability_status}" -ne 0 ]]
[[ "${container_version_immutability_output}" == *"existing version config"*"differs from prepared image"* ]]
! grep -Fq 'PUSH|' "${container_docker_log}"
! grep -Fq 'CREATE|' "${container_docker_log}"
printf '%s|%s\n' 'sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc' "${container_image_id}" >"${container_version_state}"

rm -f "${container_platform_state}"
: >"${container_docker_log}"
set +e
container_missing_platform_output="$(
  cd "${container_repository}"
  PATH="${container_bin}:${PATH}" CONTAINER_DOCKER_LOG="${container_docker_log}" CONTAINER_DOCKER_STATE="${container_docker_state}" CONTAINER_EXPECTED_IMAGE_ID="${container_image_id}" PUBLISH_PLATFORMS="linux/amd64" \
    ./scripts/release/publish_container_artifacts.sh --preflight-only 2>&1
)"
container_missing_platform_status=$?
set -e
[[ "${container_missing_platform_status}" -ne 0 ]]
[[ "${container_missing_platform_output}" == *"immutable version ghcr.io/tyemirov/loopaware:v1.2.3 exists but its versioned linux/amd64 tag is missing"* ]]
! grep -Fq 'PUSH|' "${container_docker_log}"
! grep -Fq 'CREATE|' "${container_docker_log}"

echo "publish preflight contract checks passed"
