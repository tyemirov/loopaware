#!/usr/bin/env bash
set -euo pipefail

[[ $# -eq 0 ]] || { echo "error: canonical publish accepts no arguments" >&2; exit 2; }

repo_root="$(git rev-parse --show-toplevel)"
cd "${repo_root}"
manifest_path="$(git rev-parse --git-path mprlab-release)/manifest.json"
if [[ "${manifest_path}" != /* ]]; then
  manifest_path="${repo_root}/${manifest_path}"
fi
[[ -f "${manifest_path}" ]] || { echo "error: prepared release manifest is missing; run make release" >&2; exit 1; }
expected_manifest_sha256="$(shasum -a 256 "${manifest_path}" | awk '{print $1}')"
[[ "${expected_manifest_sha256}" =~ ^[0-9a-f]{64}$ ]] || { echo "error: prepared release manifest digest is invalid" >&2; exit 1; }

release_identity="$(python3 - "${manifest_path}" <<'PY'
import json
import re
import sys

manifest = json.load(open(sys.argv[1], encoding="utf-8"))
version = manifest.get("version")
if manifest.get("schema_version") != 2 or manifest.get("artifact_kind") != "mprlab.release":
    raise SystemExit("prepared release manifest has an invalid contract")
if not isinstance(version, str) or not re.fullmatch(r"v(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)", version):
    raise SystemExit("prepared release version must be stable vMAJOR.MINOR.PATCH")
for field in ("source_commit", "release_commit"):
    value = manifest.get(field)
    if not isinstance(value, str) or not re.fullmatch(r"[0-9a-f]{40}", value):
        raise SystemExit(f"prepared release manifest has invalid {field}")
print("\t".join((version, manifest["source_commit"], manifest["release_commit"])))
PY
)"
IFS=$'\t' read -r release_version release_source_commit release_commit <<<"${release_identity}"
[[ -n "${release_version}" && -n "${release_source_commit}" && -n "${release_commit}" ]] || {
  echo "error: prepared release identity is incomplete" >&2
  exit 1
}
export LOOPAWARE_RELEASE_MANIFEST_SHA256="${expected_manifest_sha256}"
export LOOPAWARE_RELEASE_VERSION="${release_version}"
export LOOPAWARE_RELEASE_SOURCE_COMMIT="${release_source_commit}"
export LOOPAWARE_RELEASE_COMMIT="${release_commit}"

assert_manifest_unchanged() {
  [[ -f "${manifest_path}" ]] || { echo "error: prepared release manifest disappeared during publication" >&2; exit 1; }
  local actual_manifest_sha256
  actual_manifest_sha256="$(shasum -a 256 "${manifest_path}" | awk '{print $1}')"
  [[ "${actual_manifest_sha256}" == "${expected_manifest_sha256}" ]] || {
    echo "error: prepared release manifest changed during publication; refusing to mix release identities" >&2
    exit 1
  }
}

run_stage() {
  local label="$1"
  shift
  assert_manifest_unchanged
  echo "==> [publish] ${label}"
  "$@"
  assert_manifest_unchanged
}

publication_attestation_path="$(dirname "${manifest_path}")/publication.json"
if [[ -f "${publication_attestation_path}" ]]; then
  run_stage "Verifying the existing complete-publication attestation" ./scripts/release/record_publication.sh --verify-only
  echo "LoopAware publication was already completed for ${LOOPAWARE_RELEASE_VERSION}; no provider upload was repeated."
  exit 0
fi

run_stage "Running complete publication preflight" ./scripts/publish-preflight.sh
run_stage "Publishing Git refs, GitHub Release metadata, and release assets" ./scripts/publish-release.sh
run_stage "Publishing the immutable container release" ./scripts/release/publish_container_artifacts.sh
run_stage "Publishing the React Native package" ./scripts/publish-react-native.sh
run_stage "Publishing mobile store builds" ./scripts/publish-mobile.sh
run_stage "Recording complete publication" ./scripts/release/record_publication.sh

echo "LoopAware publication complete for ${LOOPAWARE_RELEASE_VERSION}."
