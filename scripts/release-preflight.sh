#!/usr/bin/env bash
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel)"
cd "${repo_root}"

env_file="${RELEASE_ENV_FILE:-${repo_root}/configs/.env.loopaware}"
if [[ "${env_file}" != /* ]]; then
  env_file="${repo_root}/${env_file}"
fi
[[ -f "${env_file}" ]] || { echo "error: release env file not found: ${env_file}" >&2; exit 1; }
source "${repo_root}/scripts/release/load_release_env.sh"
load_release_env_file "${env_file}"

selection_file="$(mktemp)"
artifact_directory=""
cleanup() {
  rm -f "${selection_file}"
  if [[ -n "${artifact_directory}" ]]; then
    rm -rf "${artifact_directory}"
  fi
}
trap cleanup EXIT

for argument in "$@"; do
  case "${argument}" in
    --dry-run|--help|-h) echo "error: release lifecycle control flags are owned by release-preflight.sh" >&2; exit 1 ;;
  esac
done

echo "==> [release-preflight] Validating clean release state and version selection"
: >"${selection_file}"
if ! RELEASE_ENV_FILE="${env_file}" ./scripts/release.sh --dry-run "$@" >"${selection_file}"; then
  cat "${selection_file}"
  exit 1
fi
cat "${selection_file}"
if grep -Fxq "release_already_prepared=true" "${selection_file}"; then
  prepared_version="$(sed -n 's/^version=//p' "${selection_file}")"
  [[ -n "${prepared_version}" ]] || { echo "error: prepared release preflight returned no version" >&2; exit 1; }
  echo "Release preflight passed for already prepared ${prepared_version}; no release state or provider was changed."
  exit 0
fi
next_version="$(sed -n 's/^next_version=//p' "${selection_file}")"
[[ -n "${next_version}" ]] || { echo "error: release preflight did not select a version" >&2; exit 1; }
source_commit="$(sed -n 's/^source_commit=//p' "${selection_file}")"
[[ "${source_commit}" =~ ^[0-9a-f]{40}$ ]] || { echo "error: release preflight did not select a valid source commit" >&2; exit 1; }
release_head_commit="$(git rev-parse HEAD)"
release_commit_reuse="$(sed -n 's/^release_commit_reuse=//p' "${selection_file}")"
[[ "${release_commit_reuse}" == "true" || "${release_commit_reuse}" == "false" ]] || {
  echo "error: release preflight did not report release-commit reuse state" >&2
  exit 1
}

required_artifact_targets="mobile-release-artifacts client-react-native-artifact container-artifacts pages-artifact"
[[ "${RELEASE_ARTIFACT_TARGETS:-}" == "${required_artifact_targets}" ]] || {
  echo "error: release preflight requires the canonical artifact target set: ${required_artifact_targets}" >&2
  exit 1
}

echo "==> [release-preflight] Running the release CI gate"
make ci
: >"${selection_file}"
if ! RELEASE_ENV_FILE="${env_file}" ./scripts/release.sh --dry-run "$@" >"${selection_file}"; then
  cat "${selection_file}"
  exit 1
fi
[[ "$(git rev-parse HEAD)" == "${release_head_commit}" ]] || { echo "error: HEAD changed while release preflight CI was running" >&2; exit 1; }
[[ "$(sed -n 's/^next_version=//p' "${selection_file}")" == "${next_version}" ]] || { echo "error: release version selection changed after CI" >&2; exit 1; }
[[ "$(sed -n 's/^source_commit=//p' "${selection_file}")" == "${source_commit}" ]] || { echo "error: release source commit changed after CI" >&2; exit 1; }
[[ "$(sed -n 's/^release_commit_reuse=//p' "${selection_file}")" == "${release_commit_reuse}" ]] || { echo "error: release-commit reuse state changed after CI" >&2; exit 1; }

echo "==> [release-preflight] Building disposable release artifacts"
artifact_directory="$(mktemp -d)"
release_timestamp="$(date +%Y-%m-%dT%H:%M:%S%z)"
./scripts/release/release_helper.py initialize-release-artifact \
  --version "${next_version}" \
  --source-commit "${source_commit}" \
  --release-timestamp "${release_timestamp}" \
  --artifact-dir "${artifact_directory}" >/dev/null
artifact_targets="${RELEASE_ARTIFACT_TARGETS:-mobile-release-artifacts client-react-native-artifact container-artifacts pages-artifact}"
read -r -a artifact_target_list <<<"${artifact_targets}"
make --no-print-directory \
  RELEASE_VERSION="${next_version}" \
  RELEASE_SOURCE_COMMIT="${source_commit}" \
  RELEASE_TIMESTAMP="${release_timestamp}" \
  MOBILE_RELEASE_TIMESTAMP="${release_timestamp}" \
  RELEASE_ARTIFACT_DIR="${artifact_directory}" \
  "${artifact_target_list[@]}"
./scripts/release/verify_staged_artifacts.py \
  --artifact-dir "${artifact_directory}" \
  --repo-root "${repo_root}" \
  --version "${next_version}" \
  --source-commit "${source_commit}"
payload_count="$(find "${artifact_directory}/payloads" -type f | wc -l | tr -d ' ')"
[[ "${payload_count}" == "9" ]] || { echo "error: release preflight did not produce the exact nine canonical payloads" >&2; exit 1; }
: >"${selection_file}"
if ! RELEASE_ENV_FILE="${env_file}" ./scripts/release.sh --dry-run "$@" >"${selection_file}"; then
  cat "${selection_file}"
  exit 1
fi
[[ "$(git rev-parse HEAD)" == "${release_head_commit}" ]] || { echo "error: HEAD changed while release preflight artifacts were built" >&2; exit 1; }
[[ "$(sed -n 's/^next_version=//p' "${selection_file}")" == "${next_version}" ]] || { echo "error: release version selection changed after artifact build" >&2; exit 1; }
[[ "$(sed -n 's/^source_commit=//p' "${selection_file}")" == "${source_commit}" ]] || { echo "error: release source commit changed after artifact build" >&2; exit 1; }
[[ "$(sed -n 's/^release_commit_reuse=//p' "${selection_file}")" == "${release_commit_reuse}" ]] || { echo "error: release-commit reuse state changed after artifact build" >&2; exit 1; }

echo "Release preflight passed with ${payload_count} disposable payloads; no changelog, commit, tag, publication, or production service was changed."
