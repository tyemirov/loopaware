#!/usr/bin/env bash
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel)"
cd "${repo_root}"
manifest_path="$(git rev-parse --git-path mprlab-release)/manifest.json"
if [[ "${manifest_path}" != /* ]]; then
  manifest_path="${repo_root}/${manifest_path}"
fi
[[ -f "${manifest_path}" ]] || { echo "error: prepared release manifest is missing; run make release" >&2; exit 1; }
expected_manifest_sha256="$(shasum -a 256 "${manifest_path}" | awk '{print $1}')"

assert_manifest_unchanged() {
  [[ -f "${manifest_path}" ]] || { echo "error: prepared release manifest disappeared during publication preflight" >&2; exit 1; }
  local actual_manifest_sha256
  actual_manifest_sha256="$(shasum -a 256 "${manifest_path}" | awk '{print $1}')"
  [[ "${actual_manifest_sha256}" == "${expected_manifest_sha256}" ]] || {
    echo "error: prepared release manifest changed during publication preflight" >&2
    exit 1
  }
}

run_preflight_stage() {
  local label="$1"
  shift
  assert_manifest_unchanged
  echo "==> [publish-preflight] ${label}"
  "$@"
  assert_manifest_unchanged
}

if [[ -n "${PUBLISH_RELEASE_ARGS:-}" ]]; then
  echo "error: make publish does not accept PUBLISH_RELEASE_ARGS; use the explicit publish-release target for noncanonical operations" >&2
  exit 1
fi
if [[ -n "${MOBILE_IOS_SUBMIT_ARGS:-}" || -n "${MOBILE_ANDROID_PUBLISH_ARGS:-}" || -n "${CLIENT_REACT_NATIVE_PUBLISH_ARGS:-}" ]]; then
  echo "error: make publish does not accept publication argument overrides; canonical prepared artifacts and destinations are mandatory" >&2
  exit 1
fi

run_preflight_stage "Validating GitHub release publication state" ./scripts/publish-release.sh --dry-run
github_permission="$(gh repo view tyemirov/loopaware --json viewerPermission --jq .viewerPermission)"
case "${github_permission}" in
  ADMIN|MAINTAIN|WRITE) ;;
  *) echo "error: GitHub identity lacks repository write permission required for release publication" >&2; exit 1 ;;
esac

run_preflight_stage \
  "Validating container artifacts and registry authentication" \
  env PUBLISH_PLATFORMS="${PUBLISH_PLATFORMS:-linux/amd64}" \
  ./scripts/release/publish_container_artifacts.sh --preflight-only

run_preflight_stage \
  "Validating mobile artifacts and store authority" \
  env RELEASE_ENV_FILE="${RELEASE_ENV_FILE:-${repo_root}/configs/.env.loopaware}" \
  ./scripts/publish-mobile.sh --preflight-only

run_preflight_stage \
  "Validating React Native package and npm write authority" \
  env RELEASE_ENV_FILE="${RELEASE_ENV_FILE:-${repo_root}/configs/.env.loopaware}" \
  ./scripts/publish-react-native.sh --preflight-only

echo "Publication preflight passed; no release asset, image, store build, or npm package was published."
