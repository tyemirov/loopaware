#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
release_tool_dir="${repo_root}/scripts/release"
release_version="v9.8.7"
pages_url="https://pages.example.invalid"
pages_domain="pages.example.invalid"
required_tools=(
  prepare_release.sh
  publish_release.sh
  release_helper.py
  prepare_pages_artifact.sh
  deploy_pages_artifact.sh
  prepare_container_artifact.sh
  publish_container_artifacts.sh
  docker_identity.sh
  verify_staged_artifacts.py
  run_lifecycle.sh
  record_publication.sh
  parse_release_env.py
)

for tool in "${required_tools[@]}"; do
  [[ -x "${release_tool_dir}/${tool}" ]] || {
    echo "error: repository-owned release tool is missing or not executable: ${tool}" >&2
    exit 1
  }
done
if grep -F 'agentSkills/gitrelease/scripts' \
  "${repo_root}/Makefile" \
  "${repo_root}/scripts/release.sh" \
  "${repo_root}/scripts/publish-release.sh" \
  "${repo_root}/scripts/publish-react-native.sh"; then
  echo "error: mutable sibling release tooling remains referenced" >&2
  exit 1
fi
grep -Fq 'override RELEASE_TOOL_DIR := $(abspath $(CURDIR)/scripts/release)' "${repo_root}/Makefile"
if grep -Fq -- '--clobber' "${release_tool_dir}/release_helper.py"; then
  echo "error: GitHub Release publication still permits asset replacement" >&2
  exit 1
fi

temporary_directory="$(mktemp -d)"
trap 'rm -rf "${temporary_directory}"' EXIT
source_repository="${temporary_directory}/source"
remote_repository="${temporary_directory}/remote.git"
artifact_directory="${source_repository}/.git/mprlab-release"
site_directory="${temporary_directory}/site"
fake_bin="${temporary_directory}/bin"
mkdir -p "${source_repository}" "${site_directory}" "${fake_bin}"

git init --bare "${remote_repository}" >/dev/null
git -C "${source_repository}" init -b master >/dev/null
git -C "${source_repository}" config user.name "Release Contract"
git -C "${source_repository}" config user.email "release-contract@mprlab.invalid"
printf '<!doctype html><title>Fixture</title>\n' >"${site_directory}/index.html"
mkdir -p "${source_repository}/web"
cp "${site_directory}/index.html" "${source_repository}/web/index.html"
git -C "${source_repository}" add web/index.html
git -C "${source_repository}" commit -m "Add Pages source" >/dev/null
source_commit="$(git -C "${source_repository}" rev-parse HEAD)"
git -C "${source_repository}" remote add origin "${remote_repository}"
git -C "${source_repository}" push -u origin master >/dev/null

mkdir -p "${artifact_directory}"
cat >"${artifact_directory}/staging.json" <<JSON
{
  "release_timestamp": "2026-07-09T12:00:00-07:00",
  "source_commit": "${source_commit}",
  "version": "${release_version}"
}
JSON
cat >"${fake_bin}/rsync" <<'EOF_RSYNC'
#!/bin/sh
set -eu
while [ "$#" -gt 2 ]; do shift; done
cp -R "$1"/. "$2"/
EOF_RSYNC
chmod +x "${fake_bin}/rsync"
PATH="${fake_bin}:${PATH}" \
RELEASE_VERSION="${release_version}" \
RELEASE_ARTIFACT_DIR="${artifact_directory}" \
"${release_tool_dir}/prepare_pages_artifact.sh" --source "${site_directory}" --domain "${pages_domain}"

archive="${artifact_directory}/payloads/release-assets/pages.tar.gz"
extracted_site="${temporary_directory}/extracted"
mkdir -p "${extracted_site}"
tar -xzf "${archive}" -C "${extracted_site}"
[[ -f "${extracted_site}/.nojekyll" ]]
[[ "$(wc -c <"${extracted_site}/.nojekyll" | tr -d ' ')" == "0" ]]
python3 - "${extracted_site}/.mprlab-release.json" "${release_version}" "${source_commit}" <<'PY'
import json
import sys

marker = json.load(open(sys.argv[1], encoding="utf-8"))
if marker != {
    "release_timestamp": "2026-07-09T12:00:00-07:00",
    "release_version": sys.argv[2],
    "schema_version": 1,
    "source_commit": sys.argv[3],
}:
    raise SystemExit("prepared Pages marker does not match source provenance")
PY

printf 'release\n' >"${source_repository}/CHANGELOG.md"
git -C "${source_repository}" add CHANGELOG.md
git -C "${source_repository}" commit -m "Release ${release_version}" >/dev/null
release_commit="$(git -C "${source_repository}" rev-parse HEAD)"
[[ "${source_commit}" != "${release_commit}" ]]
git -C "${source_repository}" tag -a "${release_version}" -m "Release ${release_version}"
git -C "${source_repository}" push origin master "refs/tags/${release_version}" >/dev/null
archive_sha256="$(shasum -a 256 "${archive}" | awk '{print $1}')"
python3 - "${artifact_directory}/manifest.json" "${archive}" "${release_version}" "${release_commit}" "${source_commit}" "${archive_sha256}" <<'PY'
import json
import os
import pathlib
import sys

manifest_path, archive_path, version, release_commit, source_commit, archive_sha256 = sys.argv[1:]
manifest = {
    "artifact_kind": "mprlab.release",
    "default_branch": "master",
    "notes_sha256": "0" * 64,
    "payloads": [{
        "path": "payloads/release-assets/pages.tar.gz",
        "sha256": archive_sha256,
        "size": os.path.getsize(archive_path),
    }],
    "release_commit": release_commit,
    "release_timestamp": "2026-07-09T12:00:00-07:00",
    "schema_version": 2,
    "source_commit": source_commit,
    "version": version,
}
pathlib.Path(manifest_path).write_text(json.dumps(manifest, indent=2, sort_keys=True) + "\n", encoding="utf-8")
PY

public_marker="${temporary_directory}/public-marker.json"
cp "${extracted_site}/.mprlab-release.json" "${public_marker}"
cat >"${fake_bin}/gh" <<'EOF_GH'
#!/bin/sh
set -eu
destination=''
while [ "$#" -gt 0 ]; do
  if [ "$1" = '--dir' ]; then destination="$2"; shift 2; else shift; fi
done
cp "${FAKE_RELEASE_DIR}/manifest.json" "${destination}/manifest.json"
cp "${FAKE_RELEASE_DIR}/payloads/release-assets/pages.tar.gz" "${destination}/pages.tar.gz"
EOF_GH
cat >"${fake_bin}/curl" <<'EOF_CURL'
#!/bin/sh
set -eu
cat "${FAKE_PAGES_MARKER}"
EOF_CURL
chmod +x "${fake_bin}/gh" "${fake_bin}/curl"

if git --git-dir="${remote_repository}" show-ref --verify --quiet refs/heads/gh-pages; then
  echo "error: Pages branch exists before verify-only coverage" >&2
  exit 1
fi
verify_only_output="$(
  cd "${source_repository}"
  PATH="${fake_bin}:${PATH}" \
  FAKE_RELEASE_DIR="${artifact_directory}" \
  timeout 30s "${release_tool_dir}/deploy_pages_artifact.sh" \
    --version "${release_version}" \
    --expected-domain "${pages_domain}" \
    --verify-only
)"
[[ "${verify_only_output}" == *"Verified published Pages artifact ${release_version} at source ${source_commit}."* ]]
if git --git-dir="${remote_repository}" show-ref --verify --quiet refs/heads/gh-pages; then
  echo "error: verify-only mutated the Pages branch" >&2
  exit 1
fi

original_archive="${temporary_directory}/pages-original.tar.gz"
original_manifest="${temporary_directory}/manifest-original.json"
cp "${archive}" "${original_archive}"
cp "${artifact_directory}/manifest.json" "${original_manifest}"
tampered_site="${temporary_directory}/tampered-site"
mkdir -p "${tampered_site}"
tar -xzf "${archive}" -C "${tampered_site}"
printf '<!doctype html><title>Tampered</title>\n' >"${tampered_site}/index.html"
COPYFILE_DISABLE=1 tar -czf "${archive}" -C "${tampered_site}" .
tampered_sha256="$(shasum -a 256 "${archive}" | awk '{print $1}')"
python3 - "${artifact_directory}/manifest.json" "${tampered_sha256}" <<'PY_TAMPERED_MANIFEST'
import json
import pathlib
import sys

path = pathlib.Path(sys.argv[1])
manifest = json.loads(path.read_text(encoding="utf-8"))
manifest["payloads"][0]["sha256"] = sys.argv[2]
path.write_text(json.dumps(manifest, indent=2, sort_keys=True) + "\n", encoding="utf-8")
PY_TAMPERED_MANIFEST
set +e
tampered_output="$(
  cd "${source_repository}"
  PATH="${fake_bin}:${PATH}" \
  FAKE_RELEASE_DIR="${artifact_directory}" \
  timeout 30s "${release_tool_dir}/deploy_pages_artifact.sh" \
    --version "${release_version}" \
    --expected-domain "${pages_domain}" \
    --verify-only 2>&1
)"
tampered_status=$?
set -e
[[ "${tampered_status}" -ne 0 ]]
[[ "${tampered_output}" == *"published Pages archive contents do not match source"* ]]
cp "${original_archive}" "${archive}"
cp "${original_manifest}" "${artifact_directory}/manifest.json"

deploy_output="$(
  cd "${source_repository}"
  PATH="${fake_bin}:${PATH}" \
  FAKE_RELEASE_DIR="${artifact_directory}" \
  FAKE_PAGES_MARKER="${public_marker}" \
  PAGES_VERIFY_ATTEMPTS=1 \
  PAGES_VERIFY_DELAY_SECONDS=0 \
  timeout 30s "${release_tool_dir}/deploy_pages_artifact.sh" \
    --version "${release_version}" \
    --url "${pages_url}" \
    --expected-domain "${pages_domain}" \
    --skip-configure
)"
[[ "${deploy_output}" == *"Verified ${pages_url} at source ${source_commit}."* ]]
[[ "${deploy_output}" != *"at source ${release_commit}."* ]]
git --git-dir="${remote_repository}" cat-file -e refs/heads/gh-pages:.nojekyll
[[ "$(git --git-dir="${remote_repository}" show refs/heads/gh-pages:CNAME)" == "${pages_domain}" ]]

python3 - "${public_marker}" "${release_commit}" <<'PY'
import json
import pathlib
import sys

marker_path = pathlib.Path(sys.argv[1])
marker = json.loads(marker_path.read_text(encoding="utf-8"))
marker["source_commit"] = sys.argv[2]
marker_path.write_text(json.dumps(marker), encoding="utf-8")
PY
set +e
rejected_output="$(
  cd "${source_repository}"
  PATH="${fake_bin}:${PATH}" \
  FAKE_RELEASE_DIR="${artifact_directory}" \
  FAKE_PAGES_MARKER="${public_marker}" \
  PAGES_VERIFY_ATTEMPTS=1 \
  PAGES_VERIFY_DELAY_SECONDS=0 \
  timeout 30s "${release_tool_dir}/deploy_pages_artifact.sh" \
    --version "${release_version}" \
    --url "${pages_url}" \
    --expected-domain "${pages_domain}" \
    --skip-configure 2>&1
)"
rejected_status=$?
set -e
[[ "${rejected_status}" -ne 0 ]]
[[ "${rejected_output}" == *"Pages marker did not reach source ${source_commit}"* ]]

notes_path="${artifact_directory}/notes.md"
printf '## %s - 2026-07-09\n\n- Fixture release notes.\n' "${release_version}" >"${notes_path}"
python3 - "${artifact_directory}/manifest.json" "${notes_path}" <<'PY_NOTES_HASH'
import hashlib
import json
import pathlib
import sys

manifest_path = pathlib.Path(sys.argv[1])
manifest = json.loads(manifest_path.read_text(encoding="utf-8"))
manifest["notes_sha256"] = hashlib.sha256(pathlib.Path(sys.argv[2]).read_bytes()).hexdigest()
manifest_path.write_text(json.dumps(manifest, indent=2, sort_keys=True) + "\n", encoding="utf-8")
PY_NOTES_HASH

github_asset_directory="${temporary_directory}/github-assets"
github_state_file="${temporary_directory}/github-release.state"
github_log="${temporary_directory}/github.log"
mkdir -p "${github_asset_directory}"

cat >"${fake_bin}/gh" <<'EOF_GITHUB_RELEASE'
#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "$*" >>"${FAKE_GH_LOG}"

release_exists() {
  case "${FAKE_GH_MODE}" in
    existing) return 0 ;;
    missing) return 1 ;;
    stateful) [[ -f "${FAKE_GH_STATE_FILE}" ]] ;;
    lookup-error)
      printf '%s\n' 'fixture GitHub authentication failure' >&2
      exit 41
      ;;
    *)
      printf 'unexpected fake GitHub mode: %s\n' "${FAKE_GH_MODE}" >&2
      exit 97
      ;;
  esac
}

if [[ "$1 $2" == "pr list" ]]; then
  printf '%s\n' '[]'
  exit 0
fi

if [[ "$1 $2" == "release view" ]]; then
  if ! release_exists; then
    printf 'release not found: %s\n' "$3" >&2
    exit 1
  fi
  if [[ "$5" == "assets" ]]; then
    python3 - "${FAKE_GH_ASSET_DIR}" <<'PY_ASSET_INVENTORY'
import json
import os
import pathlib
import sys

assets = [{"name": path.name, "size": path.stat().st_size} for path in sorted(pathlib.Path(sys.argv[1]).iterdir()) if path.is_file()]
if os.environ.get("FAKE_GH_UNEXPECTED_ASSET") == "1":
    assets.append({"name": "unexpected.bin", "size": 1})
print(json.dumps({"assets": assets}))
PY_ASSET_INVENTORY
    exit 0
  fi
  python3 - "${FAKE_GH_NOTES}" "${FAKE_GH_VERSION}" <<'PY_RELEASE_METADATA'
import json
import os
import pathlib
import sys

version = sys.argv[2]
name = "Mismatched release title" if os.environ.get("FAKE_GH_METADATA_MISMATCH") == "1" else f"Release {version}"
print(json.dumps({
    "tagName": version,
    "name": name,
    "body": pathlib.Path(sys.argv[1]).read_text(encoding="utf-8"),
    "publishedAt": "2026-07-09T19:00:00Z",
    "isDraft": False,
    "isPrerelease": False,
    "targetCommitish": "master",
    "url": f"https://example.invalid/releases/tag/{version}",
}))
PY_RELEASE_METADATA
  exit 0
fi

if [[ "$1 $2" == "release create" ]]; then
  [[ "${FAKE_GH_MODE}" == "stateful" ]]
  : >"${FAKE_GH_STATE_FILE}"
  exit 0
fi

if [[ "$1 $2" == "release upload" ]]; then
  shift 3
  for asset in "$@"; do
    [[ "${asset}" != "--clobber" ]] || {
      printf '%s\n' 'release asset replacement is forbidden' >&2
      exit 96
    }
    cp "${asset}" "${FAKE_GH_ASSET_DIR}/$(basename "${asset}")"
  done
  exit 0
fi

if [[ "$1 $2" == "release download" ]]; then
  pattern=""
  destination=""
  shift 3
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --pattern) pattern="$2"; shift 2 ;;
      --dir) destination="$2"; shift 2 ;;
      *) printf 'unexpected release download argument: %s\n' "$1" >&2; exit 97 ;;
    esac
  done
  if [[ "${FAKE_GH_TAMPER_ASSET:-}" == "${pattern}" ]]; then
    printf '%s\n' 'tampered remote asset' >"${destination}/${pattern}"
  else
    cp "${FAKE_GH_ASSET_DIR}/${pattern}" "${destination}/${pattern}"
  fi
  exit 0
fi

if [[ "$1 $2" == "run list" ]]; then
  printf '%s\n' '[]'
  exit 0
fi

printf 'unexpected gh command: %s\n' "$*" >&2
exit 97
EOF_GITHUB_RELEASE
chmod +x "${fake_bin}/gh"

run_github_publication() {
  local mode="$1"
  shift
  (
    cd "${source_repository}"
    PATH="${fake_bin}:${PATH}" \
      UV_CACHE_DIR="${source_repository}/.git/uv-cache" \
      FAKE_GH_MODE="${mode}" \
      FAKE_GH_LOG="${github_log}" \
      FAKE_GH_STATE_FILE="${github_state_file}" \
      FAKE_GH_ASSET_DIR="${github_asset_directory}" \
      FAKE_GH_NOTES="${notes_path}" \
      FAKE_GH_VERSION="${release_version}" \
      "${release_tool_dir}/release_helper.py" publish-prepared-release "$@"
  )
}

cp "${artifact_directory}/manifest.json" "${github_asset_directory}/manifest.json"
cp "${archive}" "${github_asset_directory}/pages.tar.gz"
: >"${github_log}"
github_existing_output="$(run_github_publication existing --dry-run)"
[[ "${github_existing_output}" == *'"github_release": "existing"'* ]]
[[ "${github_existing_output}" == *'"upload_release_assets": []'* ]]
[[ "$(<"${github_log}")" != *"release upload"* ]]

: >"${github_log}"
set +e
github_lookup_error_output="$(run_github_publication lookup-error --dry-run 2>&1)"
github_lookup_error_status=$?
set -e
[[ "${github_lookup_error_status}" -ne 0 ]]
[[ "${github_lookup_error_output}" == *"GitHub Release lookup failed"* ]]
[[ "${github_lookup_error_output}" == *"fixture GitHub authentication failure"* ]]
[[ "$(<"${github_log}")" != *"release create"* ]]

: >"${github_log}"
set +e
github_metadata_output="$(FAKE_GH_METADATA_MISMATCH=1 run_github_publication existing --dry-run 2>&1)"
github_metadata_status=$?
set -e
[[ "${github_metadata_status}" -ne 0 ]]
[[ "${github_metadata_output}" == *"existing GitHub Release metadata is immutable"* ]]

: >"${github_log}"
set +e
github_extra_asset_output="$(FAKE_GH_UNEXPECTED_ASSET=1 run_github_publication existing --dry-run 2>&1)"
github_extra_asset_status=$?
set -e
[[ "${github_extra_asset_status}" -ne 0 ]]
[[ "${github_extra_asset_output}" == *"contains noncanonical assets"* ]]

: >"${github_log}"
set +e
github_tampered_asset_output="$(FAKE_GH_TAMPER_ASSET=manifest.json run_github_publication existing --dry-run 2>&1)"
github_tampered_asset_status=$?
set -e
[[ "${github_tampered_asset_status}" -ne 0 ]]
[[ "${github_tampered_asset_output}" == *"asset is immutable and differs"* ]]

rm -f "${github_state_file}" "${github_asset_directory}"/*
: >"${github_log}"
github_publish_output="$(run_github_publication stateful)"
[[ "${github_publish_output}" == *'"published_release_assets"'* ]]
grep -Fq "release create ${release_version}" "${github_log}"
github_upload_command="$(grep "^release upload ${release_version}" "${github_log}")"
[[ "${github_upload_command}" == *"${artifact_directory}/manifest.json"* ]]
[[ "${github_upload_command}" == *"${archive}"* ]]
[[ "${github_upload_command}" != *"--clobber"* ]]
cmp "${artifact_directory}/manifest.json" "${github_asset_directory}/manifest.json"
cmp "${archive}" "${github_asset_directory}/pages.tar.gz"

: >"${github_log}"
github_rerun_output="$(run_github_publication stateful)"
[[ "${github_rerun_output}" == *'"published_release_assets"'* ]]
[[ "$(<"${github_log}")" != *"release upload"* ]]
[[ "$(<"${github_log}")" != *"release create"* ]]

echo "release tooling contract checks passed"
