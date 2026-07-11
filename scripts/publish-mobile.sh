#!/usr/bin/env bash
set -euo pipefail

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
readarray -t release_values < <(python3 - "${manifest_path}" <<'PY'
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
    raise SystemExit(f"prepared release is missing mobile payloads: {', '.join(missing)}")
artifact_root = pathlib.Path(sys.argv[1]).parent
print(timestamp)
for name in required:
    print(artifact_root / payloads[name])
PY
)
release_timestamp="${release_values[0]}"
ios_ipa="${release_values[1]}"
ios_manifest="${release_values[2]}"
android_aab="${release_values[3]}"
android_manifest="${release_values[4]}"
android_mapping="${release_values[5]}"

set -a
# shellcheck disable=SC1090
source "${env_file}"
set +a

echo "==> [publish] Uploading prepared LoopAware mobile artifacts"
make --no-print-directory submit-ios \
  MOBILE_RELEASE_TIMESTAMP="${release_timestamp}" \
  MOBILE_IOS_SUBMIT_ARGS="--manifest ${ios_manifest} --ipa ${ios_ipa} ${MOBILE_IOS_SUBMIT_ARGS:-}"
make --no-print-directory submit-android \
  MOBILE_RELEASE_TIMESTAMP="${release_timestamp}" \
  MOBILE_ANDROID_PUBLISH_ARGS="--aab ${android_aab} --build-manifest ${android_manifest} --mapping ${android_mapping} ${MOBILE_ANDROID_PUBLISH_ARGS:-}"
