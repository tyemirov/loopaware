#!/usr/bin/env bash
set -euo pipefail

[[ -z "${NODE_OPTIONS:-}" && -z "${NODE_PATH:-}" ]] || {
  echo "error: NODE_OPTIONS and NODE_PATH are not supported by mobile publication" >&2
  exit 1
}

usage() {
  cat <<'USAGE'
Usage:
  publish-mobile.sh [--preflight-only]

Options:
  --preflight-only  Validate exact prepared artifacts and store authority without uploading
USAGE
}

preflight_only="false"
while [[ $# -gt 0 ]]; do
  case "$1" in
    --preflight-only) preflight_only="true"; shift ;;
    --help|-h) usage; exit 0 ;;
    *) echo "error: unknown argument: $1" >&2; usage; exit 1 ;;
  esac
done

repo_root="$(git rev-parse --show-toplevel)"
cd "${repo_root}"

if [[ -v RELEASE_ENV_FILE ]] && [[ -n "${RELEASE_ENV_FILE}" ]]; then
  env_file="${RELEASE_ENV_FILE}"
else
  env_file="${repo_root}/configs/.env.loopaware"
fi
if [[ "${env_file}" != /* ]]; then
  env_file="${repo_root}/${env_file}"
fi
[[ -f "${env_file}" ]] || { echo "error: mobile publish env file not found: ${env_file}" >&2; exit 1; }

manifest_path="$(git rev-parse --git-path mprlab-release)/manifest.json"
[[ -f "${manifest_path}" ]] || { echo "error: prepared release manifest is missing; run make release" >&2; exit 1; }
helper="${repo_root}/scripts/release/release_helper.py"
[[ -x "${helper}" ]] || { echo "error: release helper not found: ${helper}" >&2; exit 1; }
"${helper}" verify-release-artifact >/dev/null
release_values_file="$(mktemp)"
trap 'rm -f "${release_values_file}"' EXIT
python3 -c '
import json
import pathlib
import sys
with open(sys.argv[1], "r", encoding="utf-8") as handle:
    manifest = json.load(handle)
timestamp = manifest.get("release_timestamp")
if not isinstance(timestamp, str) or not timestamp:
    raise SystemExit("prepared release manifest has no release_timestamp")
required = [
    "loopaware-ios.ipa",
    "loopaware-ios.json",
    "loopaware-android.aab",
    "loopaware-android.json",
    "loopaware-android-mapping.txt",
]
payloads = {pathlib.Path(entry["path"]).name: entry["path"] for entry in manifest.get("payloads", [])}
missing = [name for name in required if name not in payloads]
if missing:
    raise SystemExit("prepared release is missing mobile payloads: " + ", ".join(missing))
artifact_root = pathlib.Path(sys.argv[1]).parent
print(timestamp)
for name in required:
    print(artifact_root / payloads[name])
' "${manifest_path}" >"${release_values_file}"
release_values=()
while IFS= read -r value; do
  release_values+=("${value}")
done <"${release_values_file}"
rm -f "${release_values_file}"
[[ "${#release_values[@]}" -eq 6 ]] || { echo "error: prepared mobile release manifest returned incomplete values" >&2; exit 1; }
release_timestamp="${release_values[0]}"
ios_ipa="${release_values[1]}"
ios_manifest="${release_values[2]}"
android_aab="${release_values[3]}"
android_manifest="${release_values[4]}"
android_mapping="${release_values[5]}"

python3 - "${release_timestamp}" "${ios_manifest}" "${android_manifest}" <<'PY_MOBILE_IDENTITY'
import datetime as dt
import json
import sys

outer_timestamp = dt.datetime.fromisoformat(sys.argv[1].replace("Z", "+00:00"))
manifests = [json.load(open(path, encoding="utf-8")) for path in sys.argv[2:]]
versioning = [manifest.get("versioning") for manifest in manifests]
if not all(isinstance(value, dict) for value in versioning) or versioning[0] != versioning[1]:
    raise SystemExit("prepared mobile artifacts do not share one versioning identity")
artifact_timestamp = dt.datetime.fromisoformat(str(versioning[0].get("releaseTimestamp", "")).replace("Z", "+00:00"))
if artifact_timestamp != outer_timestamp:
    raise SystemExit("prepared mobile artifact timestamp does not match the outer release manifest")
PY_MOBILE_IDENTITY

source "${repo_root}/scripts/release/load_release_env.sh"
load_release_env_file "${env_file}"

: "${APP_STORE_CONNECT_API_KEY_ID:=82P4KZ86HM}"
: "${APP_STORE_CONNECT_API_ISSUER_ID:=94ecd239-946c-478c-8fe5-5c7f50816959}"
: "${APP_STORE_CONNECT_API_KEY_PATH:=${repo_root}/configs/AuthKey_82P4KZ86HM.p8}"
export APP_STORE_CONNECT_API_KEY_ID APP_STORE_CONNECT_API_ISSUER_ID APP_STORE_CONNECT_API_KEY_PATH

[[ -z "${MOBILE_IOS_SUBMIT_ARGS:-}" && -z "${MOBILE_ANDROID_PUBLISH_ARGS:-}" ]] || {
  echo "error: prepared mobile publication does not accept argument overrides" >&2
  exit 1
}

command -v node >/dev/null 2>&1 || { echo "error: node is required for mobile publication" >&2; exit 1; }
make --no-print-directory mobile-check
ios_submit_script="${repo_root}/mobile/scripts/submit-ios.mjs"
android_publish_script="${repo_root}/mobile/scripts/publish-android-play.mjs"

preflight_mobile_publication() {
  echo "==> [publish-preflight] Validating prepared LoopAware mobile artifacts and store authority"
  MOBILE_RELEASE_TIMESTAMP="${release_timestamp}" node "${ios_submit_script}" \
    --preflight-only \
    --manifest "${ios_manifest}" \
    --ipa "${ios_ipa}"
  MOBILE_RELEASE_TIMESTAMP="${release_timestamp}" node "${ios_submit_script}" \
    --dry-run \
    --manifest "${ios_manifest}" \
    --ipa "${ios_ipa}"
  MOBILE_RELEASE_TIMESTAMP="${release_timestamp}" node "${android_publish_script}" \
    --dry-run \
    --aab "${android_aab}" \
    --build-manifest "${android_manifest}" \
    --mapping "${android_mapping}"
}

preflight_mobile_publication
if [[ "${preflight_only}" == "true" ]]; then
  echo "Mobile publication preflight passed; the transient empty Play edit was deleted and no store artifact was uploaded."
  exit 0
fi

echo "==> [publish] Uploading prepared LoopAware mobile artifacts"
set +e
MOBILE_RELEASE_TIMESTAMP="${release_timestamp}" node "${ios_submit_script}" \
  --manifest "${ios_manifest}" \
  --ipa "${ios_ipa}"
ios_publish_status=$?
set -e
if [[ "${ios_publish_status}" -ne 0 ]]; then
  echo "error: the iOS upload outcome is unknown; App Store Connect may have accepted the single-use build identity, so inspect it before preparing a new release timestamp and do not blindly retry" >&2
  exit "${ios_publish_status}"
fi
set +e
MOBILE_RELEASE_TIMESTAMP="${release_timestamp}" node "${android_publish_script}" \
  --aab "${android_aab}" \
  --build-manifest "${android_manifest}" \
  --mapping "${android_mapping}"
android_publish_status=$?
set -e
if [[ "${android_publish_status}" -ne 0 ]]; then
  echo "error: the iOS build was accepted before Android failed; store build identifiers are single-use, so inspect both providers and prepare a new release timestamp instead of blindly retrying this mobile publication" >&2
  exit "${android_publish_status}"
fi
