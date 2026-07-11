#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage:
  prepare_release.sh [options]

Prepares a release entirely from local repository state. The command validates
the worktree, runs make ci, creates the changelog commit and annotated tag, and
writes the release manifest and notes under .git/mprlab-release.

It never fetches, pushes, calls GitHub, publishes an image/store build, updates
GitHub Pages, or deploys production.

Options:
  --bump <patch|minor|major>  SemVer bump when no exact version is supplied. Default: patch
  --version <value>           Exact local release tag/version to prepare
  --dry-run                   Validate and report the selected version without changing files
  --help                      Show this help text
USAGE
}

bump="patch"
version=""
scheme="semver"
dry_run="false"
if [[ -v RELEASE_ARTIFACT_TARGETS ]]; then
  artifact_targets="${RELEASE_ARTIFACT_TARGETS}"
else
  artifact_targets=""
fi
canonical_artifact_targets="mobile-release-artifacts client-react-native-artifact container-artifacts pages-artifact"
[[ "${artifact_targets}" == "${canonical_artifact_targets}" ]] || {
  echo "error: release requires the canonical artifact target set: ${canonical_artifact_targets}" >&2
  exit 1
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --bump)
      [[ $# -ge 2 ]] || { echo "error: --bump requires a value" >&2; exit 1; }
      bump="$2"
      shift 2
      ;;
    --version)
      [[ $# -ge 2 ]] || { echo "error: --version requires a value" >&2; exit 1; }
      version="$2"
      shift 2
      ;;
    --dry-run)
      dry_run="true"
      shift
      ;;
    --help|-h)
      usage
      exit 0
      ;;
    *)
      echo "error: unknown argument: $1" >&2
      usage
      exit 1
      ;;
  esac
done

case "${bump}" in
  patch|minor|major) ;;
  *) echo "error: --bump must be patch, minor, or major" >&2; exit 1 ;;
esac
command -v git >/dev/null 2>&1 || { echo "error: git is required" >&2; exit 1; }
command -v python3 >/dev/null 2>&1 || { echo "error: python3 is required" >&2; exit 1; }

repo_root="$(git rev-parse --show-toplevel)"
cd "${repo_root}"
git var GIT_AUTHOR_IDENT >/dev/null 2>&1 || { echo "error: Git author identity is not configured for the release commit" >&2; exit 1; }
git var GIT_COMMITTER_IDENT >/dev/null 2>&1 || { echo "error: Git committer identity is not configured for the release commit" >&2; exit 1; }

helper="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/release_helper.py"
[[ -x "${helper}" ]] || { echo "error: release helper is not executable: ${helper}" >&2; exit 1; }

json_value() {
  python3 - "$1" "$2" <<'PY'
import json
import sys

with open(sys.argv[1], "r", encoding="utf-8") as handle:
    value = json.load(handle)
for part in sys.argv[2].split("."):
    value = value.get(part) if isinstance(value, dict) else None
print("" if value is None else value)
PY
}

select_release() {
  python3 -c '
import json
import re
import sys

with open(sys.argv[1], "r", encoding="utf-8") as handle:
    data = json.load(handle)
explicit_version, bump, requested_scheme = sys.argv[2], sys.argv[3], sys.argv[4]
info = data.get("version_info") or {}
effective_scheme = requested_scheme or info.get("scheme_guess") or "none"

def semver_bump(latest):
    if not latest:
        return "v1.0.0"
    match = re.match(r"^(v?)(\d+)\.(\d+)\.(\d+)(?:[-+].*)?$", latest)
    if not match:
        raise SystemExit(f"latest SemVer tag is invalid: {latest}")
    prefix, major, minor, patch = match.groups()
    major, minor, patch = int(major), int(minor), int(patch)
    if bump == "major":
        major, minor, patch = major + 1, 0, 0
    elif bump == "minor":
        minor, patch = minor + 1, 0
    else:
        patch += 1
    selected_prefix = prefix or "v"
    return f"{selected_prefix}{major}.{minor}.{patch}"

if explicit_version:
    selected = explicit_version
elif effective_scheme in ("semver", "mixed"):
    selected = semver_bump(info.get("latest_semver_tag") or "")
elif effective_scheme == "calver":
    candidate = info.get("calver_candidate") or {}
    if candidate.get("ok") is not True:
        raise SystemExit("CalVer candidate is not valid for this release timestamp")
    selected = info.get("next_calver") or ""
else:
    selected = semver_bump("")

if effective_scheme == "calver":
    boundary = info.get("latest_calver_tag") or ""
elif effective_scheme in ("semver", "mixed"):
    boundary = info.get("latest_semver_tag") or ""
else:
    boundary = info.get("latest_tag") or ""

if not selected:
    raise SystemExit("release version selection returned an empty version")
print(selected)
print(boundary)
print(effective_scheme)
' "$1" "${version}" "${bump}" "${scheme}"
}

preflight_json="$(mktemp)"
notes_file="$(mktemp)"
cleanup() {
  rm -f "${preflight_json}" "${notes_file}"
}
trap cleanup EXIT

release_timestamp="$(date +%Y-%m-%dT%H:%M:%S%z)"
release_date="${release_timestamp%%T*}"

run_local_preflight() {
  if ! "${helper}" preflight --local --release-timestamp "${release_timestamp}" >"${preflight_json}"; then
    cat "${preflight_json}"
    echo "error: local release preflight failed" >&2
    exit 1
  fi
  cat "${preflight_json}"
}

echo "==> [release] Checking local release state"
run_local_preflight
default_branch="$(json_value "${preflight_json}" "default_branch")"
source_commit="$(git rev-parse HEAD)"

head_release_tags=()
head_tag_output="$(git tag --points-at HEAD --list 'v*' --sort=-version:refname)"
while IFS= read -r head_tag; do
  [[ -n "${head_tag}" ]] || continue
  if [[ "${head_tag}" =~ ^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$ ]]; then
    head_release_tags+=("${head_tag}")
  fi
done <<<"${head_tag_output}"
if [[ "${#head_release_tags[@]}" -gt 1 ]]; then
  echo "error: multiple stable release tags point at HEAD: ${head_release_tags[*]}" >&2
  exit 1
fi
if [[ "${#head_release_tags[@]}" -eq 1 ]]; then
  prepared_version="${head_release_tags[0]}"
  if [[ -n "${version}" && "${version}" != "${prepared_version}" ]]; then
    echo "error: HEAD is already release ${prepared_version}; refusing to prepare requested ${version}" >&2
    exit 1
  fi
  [[ "$(git cat-file -t "refs/tags/${prepared_version}")" == "tag" ]] || {
    echo "error: release tag ${prepared_version} must be annotated" >&2
    exit 1
  }
  release_parent_line="$(git rev-list --parents -n 1 HEAD)"
  read -r -a release_parent_values <<<"${release_parent_line}"
  [[ "${#release_parent_values[@]}" -eq 2 ]] || {
    echo "error: prepared release commit must have exactly one source parent" >&2
    exit 1
  }
  prepared_source_commit="${release_parent_values[1]}"
  prepared_changed_files="$(git diff-tree --no-commit-id --name-only -r HEAD)"
  [[ "${prepared_changed_files}" == "CHANGELOG.md" ]] || {
    echo "error: prepared release commit must contain only CHANGELOG.md" >&2
    exit 1
  }
  if ! prepared_artifact_verification="$("${helper}" verify-release-artifact 2>&1)"; then
    echo "error: HEAD is already ${prepared_version}, but its prepared release artifact is missing or invalid" >&2
    printf '%s\n' "${prepared_artifact_verification}" >&2
    exit 1
  fi
  prepared_manifest_path="$(git rev-parse --git-path mprlab-release)/manifest.json"
  python3 - "${prepared_manifest_path}" "${prepared_version}" "${source_commit}" "${prepared_source_commit}" "${default_branch}" <<'PY_PREPARED_RELEASE'
import datetime as dt
import json
import pathlib
import re
import sys

manifest = json.load(open(sys.argv[1], encoding="utf-8"))
expected = {
    "schema_version": 2,
    "artifact_kind": "mprlab.release",
    "version": sys.argv[2],
    "release_commit": sys.argv[3],
    "source_commit": sys.argv[4],
    "default_branch": sys.argv[5],
}
for field, value in expected.items():
    if manifest.get(field) != value:
        raise SystemExit(f"prepared release manifest {field} is {manifest.get(field)!r}, expected {value!r}")
timestamp = manifest.get("release_timestamp")
if not isinstance(timestamp, str):
    raise SystemExit("prepared release manifest has no release_timestamp")
parsed_timestamp = dt.datetime.fromisoformat(timestamp.replace("Z", "+00:00"))
if parsed_timestamp.tzinfo is None or parsed_timestamp.utcoffset() is None:
    raise SystemExit("prepared release manifest release_timestamp must include a timezone")
paths = [entry.get("path") for entry in manifest.get("payloads", []) if isinstance(entry, dict)]
react_native = [path for path in paths if isinstance(path, str) and re.fullmatch(r"payloads/release-assets/loopaware-react-native-[0-9]+\.[0-9]+\.[0-9]+\.tgz", path)]
if len(react_native) != 1:
    raise SystemExit("prepared release manifest must contain exactly one canonical React Native package")
expected_paths = {
    "payloads/containers/loopaware/container.json",
    "payloads/containers/loopaware/linux-amd64.tar",
    "payloads/release-assets/loopaware-android-mapping.txt",
    "payloads/release-assets/loopaware-android.aab",
    "payloads/release-assets/loopaware-android.json",
    "payloads/release-assets/loopaware-ios.ipa",
    "payloads/release-assets/loopaware-ios.json",
    "payloads/release-assets/pages.tar.gz",
    react_native[0],
}
if len(paths) != 9 or set(paths) != expected_paths:
    raise SystemExit("prepared release manifest does not contain the exact canonical nine-file payload set")
PY_PREPARED_RELEASE
  echo "release_already_prepared=true"
  echo "release_scope=local"
  echo "default_branch=${default_branch}"
  echo "version=${prepared_version}"
  echo "source_commit=${prepared_source_commit}"
  echo "release_commit=${source_commit}"
  echo "Release ${prepared_version} is already prepared with its exact payloads; run make publish-dry-run."
  exit 0
fi

selection="$(select_release "${preflight_json}")"
next_version="$(sed -n '1p' <<<"${selection}")"
boundary_tag="$(sed -n '2p' <<<"${selection}")"
effective_scheme="$(sed -n '3p' <<<"${selection}")"
[[ "${next_version}" =~ ^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$ ]] || {
  echo "error: LoopAware releases must use deployable stable vMAJOR.MINOR.PATCH versions (got: ${next_version})" >&2
  exit 1
}
selected_version="${next_version}"
selected_boundary_tag="${boundary_tag}"
selected_scheme="${effective_scheme}"

if [[ "${dry_run}" == "true" ]]; then
  echo "release_dry_run=true"
  echo "release_scope=local"
  echo "default_branch=${default_branch}"
  echo "version_scheme=${effective_scheme}"
  echo "next_version=${next_version}"
  echo "changelog_boundary=${boundary_tag:-<none>}"
  exit 0
fi

echo "==> [release] Running make ci"
make ci

echo "==> [release] Rechecking local state after CI"
run_local_preflight
[[ "$(git rev-parse HEAD)" == "${source_commit}" ]] || { echo "error: HEAD changed while make ci was running" >&2; exit 1; }
selection="$(select_release "${preflight_json}")"
next_version="$(sed -n '1p' <<<"${selection}")"
boundary_tag="$(sed -n '2p' <<<"${selection}")"
effective_scheme="$(sed -n '3p' <<<"${selection}")"
[[ "${next_version}" == "${selected_version}" && "${boundary_tag}" == "${selected_boundary_tag}" && "${effective_scheme}" == "${selected_scheme}" ]] || {
  echo "error: release version selection changed while make ci was running" >&2
  exit 1
}

"${helper}" initialize-release-artifact \
  --version "${next_version}" \
  --source-commit "${source_commit}" \
  --release-timestamp "${release_timestamp}"
artifact_dir="$(git rev-parse --git-path mprlab-release)"
if [[ "${artifact_dir}" != /* ]]; then
  artifact_dir="${repo_root}/${artifact_dir}"
fi

read -r -a artifact_target_list <<<"${artifact_targets}"
echo "==> [release] Preparing local artifacts: ${artifact_targets}"
make --no-print-directory \
  RELEASE_VERSION="${next_version}" \
  RELEASE_SOURCE_COMMIT="${source_commit}" \
  RELEASE_TIMESTAMP="${release_timestamp}" \
  MOBILE_RELEASE_TIMESTAMP="${release_timestamp}" \
  RELEASE_ARTIFACT_DIR="${artifact_dir}" \
  "${artifact_target_list[@]}"
"$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/verify_staged_artifacts.py" \
  --artifact-dir "${artifact_dir}" \
  --repo-root "${repo_root}" \
  --version "${next_version}" \
  --source-commit "${source_commit}"
echo "==> [release] Rechecking local state after artifact preparation"
run_local_preflight
[[ "$(git rev-parse HEAD)" == "${source_commit}" ]] || { echo "error: HEAD changed while preparing release artifacts" >&2; exit 1; }
post_artifact_selection="$(select_release "${preflight_json}")"
[[ "$(sed -n '1p' <<<"${post_artifact_selection}")" == "${selected_version}" ]] || { echo "error: release version changed while preparing artifacts" >&2; exit 1; }
[[ "$(sed -n '2p' <<<"${post_artifact_selection}")" == "${selected_boundary_tag}" ]] || { echo "error: changelog boundary changed while preparing artifacts" >&2; exit 1; }
[[ "$(sed -n '3p' <<<"${post_artifact_selection}")" == "${selected_scheme}" ]] || { echo "error: release version scheme changed while preparing artifacts" >&2; exit 1; }

echo "==> [release] Preparing ${next_version} from local Git history"
notes_args=(generate-notes --version "${next_version}" --release-date "${release_date}")
if [[ -n "${boundary_tag}" ]]; then
  notes_args+=(--since-tag "${boundary_tag}")
fi
"${helper}" "${notes_args[@]}" | tee "${notes_file}"
"${helper}" insert-changelog --notes-file "${notes_file}"

git add CHANGELOG.md
if git diff --cached --quiet -- CHANGELOG.md; then
  echo "error: CHANGELOG.md has no staged release changes" >&2
  exit 1
fi
staged_files="$(git diff --cached --name-only)"
if [[ "${staged_files}" != "CHANGELOG.md" ]]; then
  echo "error: release commit may contain only CHANGELOG.md" >&2
  printf '%s\n' "${staged_files}" >&2
  exit 1
fi

git commit --no-verify --no-gpg-sign -m "Release ${next_version}"
release_commit="$(git rev-parse HEAD)"
git tag --no-sign -a "${next_version}" -m "Release ${next_version}" "${release_commit}"
"${helper}" write-release-artifact \
  --version "${next_version}" \
  --source-commit "${source_commit}" \
  --release-commit "${release_commit}" \
  --notes-file "${notes_file}" \
  --default-branch "${default_branch}" \
  --release-timestamp "${release_timestamp}"

echo "Prepared ${next_version} at ${release_commit}. Run make publish to publish it."
