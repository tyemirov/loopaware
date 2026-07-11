#!/usr/bin/env bash
set -euo pipefail

verify_only="false"
while [[ $# -gt 0 ]]; do
  case "$1" in
    --verify-only) verify_only="true"; shift ;;
    --help|-h)
      echo "Usage: record_publication.sh [--verify-only]"
      exit 0
      ;;
    *) echo "error: unknown argument: $1" >&2; exit 1 ;;
  esac
done

repo_root="$(git rev-parse --show-toplevel)"
cd "${repo_root}"
source "${repo_root}/scripts/release/repository_identity.sh"
assert_no_github_repository_override
assert_canonical_github_origin "${repo_root}" LoopAware "tyemirov/loopaware"

artifact_directory="$(git rev-parse --git-path mprlab-release)"
if [[ "${artifact_directory}" != /* ]]; then
  artifact_directory="${repo_root}/${artifact_directory}"
fi
manifest_path="${artifact_directory}/manifest.json"
publication_path="${artifact_directory}/publication.json"
[[ -f "${manifest_path}" ]] || { echo "error: prepared release manifest is missing" >&2; exit 1; }
actual_manifest_sha256="$(shasum -a 256 "${manifest_path}" | awk '{print $1}')"
[[ -n "${LOOPAWARE_RELEASE_MANIFEST_SHA256:-}" && "${LOOPAWARE_RELEASE_MANIFEST_SHA256}" == "${actual_manifest_sha256}" ]] || {
  echo "error: publication attestation requires the pinned prepared-release manifest digest" >&2
  exit 1
}
"${repo_root}/scripts/release/release_helper.py" verify-release-artifact >/dev/null

temporary_directory="$(mktemp -d)"
trap 'rm -rf "${temporary_directory}"' EXIT
expected_path="${temporary_directory}/publication.json"
python3 - "${manifest_path}" "${expected_path}" <<'PY_ATTESTATION'
import hashlib
import json
import pathlib
import re
import sys

manifest_path = pathlib.Path(sys.argv[1])
manifest = json.loads(manifest_path.read_text(encoding="utf-8"))
version = manifest.get("version")
if manifest.get("schema_version") != 2 or manifest.get("artifact_kind") != "mprlab.release":
    raise SystemExit("prepared release manifest has an invalid contract")
if not isinstance(version, str) or not re.fullmatch(r"v(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)", version):
    raise SystemExit("prepared release version is invalid")
attestation = {
    "schema": "mprlab.loopaware-publication.v1",
    "status": "complete",
    "version": version,
    "source_commit": manifest["source_commit"],
    "release_commit": manifest["release_commit"],
    "release_manifest_sha256": hashlib.sha256(manifest_path.read_bytes()).hexdigest(),
    "completed_stages": [
        "github-release",
        "ghcr",
        "npm",
        "app-store-connect-upload-accepted",
        "google-play",
    ],
}
pathlib.Path(sys.argv[2]).write_text(json.dumps(attestation, indent=2, sort_keys=True) + "\n", encoding="utf-8")
PY_ATTESTATION

if [[ -f "${publication_path}" ]]; then
  cmp "${expected_path}" "${publication_path}" >/dev/null || {
    echo "error: local publication attestation differs from the prepared release" >&2
    exit 1
  }
else
  [[ "${verify_only}" == "false" ]] || {
    echo "error: complete-publication attestation is missing; make publish did not complete every provider stage" >&2
    exit 1
  }
  mv "${expected_path}" "${publication_path}"
fi

assets_json="$(gh release view "$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["version"])' "${manifest_path}")" --repo tyemirov/loopaware --json assets)"
publication_asset_count="$(python3 - "${assets_json}" <<'PY_ASSETS'
import json
import sys

payload = json.loads(sys.argv[1])
print(sum(1 for asset in payload.get("assets", []) if asset.get("name") == "publication.json"))
PY_ASSETS
)"
[[ "${publication_asset_count}" == "0" || "${publication_asset_count}" == "1" ]] || {
  echo "error: GitHub Release contains duplicate publication attestations" >&2
  exit 1
}
version="$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["version"])' "${manifest_path}")"
if [[ "${publication_asset_count}" == "0" ]]; then
  [[ "${verify_only}" == "false" ]] || {
    echo "error: complete-publication attestation is not published for ${version}" >&2
    exit 1
  }
  gh release upload "${version}" "${publication_path}" --repo tyemirov/loopaware
fi

download_directory="${temporary_directory}/download"
mkdir -p "${download_directory}"
gh release download "${version}" --repo tyemirov/loopaware --pattern publication.json --dir "${download_directory}"
cmp "${publication_path}" "${download_directory}/publication.json" >/dev/null || {
  echo "error: published completion attestation differs from the local release" >&2
  exit 1
}

if [[ "${verify_only}" == "true" ]]; then
  echo "Verified complete publication for ${version}."
else
  echo "Recorded complete publication for ${version}."
fi
