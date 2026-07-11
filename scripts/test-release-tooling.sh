#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
release_tool_dir="${repo_root}/scripts/release"
release_version="v9.8.7"
pages_url="https://pages.example.invalid"
required_tools=(
  prepare_release.sh
  publish_release.sh
  release_helper.py
  prepare_pages_artifact.sh
  deploy_pages_artifact.sh
  prepare_container_artifact.sh
  publish_container_artifacts.sh
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
grep -Fq 'RELEASE_TOOL_DIR := $(abspath $(CURDIR)/scripts/release)' "${repo_root}/Makefile"

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
cp "${site_directory}/index.html" "${source_repository}/index.html"
git -C "${source_repository}" add index.html
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
"${release_tool_dir}/prepare_pages_artifact.sh" --source "${site_directory}"

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

deploy_output="$(
  cd "${source_repository}"
  PATH="${fake_bin}:${PATH}" \
  FAKE_RELEASE_DIR="${artifact_directory}" \
  FAKE_PAGES_MARKER="${public_marker}" \
  PAGES_VERIFY_ATTEMPTS=1 \
  PAGES_VERIFY_DELAY_SECONDS=0 \
  "${release_tool_dir}/deploy_pages_artifact.sh" \
    --version "${release_version}" \
    --url "${pages_url}" \
    --skip-configure
)"
[[ "${deploy_output}" == *"Verified ${pages_url} at source ${source_commit}."* ]]
[[ "${deploy_output}" != *"at source ${release_commit}."* ]]
git --git-dir="${remote_repository}" cat-file -e refs/heads/gh-pages:.nojekyll

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
  "${release_tool_dir}/deploy_pages_artifact.sh" \
    --version "${release_version}" \
    --url "${pages_url}" \
    --skip-configure 2>&1
)"
rejected_status=$?
set -e
[[ "${rejected_status}" -ne 0 ]]
[[ "${rejected_output}" == *"Pages marker did not reach source ${source_commit}"* ]]
